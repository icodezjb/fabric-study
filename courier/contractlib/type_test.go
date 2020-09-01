package contractlib

import (
	"encoding/json"
	"testing"
)

func TestPrecommitContract(t *testing.T) {
	precmmit := PrecommitContract{Status: Init}

	contract := Contract{&precmmit}

	raw, err := json.Marshal(contract)
	if err != nil {
		t.Error(err)
	}

	var contract2 Contract
	err = json.Unmarshal(raw, &contract2)
	if err != nil {
		t.Error(err)
	}

	if _, ok := contract2.IContract.(*PrecommitContract); !ok {
		t.Error("Contract UnmarshalJson failed")
	}
}

func TestCommitedContract(t *testing.T) {
	commit := CommitContract{Status: Finished}

	contract := Contract{&commit}

	raw, err := json.Marshal(contract)
	if err != nil {
		t.Error(err)
	}

	var contract2 Contract
	err = json.Unmarshal(raw, &contract2)
	if err != nil {
		t.Error(err)
	}

	if _, ok := contract2.IContract.(*CommitContract); !ok {
		t.Error("Contract UnmarshalJson failed")
	}
}

type Test struct {
	Name string
	Contract
}

func (t *Test) UnmarshalJSON(bytes []byte) (err error) {
	var errList []error

	var objMap map[string]*json.RawMessage
	errList = append(errList, json.Unmarshal(bytes, &objMap))
	errList = append(errList, json.Unmarshal(*objMap["Name"], &t.Name))

	t.IContract, err = RebuildIContract(*objMap["IContract"])
	errList = append(errList, err)

	for _, err := range errList {
		if err != nil {
			return err
		}
	}

	return nil
}

func TestIncRebuildIContract(t *testing.T) {
	var t1 = Test{
		Name: "test",
		Contract: Contract{&PrecommitContract{
			Status:       Init,
			ContractID:   "test",
			Receipt:      "",
			ContractCore: ContractCore{},
		}},
	}
	raw, err := json.Marshal(t1)
	if err != nil {
		t.Error(err)
	}

	var t2 Test
	if err = json.Unmarshal(raw, &t2); err != nil {
		t.Error(err)
	}

	if t2.Name != t1.Name {

	}
}
