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
	PK          int64                `storm:"id,increment"`
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
	prq     *prque.Prque
	process chan struct{}
	mu      sync.Mutex
}

type mockSimplechain struct {
	count uint32
}

func (ms *mockSimplechain) Send([]byte) error {
	ms.count++
	log.Info("send to simplechain", "count", ms.count)
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
		pending:  Prqueue{prq: prque.New(nil), process: make(chan struct{}, 4)},
		executed: Prqueue{prq: prque.New(nil), process: make(chan struct{}, 8)},
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
		t.pending.prq.Push(tx, -tx.TimeStamp.Seconds)
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
			t.pending.prq.Push(tx, -tx.TimeStamp.Seconds)
		}
	}
	t.pending.mu.Unlock()

	// store to db
	if err := t.DB.Save(txs); err != nil {
		return err
	}

	// start send
	if t.pending.prq.Size() != 0 {
		t.pending.process <- struct{}{}
	}

	return nil
}

func (t *TxManager) loop() {
	defer t.wg.Done()

	for {
		select {
		case <-t.pending.process:
			var pending = make([]*CrossTx, 0)

			t.pending.mu.Lock()
			for !t.pending.prq.Empty() {
				item, _ := t.pending.prq.Pop()
				tx := item.(*CrossTx)
				pending = append(pending, tx)
			}
			t.pending.mu.Unlock()

			successList := make([]string, 0)
			updaters := make([]func(c *CrossTx), 0)

			for _, tx := range pending {
				raw, err := json.Marshal(tx)
				if err != nil {
					log.Error("[TxManager] marshal tx", "crossID", tx.CrossID, "status", tx.GetStatus(), "err", err)
					continue
				}

				// TODO: batch send, MaxBatchSize = 64
				if err := t.sClient.Send(raw); err != nil {
					log.Error("[TxManager] send tx to out chain", "crossID", tx.CrossID, "status", tx.GetStatus(), "err", err)
					t.pending.prq.Push(tx, -tx.TimeStamp.Seconds)
					continue
				}

				successList = append(successList, tx.CrossID)
				updaters = append(updaters, func(c *CrossTx) {
					c.UpdateStatus(contractlib.Pending)
				})
			}

			go func() {
				if err := t.DB.Updates(successList, updaters); err != nil {
					log.Debug("[TxManager] update Init to Pending", "len(successList)", len(successList), "err", err)
					panic(err)
				}
			}()

			log.Info("[TxManager] update Init to Pending", "len(successList)", len(successList))

		case <-t.stopCh:
			return
		}
	}
}

func (t *TxManager) HandleReceipts(reqs []Request) error {
	var updaters []func(c *CrossTx)
	var ids []string

	for _, req := range reqs {
		ids = append(ids, req.CrossID)
		updaters = append(updaters, func(c *CrossTx) {
			c.UpdateStatus(contractlib.Executed)
			pc, ok := c.IContract.(*contractlib.PrecommitContract)
			if ok {
				pc.UpdateReceipt(req.Receipt)
			}
		})
	}

	log.Debug("[TxManager] handle receipt", "ids", ids)

	return t.DB.Updates(ids, updaters)
}
