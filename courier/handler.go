package courier

import (
	"sync"

	"github.com/asdine/storm/v3"
	"github.com/icodezjb/fabric-study/courier/client"
	"github.com/icodezjb/fabric-study/log"
)

type Handler struct {
	fabCli  client.FabricClient
	blkSync *BlockSync
	rootDB  *storm.DB
	txMgr   *TxManager
	server  *Server

	taskWg sync.WaitGroup

	stopCh chan struct{}
}

func New(cfg *client.Config) (*Handler, error) {
	fabCli := client.New(cfg)

	rootDB, err := OpenStormDB("rootdb")
	if err != nil {
		return nil, err
	}

	store, err := NewStore(rootDB)
	if err != nil {
		return nil, err
	}

	txMrg := NewTxManager(store)
	h := &Handler{
		fabCli:  fabCli,
		blkSync: NewBlockSync(fabCli, txMrg),
		rootDB:  rootDB,
		txMgr:   txMrg,
		stopCh:  make(chan struct{}),
	}

	h.server = NewServer("8080", h)

	return h, nil
}

func (h *Handler) Start() {
	h.txMgr.Start()
	h.blkSync.Start()

	h.processReq()

	h.server.Start()

}

func (h *Handler) Stop() {
	h.blkSync.Stop()

	close(h.stopCh)
	h.taskWg.Wait()

	h.fabCli.Close()
	h.txMgr.Stop()

	h.rootDB.Close()
}

func (h *Handler) RecvMsg(req Request) {
	h.taskWg.Add(1)
	go func() {
		defer h.taskWg.Done()

		h.txMgr.executed.mu.Lock()
		h.txMgr.executed.prq.Push(req, -req.Sequence)
		h.txMgr.executed.mu.Unlock()

		select {
		case h.txMgr.executed.process <- struct{}{}:
		case <-h.stopCh:
			return
		}
	}()
}

func (h *Handler) processReq() {
	log.Info("[ProcessReq] started")
	h.taskWg.Add(1)
	go func() {
		defer h.taskWg.Done()

		for {
			select {
			case <-h.txMgr.executed.process:
				var executed = make([]Request, 0)

				h.txMgr.executed.mu.Lock()
				for !h.txMgr.executed.prq.Empty() {
					item, _ := h.txMgr.executed.prq.Pop()
					req := item.(Request)
					executed = append(executed, req)
				}
				h.txMgr.executed.mu.Unlock()

				if err := h.txMgr.HandleReceipts(executed); err != nil {
					log.Warn("[TxManager] HandleReceipt err: %v", err)

					for _, req := range executed {
						h.txMgr.executed.prq.Push(req, -req.Sequence)
					}
				}

				h.taskWg.Add(1)
				go func() {
					h.taskWg.Done()

					for _, req := range executed {
						_, err := h.fabCli.InvokeChainCode("commit", []string{req.CrossID, req.Receipt})
						if err != nil {
							log.Warn("[ProcessReq] InvokeChainCode err: %v", err)
						}
					}
				}()

			case <-h.stopCh:
				return
			}
		}

		log.Info("[ProcessReq] stopped")
	}()

}
