package courier

import (
	"fmt"
	"strings"
	"time"

	"github.com/icodezjb/fabric-study/courier/utils"
	"github.com/icodezjb/fabric-study/log"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
)

// transactionActions aliasing for peer.TransactionAction pointers slice
type transactionActions []*peer.TransactionAction

func (ta transactionActions) toFilteredActions() (*peer.FilteredTransaction_TransactionActions, error) {
	transactionActions := &peer.FilteredTransactionActions{}
	for _, action := range ta {
		chaincodeActionPayload, err := utils.GetChaincodeActionPayload(action.Payload)
		if err != nil {
			return nil, fmt.Errorf("error unmarshal transaction action payload for block event: %w", err)
		}

		if chaincodeActionPayload.Action == nil {
			log.Debug("chaincode action, the payload action is nil, skipping")
			continue
		}
		propRespPayload, err := utils.GetProposalResponsePayload(chaincodeActionPayload.Action.ProposalResponsePayload)
		if err != nil {
			return nil, fmt.Errorf("error unmarshal proposal response payload for block event: %w", err)
		}

		caPayload, err := utils.GetChaincodeAction(propRespPayload.Extension)
		if err != nil {
			return nil, fmt.Errorf("error unmarshal chaincode action for block event: %w", err)
		}

		ccEvent, err := utils.GetChaincodeEvents(caPayload.Events)
		if err != nil {
			return nil, fmt.Errorf("error unmarshal chaincode event for block event: %w", err)
		}

		if ccEvent.GetChaincodeId() != "" {
			filteredAction := &peer.FilteredChaincodeAction{
				ChaincodeEvent: &peer.ChaincodeEvent{
					TxId:        ccEvent.TxId,
					ChaincodeId: ccEvent.ChaincodeId,
					EventName:   ccEvent.EventName,
					Payload:     ccEvent.Payload,
				},
			}
			transactionActions.ChaincodeActions = append(transactionActions.ChaincodeActions, filteredAction)
		}
	}
	return &peer.FilteredTransaction_TransactionActions{
		TransactionActions: transactionActions,
	}, nil
}

type PrepareCrossTx struct {
	//Block->Header->Number
	BlockNumber uint64

	//Block->Data->Data(Envelope[x])->Payload->Header->ChannelHeader->TxId
	TxID string
	//Block->Data->Data(Envelope[x])->Payload->Header->ChannelHeader->Timestamp
	TimeStamp *timestamp.Timestamp

	// hyperledger fabric version 1
	// only supports a single action per transaction
	// Block->Data->Data(Envelope[x])->Payload->Data->Transaction->Action[0]->ChainCodeAction[0]->ChaincodeEvent->EventName
	EventName string
	// Block->Data->Data(Envelope[x])->Payload->Data->Transaction->Action[0]->ChainCodeAction[0]->ChaincodeEvent->Payload
	Payload []byte
}

func (t *PrepareCrossTx) String() string {
	ts := time.Unix(t.TimeStamp.Seconds, int64(t.TimeStamp.Nanos))
	return fmt.Sprintf("TxID = %s\nNumber = %d\nTimeStamp = %s\nEventName = %s\nPayload = %v",
		t.TxID, t.BlockNumber, ts, t.EventName, string(t.Payload))
}

// GetPrepareCrossTxs to collect ENDORSER_TRANSACTION and with event tx, if withEvent set true
func GetPrepareCrossTxs(block *common.Block, filterMap map[string]struct{}) (preCrossTxs []*PrepareCrossTx, err error) {
	txsFltr := utils.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	blockNum := block.Header.Number

	for txIndex, ebytes := range block.Data.Data {
		if txsFltr.Flag(txIndex) != peer.TxValidationCode_VALID {
			continue
		}

		headerData, payloadData, err := ParseEnvelopePayload(txIndex, ebytes)
		if err != nil {
			return nil, err
		}

		chdr, err := getChannelHeader(headerData)
		if err != nil && strings.Contains(err.Error(), "HeaderType_ENDORSER_TRANSACTION") {
			continue
		}

		eventName, eventPayload, err := getTxEvents(payloadData)
		if err != nil && strings.Contains(err.Error(), "no chaincode event") {
			continue
		}

		if _, ok := filterMap[eventName]; !ok {
			continue
		}

		if err != nil {
			return nil, err
		}

		preCrossTx := &PrepareCrossTx{
			BlockNumber: blockNum,
			TxID:        chdr.TxId,
			TimeStamp:   chdr.Timestamp,
			EventName:   eventName,
			Payload:     eventPayload,
		}

		preCrossTxs = append(preCrossTxs, preCrossTx)
	}

	if len(preCrossTxs) == 0 {
		return nil, fmt.Errorf("ignore %d block", blockNum)
	}

	return preCrossTxs, nil
}

func ParseEnvelopePayload(txIndex int, ebytes []byte) ([]byte, []byte, error) {
	if ebytes == nil {
		return nil, nil, fmt.Errorf("got nil data bytes for tx %d", txIndex)
	}

	env, err := utils.GetEnvelopeFromBlock(ebytes)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting tx from block: %w", err)
	}

	// get the payload from the envelope
	payload, err := utils.GetPayload(env)
	if err != nil {
		return nil, nil, fmt.Errorf("could not extract payload from envelope: %w", err)
	}

	if payload.Header == nil {
		return nil, nil, fmt.Errorf("transaction payload header is nil")
	}

	return payload.Header.ChannelHeader, payload.Data, nil
}

func getChannelHeader(channelHeader []byte) (*common.ChannelHeader, error) {
	chdr, err := utils.UnmarshalChannelHeader(channelHeader)
	if err != nil {
		return nil, err
	}
	if common.HeaderType(chdr.Type) != common.HeaderType_ENDORSER_TRANSACTION {
		return nil, fmt.Errorf("not HeaderType_ENDORSER_TRANSACTION")
	}

	return chdr, nil
}

func getTxEvents(payload []byte) (string, []byte, error) {
	tx, err := utils.GetTransaction(payload)
	if err != nil {
		return "", nil, fmt.Errorf("error unmarshal transaction payload for block event: %w", err)
	}

	actionsData, err := transactionActions(tx.Actions).toFilteredActions()
	if err != nil {
		return "", nil, err
	}

	// hyperledger fabric version 1
	// only supports a single action per transaction
	if actionsData.TransactionActions.ChaincodeActions != nil {
		ccEvent := actionsData.TransactionActions.ChaincodeActions[0].ChaincodeEvent
		return ccEvent.EventName, ccEvent.Payload, nil
	}

	return "", nil, fmt.Errorf("no chaincode event")
}
