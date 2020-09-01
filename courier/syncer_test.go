package courier

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/icodezjb/fabric-study/courier/client"

	"github.com/asdine/storm/v3/q"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
)

func TestBlockSync(t *testing.T) {
	var expected = [][]string{
		{"53ff97aa06a446bc9d27ad7dc1656fbb7f4e9b1a5d162157beab945788e4136c", "Init"},
		{"53ff97aa06a446bc9d27ad7dc1656fbb7f4e9b1a5d162157beab945788e4136c", "Finished"},
		{"3d688e09b0bfbad4e758529b236857397ff2fe128313af9bf3262fc0c60370b3", "Init"},
		{"3d688e09b0bfbad4e758529b236857397ff2fe128313af9bf3262fc0c60370b3", "Finished"},
		{"57763bd245b062f95f068338d0719c43274305d8eae492de530e68e4eb5f40ae", "Init"},
		{"9e415b4f883784ef4eacf7cca4496365f0915cb81bed82f430e358b982bf2184", "Init"},
	}

	stopCh := make(chan struct{})

	fabCli, err := newTestFabricClient()
	if err != nil {
		t.Fatal(err)
	}

	txm := &TxManager{
		DB: &MockDB{db: map[string]uint64{}},
	}

	blksync := NewBlockSync(fabCli, txm)

	var recvList = []*CrossTx{}

	blksync.syncTestHook = func(txList []*CrossTx) {
		recvList = append(recvList, txList...)
		if len(recvList) == 6 {
			stopCh <- struct{}{}
		}
	}

	blksync.Start()
	<-stopCh

	blksync.Stop()

	for i, tx := range recvList {
		if tx.CrossID != expected[i][0] || fmt.Sprintf("%v", tx.GetStatus()) != expected[i][1] {
			t.Fatalf("expected[%d]: %v, got: %s, %v", i, expected[i], tx.CrossID, tx.GetStatus())
		}
	}
}

func newTestFabricClient() (client.FabricClient, error) {
	mfc := &MockFabricClient{}

	blocks, err := initBlocks()
	if err != nil {
		return nil, err
	}

	mfc.blocks = blocks

	return mfc, nil
}

type MockFabricClient struct {
	blocks []*common.Block
}

func (m *MockFabricClient) QueryBlockByNum(number uint64) (*common.Block, error) {
	if number > 9 {
		return nil, fmt.Errorf("Entry not found in index")
	}
	return m.blocks[number], nil
}

func (m *MockFabricClient) InvokeChainCode(fcn string, args []string) (fab.TransactionID, error) {
	return "", nil
}

func (m *MockFabricClient) FilterEvents() []string {
	return []string{"precommit", "commit"}
}

func (m *MockFabricClient) Close() {

}

type MockDB struct {
	db map[string]uint64
}

func (d *MockDB) Save(txList []*CrossTx) error {
	return nil
}

func (d *MockDB) Updates(idList []string, updaters []func(c *CrossTx)) error {
	return nil
}

func (d *MockDB) One(fieldName string, value interface{}) *CrossTx {
	return nil
}

func (d *MockDB) Set(key string, value uint64) error {
	d.db[key] = value
	return nil
}

func (d *MockDB) Get(key string) uint64 {
	return d.db[key]
}

func (d *MockDB) Query(pageSize int, startPage int, orderBy []FieldName, reverse bool, filter ...q.Matcher) []*CrossTx {
	return nil
}

func initBlocks() (blocks []*common.Block, err error) {
	file, err := os.Open("./test/testdata/blockdata.hex")
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var block common.Block

		data, err := hex.DecodeString(scanner.Text())
		if err != nil {
			return nil, err
		}

		if err = proto.Unmarshal(data, &block); err != nil {
			return nil, err
		}

		blocks = append(blocks, &block)
	}

	return blocks, err
}
