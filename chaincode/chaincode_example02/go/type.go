package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type CStatus uint8

const (
	// Init is the fabric contract status flag of precommit-transaction
	Init CStatus = 1 << (8 - 1 - iota)
	Pending
	Executed
	Finished
	Completed
	NoReceipt
)

func (c CStatus) String() string {
	switch c {
	case Init:
		return "Init"
	case Pending:
		return "Pending"
	case Executed:
		return "Executed"
	case Finished:
		return "Finished"
	case Completed:
		return "Completed"
	case NoReceipt:
		return "NoReceipt"
	default:
		return "UnSupport"
	}
}

func ParseCStatus(c string) (CStatus, error) {
	switch c {
	case "Init":
		return Init, nil
	case "Pending":
		return Pending, nil
	case "Executed":
		return Executed, nil
	case "Finished":
		return Finished, nil
	case "Completed":
		return Completed, nil
	case "NoReceipt":
		return NoReceipt, nil
	}

	var status CStatus
	return status, fmt.Errorf("not a valid cstatus flag: %s", c)
}

func (c CStatus) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c *CStatus) UnmarshalText(text []byte) error {
	status, err := ParseCStatus(string(text))
	if err != nil {
		return err
	}

	*c = status
	return nil
}

type Contract struct {
	IContract
}

func (c *Contract) UnmarshalJSON(bytes []byte) (err error) {
	var objMap map[string]*json.RawMessage
	err = json.Unmarshal(bytes, &objMap)
	if err != nil {
		return err
	}

	c.IContract, err = RebuildIContract(*objMap["IContract"])

	return err
}

func RebuildIContract(bytes json.RawMessage) (c IContract, err error) {
	var contractMap map[string]*json.RawMessage
	err = json.Unmarshal(bytes, &contractMap)
	if err != nil {
		return nil, err
	}

	var typ string
	err = json.Unmarshal(*contractMap["status"], &typ)
	if err != nil {
		return nil, err
	}

	switch typ {
	case "Init":
		fallthrough
	case "Pending":
		fallthrough
	case "Executed":
		fallthrough
	case "Completed":
		var pc PrecommitContract
		err = json.Unmarshal(bytes, &pc)
		c = &pc
	case "Finished":
		var cc CommitContract
		err = json.Unmarshal(bytes, &cc)
		c = &cc
	default:
		return nil, fmt.Errorf("unsupport contract type: %s", typ)
	}

	return c, nil
}

type IContract interface {
	GetContractID() string
	GetStatus() CStatus
	GetCoreInfo() *ContractCore
	UpdateStatus(CStatus)
}

type ContractCore struct {
	Address     string   `json:"address"`
	Value       string   `json:"value"`
	Description string   `json:"description"`
	Owner       string   `json:"owner"`
	ToCallFunc  string   `json:"to_call"`
	Args        []string `json:"args"`
	Creator     string   `json:"creator"`
}

func (core *ContractCore) genContractID(txid string) (string, error) {
	rawData, err := json.Marshal(core)
	if err != nil {
		return "", err
	}

	var hash [32]byte

	h := sha256.New()
	h.Write(rawData)
	h.Write([]byte(txid))
	h.Sum(hash[:0])

	return hex.EncodeToString(hash[:]), nil
}

type PrecommitContract struct {
	Status     CStatus `json:"status" storm:"index"`
	ContractID string  `json:"contract_id"`
	Receipt    string  `json:"receipt" storm:"index"`
	ContractCore
}

func (c *PrecommitContract) GetCoreInfo() *ContractCore {
	return &c.ContractCore
}

func (c *PrecommitContract) GetContractID() string {
	return c.ContractID
}

func (c *PrecommitContract) GetStatus() CStatus {
	return c.Status
}

func (c *PrecommitContract) UpdateStatus(s CStatus) {
	c.Status = s
}

func (c *PrecommitContract) UpdateReceipt(receipt string) {
	c.Receipt = receipt
}

type CommitContract struct {
	Status     CStatus `json:"status" storm:"index"`
	ContractID string  `json:"contract_id"`
}

func (c *CommitContract) GetContractID() string {
	return c.ContractID
}

func (c *CommitContract) GetStatus() CStatus {
	return c.Status
}

func (c *CommitContract) UpdateStatus(s CStatus) {
	c.Status = s
}

func (c *CommitContract) GetCoreInfo() *ContractCore {
	return nil
}
