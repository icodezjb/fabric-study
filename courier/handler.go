package courier

import (
	"github.com/asdine/storm/v3"
	"github.com/icodezjb/fabric-study/courier/client"
)

type Handler struct {
	fabCli  *client.Client
	blkSync *BlockSync
	rootDB  *storm.DB
	txMgr   *TxManager
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
	return &Handler{
		fabCli:  fabCli,
		blkSync: NewBlockSync(fabCli, txMrg),
		rootDB:  rootDB,
		txMgr:   txMrg,
	}, nil
}

func (h *Handler) Start() {
	h.txMgr.Start()
	h.blkSync.Start()
}

func (h *Handler) Stop() {
	h.blkSync.Stop()
	h.fabCli.Close()
	h.txMgr.Stop()

	h.rootDB.Close()
}
