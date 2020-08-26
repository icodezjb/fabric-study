package client

import (
	"fmt"
	"github.com/icodezjb/fabric-study/log"
	"strings"
	"time"
)

const blockInterval = 2 * time.Second

type blockSync struct {
	blockNum uint64
	client   *Client

	stopCh chan struct{}
}

func NewBlockSync(c *Client) *blockSync {
	s := &blockSync{
		// skip genesis
		blockNum: 1,
		client:   c,
		stopCh:   make(chan struct{}),
	}

	go s.SyncBlock()

	return s
}

func (s *blockSync) Stop() {
	close(s.stopCh)
}

func (s *blockSync) SyncBlock() {
	blockTimer := time.NewTimer(0)
	defer blockTimer.Stop()

	apply := func(err error) {
		switch {
		case strings.Contains(err.Error(), "error Entry not found in index"):
			blockTimer.Reset(blockInterval)
		case strings.Contains(err.Error(), "ignore"):
			s.blockNum++
			blockTimer.Reset(blockInterval)
		default:
			log.Error("SyncBlock err: %v", err)
		}
	}

	for {
		select {
		case <-blockTimer.C:
			log.Debug("sync block #%d", s.blockNum)
			block, err := s.client.lc.QueryBlock(s.blockNum)
			if err != nil {
				apply(err)
				break
			}

			preCrossTxs, err := GetPrepareCrossTxs(block, true)
			if err != nil {
				apply(err)
				break
			}

			for _, tx := range preCrossTxs {
				//TODO
				fmt.Println(tx)
			}

			s.blockNum++
			blockTime := time.Unix(preCrossTxs[0].TimeStamp.Seconds, int64(preCrossTxs[0].TimeStamp.Seconds))
			if interval := time.Since(blockTime); interval > blockInterval {
				//sync new block immediately
				blockTimer.Reset(0)
			} else {
				//sync next block timestamp
				blockTimer.Reset(blockInterval)
			}
		case <-s.stopCh:
			return
		}

	}

}
