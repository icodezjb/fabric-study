package contractlib

// copy from chaincode/chaincode_example02/go/type.go

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

type IContract interface {
	GetContractID(string) (string, error)
	GetStatus() CStatus
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

// TODO: 待验证:在chaincode执行环境内,不应该使用timestamp,uuid等任何随机源数据,
//  因为多个背书节点从随机源获取不同的数据,造成共识失败
func (core *ContractCore) GetContractID(txid string) (string, error) {
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

type Contract struct {
	Status     CStatus `json:"status"`
	ContractID string  `json:"contract_id"`
	Receipt    string  `json:"receipt"`
	ContractCore
}

func (c *Contract) GetStatus() CStatus {
	return c.Status
}

type CommittedContract struct {
	Staus      CStatus `json:"staus"`
	ContractID string  `json:"contract_id"`
}

func (c *CommittedContract) GetContractID(_ string) (string, error) {
	return c.ContractID, nil
}

func (c *CommittedContract) GetStatus() CStatus {
	return c.Staus
}
