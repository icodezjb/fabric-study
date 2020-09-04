package courier

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/icodezjb/fabric-study/courier/client"
	"github.com/icodezjb/fabric-study/log"

	"github.com/asdine/storm/v3"
)

type Handler struct {
	blkSync *BlockSync
	rootDB  *storm.DB
	txm     *TxManager
	server  *Server

	taskWg sync.WaitGroup

	stopCh chan struct{}
}

func New(cfg *client.Config) (*Handler, error) {
	fabCli := client.NewFabCli(cfg)

	rootDB, err := OpenStormDB(cfg.DataDir())
	if err != nil {
		return nil, err
	}

	store, err := NewStore(rootDB)
	if err != nil {
		return nil, err
	}

	txm := NewTxManager(fabCli, &client.MockOutChainClient{}, store)
	h := &Handler{
		blkSync: NewBlockSync(fabCli, txm),
		rootDB:  rootDB,
		txm:     txm,
		stopCh:  make(chan struct{}),
	}

	h.server = NewServer(cfg.HTTPEndpoint(), h)

	return h, nil
}

func (h *Handler) Start() {
	h.txm.Start()
	h.blkSync.Start()
	h.server.Start()
}

func (h *Handler) Stop() {
	h.blkSync.Stop()
	h.server.Stop()

	close(h.stopCh)
	h.taskWg.Wait()

	h.txm.Stop()

	h.rootDB.Close()
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	code, msg := http.StatusOK, ""

	switch req.URL.Path {
	case "/v1/receipt":
		if req.Method != "POST" {
			code, msg = http.StatusBadRequest, "support POST request only"
			break
		}

		crossID := req.PostFormValue("crossid")
		receipt := req.PostFormValue("receipt")
		sequence := req.PostFormValue("sequence")

		//TODO check crossID, receipt, sequence
		seq, _ := strconv.Atoi(sequence)

		h.RecvMsg(CrossTxReceipt{crossID, receipt, int64(seq)})
	default:
		code = http.StatusNotFound
		msg = fmt.Sprintf("%s not found\n", req.URL.Path)
	}

	w.WriteHeader(code)
	if _, err := w.Write([]byte(msg)); err != nil {
		log.Error("[Server] serve http", "err", err)
	}
}

func (h *Handler) RecvMsg(ctr CrossTxReceipt) {
	h.taskWg.Add(1)
	go func() {
		defer h.taskWg.Done()

		h.txm.executed.mu.Lock()
		h.txm.executed.prq.Push(ctr, -ctr.Sequence)
		h.txm.executed.mu.Unlock()

		select {
		case h.txm.executed.process <- struct{}{}:
		case <-h.stopCh:
			return
		}
	}()
}
