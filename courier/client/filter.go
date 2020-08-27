package client

import (
	"fmt"
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
			//TODO: log.debug
			//logger.Debugf("chaincode action, the payload action is nil, skipping")
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
	TxID        string
	ChainCodeID string

	ChannelID string
	Number    uint64
	TimeStamp *timestamp.Timestamp

	// hyperledger fabric version 1
	// only supports a single action per transaction
	EventName string
	Payload   []byte
}

func (t *PrepareCrossTx) String() string {
	ts := time.Unix(t.TimeStamp.Seconds, int64(t.TimeStamp.Nanos))
	return fmt.Sprintf("TxID = %s\nChainCodeID = %s\nChannelID = %s\nNumber = %d\nTimeStamp = %s\nEventName = %s\nPayload = %v",
		t.TxID, t.ChainCodeID, t.ChannelID, t.Number, ts, t.EventName, t.Payload)
}

// GetPrepareCrossTxs to collect ENDORSER_TRANSACTION and with event tx, if withEvent set true
func GetPrepareCrossTxs(block *common.Block, withEvent bool) (preCrossTxs []*PrepareCrossTx, err error) {
	txsFltr := utils.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	blockNum := block.Header.Number

	for txIndex, ebytes := range block.Data.Data {
		if txsFltr.Flag(txIndex) != peer.TxValidationCode_VALID {
			continue
		}

		channelHeader, payloadData, err := getChannelHeaderAndData(txIndex, blockNum, ebytes)
		switch {
		case err != nil:
			return nil, err
		case channelHeader == nil:
			continue
		}

		preCrossTx, err := getPrepareCrossTx(channelHeader, blockNum)
		switch {
		case err != nil:
			return nil, err
		case preCrossTx == nil:
			continue
		}

		if err = getTxEvents(preCrossTx, payloadData); err != nil {
			return nil, err
		}

		if withEvent && preCrossTx.Payload == nil {
			continue
		}

		preCrossTxs = append(preCrossTxs, preCrossTx)
	}

	if len(preCrossTxs) == 0 {
		return nil, fmt.Errorf("ignore %d block", blockNum)
	}

	return preCrossTxs, nil
}

func getChannelHeaderAndData(txIndex int, number uint64, ebytes []byte) ([]byte, []byte, error) {
	if ebytes == nil {
		log.Debug("got nil data bytes for tx, txIndex=%d, blockNum=%d", txIndex, number)
		return nil, nil, nil
	}

	env, err := utils.GetEnvelopeFromBlock(ebytes)
	if err != nil {
		log.Error("error GetEnvelopeFromBlock, txIndex=%d, blockNum=%d", txIndex, number)
		return nil, nil, fmt.Errorf("error getting tx from block: %w", err)
	}

	// get the payload from the envelope
	paload, err := utils.GetPayload(env)
	if err != nil {
		log.Error("error GetPayload, txIndex=%d, blockNum=%d", txIndex, number)
		return nil, nil, fmt.Errorf("could not extract payload from envelope: %w", err)
	}

	if paload.Header == nil {
		log.Debug("transaction payload header is nil, %d, block num %d", txIndex, number)
		return nil, nil, nil
	}

	return paload.Header.ChannelHeader, paload.Data, nil
}

func getPrepareCrossTx(channelHeader []byte, blockNum uint64) (*PrepareCrossTx, error) {
	chdr, err := utils.UnmarshalChannelHeader(channelHeader)
	if err != nil {
		return nil, err
	}
	if common.HeaderType(chdr.Type) != common.HeaderType_ENDORSER_TRANSACTION {
		return nil, nil
	}
	ccHdrExt, err := utils.UnmarshalChaincodeHeaderExtension(chdr.Extension)
	if err != nil {
		return nil, err
	}

	pickTx := &PrepareCrossTx{
		ChannelID:   chdr.ChannelId,
		Number:      blockNum,
		TimeStamp:   chdr.Timestamp,
		TxID:        chdr.TxId,
		ChainCodeID: ccHdrExt.ChaincodeId.Name,
	}

	return pickTx, nil
}

func getTxEvents(preCrossTx *PrepareCrossTx, payload []byte) error {
	tx, err := utils.GetTransaction(payload)
	if err != nil {
		return fmt.Errorf("error unmarshal transaction payload for block event: %w", err)
	}

	actionsData, err := transactionActions(tx.Actions).toFilteredActions()
	if err != nil {
		//TODO:log.err
		return err
	}

	// hyperledger fabric version 1
	// only supports a single action per transaction
	if actionsData.TransactionActions.ChaincodeActions != nil {
		ccEvent := actionsData.TransactionActions.ChaincodeActions[0]

		preCrossTx.EventName = ccEvent.ChaincodeEvent.EventName
		preCrossTx.Payload = ccEvent.ChaincodeEvent.Payload
	}

	return nil
}
