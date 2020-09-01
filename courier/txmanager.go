package courier

import (
	"encoding/json"
	"github.com/asdine/storm/v3/q"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/icodezjb/fabric-study/courier/client"
	"github.com/icodezjb/fabric-study/courier/contractlib"
	"github.com/icodezjb/fabric-study/courier/utils/prque"
	"github.com/icodezjb/fabric-study/log"
	"sync"
)

type CrossTx struct {
	contractlib.Contract
	PK          uint64               `storm:"id,increment"`
	CrossID     string               `storm:"unique"`
	TxID        string               `storm:"index"`
	BlockNumber uint64               `storm:"index"`
	TimeStamp   *timestamp.Timestamp `storm:"index"`
}

func (c *CrossTx) UnmarshalJSON(bytes []byte) (err error) {
	var errList []error

	var objMap map[string]*json.RawMessage
	errList = append(errList, json.Unmarshal(bytes, &objMap))
	errList = append(errList, json.Unmarshal(*objMap["PK"], &c.PK))
	errList = append(errList, json.Unmarshal(*objMap["CrossID"], &c.CrossID))
	errList = append(errList, json.Unmarshal(*objMap["TxID"], &c.TxID))
	errList = append(errList, json.Unmarshal(*objMap["BlockNumber"], &c.BlockNumber))
	errList = append(errList, json.Unmarshal(*objMap["TimeStamp"], &c.TimeStamp))

	c.IContract, err = contractlib.RebuildIContract(*objMap["IContract"])
	errList = append(errList, err)

	for _, err := range errList {
		if err != nil {
			return err
		}
	}

	return nil
}

type Prqueue struct {
	pr      *prque.Prque
	process chan struct{}
	mu      sync.Mutex
}

type mockSimplechain struct {
	count uint32
}

func (ms *mockSimplechain) Send([]byte) error {
	ms.count++
	log.Info("send to simplechain: %d", ms.count)
	return nil
}

type TxManager struct {
	DB
	sClient  client.SimpleClient
	wg       sync.WaitGroup
	stopCh   chan struct{}
	pending  Prqueue
	executed Prqueue
}

func NewTxManager(db DB) *TxManager {
	return &TxManager{
		DB:       db,
		stopCh:   make(chan struct{}),
		pending:  Prqueue{pr: prque.New(nil), process: make(chan struct{}, 4)},
		executed: Prqueue{pr: prque.New(nil), process: make(chan struct{}, 4)},
		sClient:  &mockSimplechain{},
	}
}

func (t *TxManager) Start() {
	log.Info("[TxManager] starting")
	t.wg.Add(1)
	go t.loop()

	t.reload()
	log.Info("[TxManager] started")
}

func (t *TxManager) Stop() {
	log.Info("[TxManager] stopping")
	close(t.stopCh)
	t.wg.Wait()
	log.Info("[TxManager] stopped")
}

func (t *TxManager) reload() {
	log.Debug("[TxManager] reloading")
	toPending := t.DB.Query(0, 0, []FieldName{TimestampField}, false, q.Eq(StatusField, contractlib.Init))

	t.pending.mu.Lock()
	for _, tx := range toPending {
		t.pending.pr.Push(tx, -tx.TimeStamp.Seconds)
	}
	t.pending.mu.Unlock()

	t.pending.process <- struct{}{}

	log.Debug("[TxManager] reload completed")
	//TODO: executed queue
}

func (t *TxManager) AddTxs(txs []*CrossTx) error {
	// pick up the precommit contract txs
	t.pending.mu.Lock()
	for _, tx := range txs {
		if tx.Contract.GetStatus() != contractlib.Finished {
			t.pending.pr.Push(tx, -tx.TimeStamp.Seconds)
		}
	}
	t.pending.mu.Unlock()

	// store to db
	if err := t.DB.Save(txs); err != nil {
		return err
	}

	// start send
	t.pending.process <- struct{}{}

	return nil
}

func (t *TxManager) loop() {
	defer t.wg.Done()

	for {
		select {
		case <-t.pending.process:
			var pending = make([]*CrossTx, 0)

			t.pending.mu.Lock()
			for !t.pending.pr.Empty() {
				item, _ := t.pending.pr.Pop()
				tx := item.(*CrossTx)
				pending = append(pending, tx)
			}
			t.pending.mu.Unlock()

			successList := make([]string, 0)
			updaters := make([]func(c *CrossTx), 0)
			for _, tx := range pending {
				raw, err := json.Marshal(tx)
				if err != nil {
					log.Error("[TxManager] SendToSimpleChain crossID=%s, status=%v, err=%v", tx.CrossID, tx.GetStatus(), err)
					continue
				}

				if err := t.sClient.Send(raw); err != nil {
					log.Error("[TxManager] SendToSimpleChain crossID=%s, status=%v, err=%v", tx.CrossID, tx.GetStatus(), err)
					continue
				}

				successList = append(successList, tx.CrossID)
				updaters = append(updaters, func(c *CrossTx) {
					c.UpdateStatus(contractlib.Pending)
				})
			}

			go func() {
				if err := t.DB.Updates(successList, updaters); err != nil {
					panic(err)
					//log.Crit("[TxManager] SendToSimpleChain update pending status, len(successList)=%d, err=%v",
					//	len(successList), err)
				}
			}()

		case <-t.stopCh:
			return
		}
	}
}
