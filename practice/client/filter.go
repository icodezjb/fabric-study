package client

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/core/ledger/util"
)

// transactionActions aliasing for peer.TransactionAction pointers slice
type transactionActions []*peer.TransactionAction

func (ta transactionActions) toFilteredActions() (*peer.FilteredTransaction_TransactionActions, error) {
	transactionActions := &peer.FilteredTransactionActions{}
	for _, action := range ta {
		chaincodeActionPayload, err := GetChaincodeActionPayload(action.Payload)
		if err != nil {
			return nil, fmt.Errorf("error unmarshal transaction action payload for block event: %w", err)
		}

		if chaincodeActionPayload.Action == nil {
			//TODO: log.debug
			//logger.Debugf("chaincode action, the payload action is nil, skipping")
			continue
		}
		propRespPayload, err := GetProposalResponsePayload(chaincodeActionPayload.Action.ProposalResponsePayload)
		if err != nil {
			return nil, fmt.Errorf("error unmarshal proposal response payload for block event: %w", err)
		}

		caPayload, err := GetChaincodeAction(propRespPayload.Extension)
		if err != nil {
			return nil, fmt.Errorf("error unmarshal chaincode action for block event: %w", err)
		}

		ccEvent, err := GetChaincodeEvents(caPayload.Events)
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

type PickEvent struct {
	EventName string
	Payload   []byte
}

type PickTransaction struct {
	ChainCodeID string
	TxID        string
	Events      []*PickEvent
}

type PickBlock struct {
	ChannelID    string
	Number       uint64
	Transactions []*PickTransaction
}

func ToFilteredBlock(block *common.Block, withEvent bool) (*PickBlock, error) {
	pickBlock := &PickBlock{
		Number: block.Header.Number,
	}

	txsFltr := util.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	for txIndex, ebytes := range block.Data.Data {
		var err error

		if txsFltr.Flag(txIndex) != peer.TxValidationCode_VALID {
			continue
		}

		channelHeader, payloadData, err := getChannelHeaderAndData(ebytes)
		switch {
		case err != nil:
			return nil, err
		case channelHeader == nil:
			continue
		}

		cid, pickTx, err := getChannelIDAndPickTx(channelHeader)
		switch {
		case err != nil:
			return nil, err
		case pickTx == nil:
			continue
		}

		if err = setChainCodeEvents(pickTx, payloadData); err != nil {
			return nil, err
		}

		if withEvent && pickTx.Events == nil {
			continue
		}

		pickBlock.ChannelID = cid
		pickBlock.Transactions = append(pickBlock.Transactions, pickTx)
	}

	if len(pickBlock.Transactions) == 0 {
		return nil, fmt.Errorf("ignore %d block", pickBlock.Number)
	}

	return pickBlock, nil
}

func getChannelHeaderAndData(ebytes []byte) ([]byte, []byte, error) {
	if ebytes == nil {
		//TODO:log.debug
		//logger.Debugf("got nil data bytes for tx index %d, "+
		//	"block num %d", txIndex, block.Header.Number)
		return nil, nil, nil
	}

	env, err := GetEnvelopeFromBlock(ebytes)
	if err != nil {
		//TODO:log.error
		//logger.Errorf("error getting tx from block, %s", err)
		return nil, nil, nil
	}

	// get the payload from the envelope
	paload, err := GetPayload(env)
	if err != nil {
		return nil, nil, fmt.Errorf("could not extract payload from envelope: %w", err)
	}

	if paload.Header == nil {
		//TODO: log.debug
		//logger.Debugf("transaction payload header is nil, %d, block num %d",
		//	txIndex, block.Header.Number)
		return nil, nil, nil
	}

	return paload.Header.ChannelHeader, paload.Data, nil
}

func getChannelIDAndPickTx(channelHeader []byte) (string, *PickTransaction, error) {
	chdr, err := UnmarshalChannelHeader(channelHeader)
	if err != nil {
		return "", nil, err
	}
	if common.HeaderType(chdr.Type) != common.HeaderType_ENDORSER_TRANSACTION {
		return "", nil, nil
	}
	ccHdrExt, err := UnmarshalChaincodeHeaderExtension(chdr.Extension)
	if err != nil {
		return "", nil, err
	}

	pickTx := &PickTransaction{
		TxID:        chdr.TxId,
		ChainCodeID: ccHdrExt.ChaincodeId.Name,
	}

	return chdr.ChannelId, pickTx, nil
}

func setChainCodeEvents(pickTx *PickTransaction, payload []byte) error {
	tx, err := GetTransaction(payload)
	if err != nil {
		return fmt.Errorf("error unmarshal transaction payload for block event: %w", err)
	}

	actionsData, err := transactionActions(tx.Actions).toFilteredActions()
	if err != nil {
		//TODO:log.err
		return err
	}

	for _, ccEvent := range actionsData.TransactionActions.ChaincodeActions {
		pickTx.Events = append(pickTx.Events, &PickEvent{
			EventName: ccEvent.ChaincodeEvent.EventName,
			Payload:   ccEvent.ChaincodeEvent.Payload,
		})
	}

	return nil
}

func wrapErrorf(err error, msg string) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", msg, err)
}

// GetChaincodeEvents gets the ChaincodeEvents given chaincode event bytes
func GetChaincodeEvents(eBytes []byte) (*peer.ChaincodeEvent, error) {
	chaincodeEvent := &peer.ChaincodeEvent{}
	err := proto.Unmarshal(eBytes, chaincodeEvent)
	return chaincodeEvent, wrapErrorf(err, "error unmarshaling ChaicnodeEvent")
}

// GetChaincodeAction gets the ChaincodeAction given chaicnode action bytes
func GetChaincodeAction(caBytes []byte) (*peer.ChaincodeAction, error) {
	chaincodeAction := &peer.ChaincodeAction{}
	err := proto.Unmarshal(caBytes, chaincodeAction)
	return chaincodeAction, wrapErrorf(err, "error unmarshaling ChaincodeAction")
}

// GetProposalResponsePayload gets the proposal response payload
func GetProposalResponsePayload(prpBytes []byte) (*peer.ProposalResponsePayload, error) {
	prp := &peer.ProposalResponsePayload{}
	err := proto.Unmarshal(prpBytes, prp)
	return prp, wrapErrorf(err, "error unmarshaling ProposalResponsePayload")
}

// GetChaincodeActionPayload Get ChaincodeActionPayload from bytes
func GetChaincodeActionPayload(capBytes []byte) (*peer.ChaincodeActionPayload, error) {
	c := &peer.ChaincodeActionPayload{}
	err := proto.Unmarshal(capBytes, c)
	return c, wrapErrorf(err, "error unmarshaling ChaincodeActionPayload")
}

// GetEnvelopeFromBlock gets an envelope from a block's Data field.
func GetEnvelopeFromBlock(data []byte) (*common.Envelope, error) {
	// Block always begins with an envelope
	var err error
	env := &common.Envelope{}
	err = proto.Unmarshal(data, env)
	return env, wrapErrorf(err, "error unmarshaling Envelope")
}

// GetPayload Get Payload from Envelope message
func GetPayload(e *common.Envelope) (*common.Payload, error) {
	payload := &common.Payload{}
	err := proto.Unmarshal(e.Payload, payload)
	return payload, wrapErrorf(err, "error unmarshaling Payload")
}

// UnmarshalChannelHeader returns a ChannelHeader from bytes
func UnmarshalChannelHeader(bytes []byte) (*common.ChannelHeader, error) {
	chdr := &common.ChannelHeader{}
	err := proto.Unmarshal(bytes, chdr)
	return chdr, wrapErrorf(err, "error unmarshaling ChannelHeader")
}

// GetTransaction Get Transaction from bytes
func GetTransaction(txBytes []byte) (*peer.Transaction, error) {
	tx := &peer.Transaction{}
	err := proto.Unmarshal(txBytes, tx)
	return tx, wrapErrorf(err, "error unmarshaling Transaction")
}

// UnmarshalChaincodeHeaderExtension unmarshals bytes to a ChaincodeHeaderExtension
func UnmarshalChaincodeHeaderExtension(hdrExtension []byte) (*peer.ChaincodeHeaderExtension, error) {
	chaincodeHdrExt := &peer.ChaincodeHeaderExtension{}
	err := proto.Unmarshal(hdrExtension, chaincodeHdrExt)
	return chaincodeHdrExt, wrapErrorf(err, "error unmarshaling ChaincodeHeaderExtension")
}
