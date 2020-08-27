package courier

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/icodezjb/fabric-study/courier/client"
	"github.com/icodezjb/fabric-study/log"
)

const blockInterval = 2 * time.Second

type blockSync struct {
	blockNum uint64
	client   *client.Client
	wg       sync.WaitGroup
	stopCh   chan struct{}
}

func NewBlockSync(c *client.Client) *blockSync {
	s := &blockSync{
		// skip genesis
		blockNum: 1,
		client:   c,
		stopCh:   make(chan struct{}),
	}

	return s
}

func (s *blockSync) Start() {
	s.wg.Add(1)
	go s.syncBlock()

	log.Debug("blockSync start")
}

func (s *blockSync) Stop() {
	log.Debug("blockSync stopping")
	close(s.stopCh)
	s.wg.Wait()
	log.Debug("blockSync stoped")
}

func (s *blockSync) syncBlock() {
	defer s.wg.Done()

	blockTimer := time.NewTimer(0)
	defer blockTimer.Stop()

	apply := func(err error) {
		switch {
		case strings.Contains(err.Error(), "error Entry not found in index"):
			blockTimer.Reset(blockInterval)
		case strings.Contains(err.Error(), "ignore"):
			log.Debug("sync %v", err)
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
			block, err := s.client.QueryBlockByNum(s.blockNum)
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
