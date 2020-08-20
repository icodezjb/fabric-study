package client

import (
	"log"
	"net/http"
	"strings"

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/ccpackager/gopackager"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/pkg/errors"
)

func (c *Client) InstallChainCode(verion, peer string) error {
	targetPeers := resmgmt.WithTargetEndpoints(peer)

	// pack the chaincode
	chaincodePack, err := gopackager.NewCCPackage(c.ChainCodePath, c.ChainCodeGoPath)
	if err != nil {
		return errors.WithMessage(err, "pack chaincode error")
	}

	// installing chaincode request
	req := resmgmt.InstallCCRequest{
		Name:    c.ChainCodeID,
		Path:    c.ChainCodePath,
		Version: verion,
		Package: chaincodePack,
	}

	resps, err := c.rc.InstallCC(req, targetPeers)
	if err != nil {
		return errors.WithMessage(err, "install chaincode err")
	}

	// check other errors
	var errs []error
	for _, resp := range resps {
		if resp.Info == "already installed" {
			log.Printf("chaincode %s-%s already installed on peer: %s\n", c.ChainCodeID, verion, peer)
			return nil
		}

		if resp.Status != http.StatusOK {
			errs = append(errs, errors.New(resp.Info))
		}
	}

	if len(errs) > 0 {
		return errors.WithMessage(errs[0], "install chaincode first error")
	}

	return nil
}

func (c *Client) genPolicy(policy string) (*common.SignaturePolicyEnvelope, error) {
	if policy == "ANY" {
		return cauthdsl.SignedByAnyMember([]string{c.OrgName}), nil
	}
	return cauthdsl.FromString(policy)
}

func (c *Client) InstantiateChainCode(version, peer string) (txid fab.TransactionID, err error) {

	// endorser policy
	Org1OrOrg2 := "OR('Org1MSP.member','Org2MSP.member')"
	chaincodePolicy, err := c.genPolicy(Org1OrOrg2)
	if err != nil {
		return "", errors.WithMessage(err, "gen policy error")
	}

	req := resmgmt.InstantiateCCRequest{
		Name:    c.ChainCodeID,
		Path:    c.ChainCodePath,
		Version: version,
		Args:    c.PackArgs([]string{"init", "a", "100", "b", "200"}),
		Policy:  chaincodePolicy,
	}

	targetPeers := resmgmt.WithTargetEndpoints(peer)
	resp, err := c.rc.InstantiateCC(c.ChainCodeID, req, targetPeers)

	switch {
	case err == nil:
		log.Println("Instantiated chaincode tx:", resp.TransactionID)
		txid = resp.TransactionID
	case strings.Contains(err.Error(), "already exists"):
		err = nil
	default:
		err = errors.WithMessage(err, "instantiate chaincode error")
	}

	return txid, err
}

func (c *Client) InvokeChainCode(peers []string) (fab.TransactionID, error) {
	req := channel.Request{
		ChaincodeID: c.ChainCodeID,
		Fcn:         "invoke",
		Args:        c.PackArgs([]string{"a", "b", "10"}),
	}

	targetPeers := channel.WithTargetEndpoints(peers...)
	resp, err := c.cc.Execute(req, targetPeers)
	if err != nil {
		return "", errors.WithMessage(err, "invoke chaincode error")
	}

	return resp.TransactionID, nil
}

func (c *Client) InvokeChainCodeDelete(peers []string) (fab.TransactionID, error) {
	req := channel.Request{
		ChaincodeID: c.ChainCodeID,
		Fcn:         "delete",
		Args:        c.PackArgs([]string{"c"}),
	}

	targetPeers := channel.WithTargetEndpoints(peers...)
	resp, err := c.cc.Execute(req, targetPeers)
	if err != nil {
		return "", errors.WithMessage(err, "delete chaincode error")
	}

	return resp.TransactionID, nil
}

func (c *Client) QueryChainCode(peer, keys string) error {
	req := channel.Request{
		ChaincodeID: c.ChainCodeID,
		Fcn:         "query",
		Args:        c.PackArgs([]string{keys}),
	}

	targetPeers := channel.WithTargetEndpoints(peer)
	resp, err := c.cc.Execute(req, targetPeers)
	if err != nil {
		return errors.WithMessage(err, "query chaincode error")
	}

	log.Printf("Query chaincode tx response:\ntx: %s\nresult: %v\n\n", resp.TransactionID, string(resp.Payload))

	return nil
}

func (c *Client) UpgradeChainCode(version, peer string) error {
	//endorser policy
	org1AndOrg2 := "AND('Org1MSP.member','Org2MSP.member')"
	chaincodePolicy, err := c.genPolicy(org1AndOrg2)
	if err != nil {
		return errors.WithMessage(err, "gen policy from string")
	}

	req := resmgmt.UpgradeCCRequest{
		Name:    c.ChainCodeID,
		Path:    c.ChainCodePath,
		Version: version,
		Args:    c.PackArgs([]string{"init", "a", "1000", "b", "2000"}),
		Policy:  chaincodePolicy,
	}

	targetPeers := resmgmt.WithTargetEndpoints(peer)
	resp, err := c.rc.UpgradeCC(c.ChainCodeID, req, targetPeers)
	if err != nil {
		return errors.WithMessage(err, "Upgrade chaincode error")
	}

	log.Printf("Upgrade chaincode tx: %s", resp.TransactionID)
	return nil
}
