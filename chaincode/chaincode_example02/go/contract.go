package main

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// TODO: 锁住所有与precommit提交字段中key相同的资源,直到commit完成
func (t *SimpleChaincode) precommit(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	callArgs := strings.Split(args[4], " ")
	core := ContractCore{
		Address:     args[0],
		Value:       args[1],
		Description: args[2],
		ToCallFunc:  args[3],
		Creator:     t.creator(stub),
		Args:        callArgs,
		Owner:       callArgs[0],
	}

	id, err := core.ContractID(stub.GetTxID())
	if err != nil {
		return shim.Error(fmt.Sprintf("get contract id err: %v", err))
	}

	contract := Contract{
		Status:       Init,
		ContractID:   id,
		ContractCore: core,
	}

	rawContract, err := json.Marshal(&contract)
	if err != nil {
		return shim.Error(err.Error())
	}

	// store to ledger
	if err = stub.PutState(id, rawContract); err != nil {
		return shim.Error(err.Error())
	}

	// send event
	if err = stub.SetEvent("precommit", rawContract); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

//commit <contractID, receipt>
// TODO: 处理args[0]=="noreceipt"
func (t *SimpleChaincode) commit(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	contractID := args[0]
	receipt := args[1]

	rawContract, err := stub.GetState(contractID)
	if err != nil {
		shim.Error(fmt.Sprintf("get contract by %s, err: %v", contractID, err))
	} else if rawContract == nil {
		shim.Error(fmt.Sprintf("invalid contractid %s", contractID))
	}

	var contract Contract
	if err = json.Unmarshal(rawContract, &contract); err != nil {
		shim.Error(fmt.Sprintf("parse contract with %s, err: %v", contractID, err))
	}

	if contract.Status == Finished {
		shim.Success([]byte("replicate call commit"))
	}

	if err = t.doCommit(stub, &contract); err != nil {
		shim.Error(fmt.Sprintf("doCommit err: %v", err))
	}

	contract.Status = Finished
	contract.Receipt = receipt

	updateData, err := json.Marshal(contract)
	if err != nil {
		shim.Error(err.Error())
	}

	// store to ledger
	if err = stub.PutState(contractID, updateData); err != nil {
		shim.Error(err.Error())
	}

	// send event
	if err = stub.SetEvent("commit", updateData); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func (t *SimpleChaincode) doCommit(stub shim.ChaincodeStubInterface, c *Contract) error {
	switch c.ToCallFunc {
	case "invoke":
		return t.doInvoke(stub, c.Args)
	default:
		return fmt.Errorf("undefined %s", c.ToCallFunc)
	}
}

func (t *SimpleChaincode) creator(stub shim.ChaincodeStubInterface) string {
	creatorByte, _ := stub.GetCreator()
	certStart := bytes.IndexAny(creatorByte, "-----BEGIN")
	if certStart == -1 {
		shim.Error("No certificate found")
	}
	certText := creatorByte[certStart:]
	bl, _ := pem.Decode(certText)
	if bl == nil {
		shim.Error("Could not decode the PEM structure")
	}
	cert, err := x509.ParseCertificate(bl.Bytes)
	if err != nil {
		shim.Error("ParseCertificate failed")
	}

	return cert.Subject.CommonName
}
