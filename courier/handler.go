package courier

import "github.com/icodezjb/fabric-study/courier/client"

type Handler struct {
	fabCli  *client.Client
	blkSync *blockSync
}

func New(cfg *client.Config) *Handler {
	fabCli := client.New(cfg)

	return &Handler{
		fabCli:  fabCli,
		blkSync: NewBlockSync(fabCli),
	}
}

func (h *Handler) Start() {
	h.blkSync.Start()
}

func (h *Handler) Stop() {
	h.blkSync.Stop()
	h.fabCli.Close()
}
