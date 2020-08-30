package courier

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/icodezjb/fabric-study/courier/client"
	contract "github.com/icodezjb/fabric-study/courier/contractlib"
	"github.com/icodezjb/fabric-study/log"
)

const blockInterval = 2 * time.Second

type BlockSync struct {
	blockNum     uint64
	filterEvents map[string]struct{}
	client       *client.Client
	wg           sync.WaitGroup
	stopCh       chan struct{}
	preTxsCh     chan []*PrepareCrossTx
}

func NewBlockSync(c *client.Client) *BlockSync {
	s := &BlockSync{
		// skip genesis
		blockNum:     1,
		filterEvents: make(map[string]struct{}),
		client:       c,
		stopCh:       make(chan struct{}),
		preTxsCh:     make(chan []*PrepareCrossTx),
	}
	fmt.Println(c.FilterEvents())
	for _, ev := range c.FilterEvents() {
		s.filterEvents[ev] = struct{}{}
	}

	return s
}

func (s *BlockSync) Start() {
	s.wg.Add(2)
	go s.syncBlock()
	go s.ProcessPreTxs()

	log.Debug("blockSync start")
}

func (s *BlockSync) Stop() {
	log.Debug("blockSync stopping")
	close(s.stopCh)
	s.wg.Wait()
	log.Debug("blockSync stopped")
}

func (s *BlockSync) syncBlock() {
	defer s.wg.Done()

	blockTimer := time.NewTimer(0)
	defer blockTimer.Stop()

	apply := func(err error) {
		switch {
		case strings.Contains(err.Error(), "Entry not found in index"):
			blockTimer.Reset(blockInterval)
		case strings.Contains(err.Error(), "ignore"):
			log.Debug("sync %v", err)
			s.blockNum++
			blockTimer.Reset(blockInterval)
		default:
			log.Error("sync block err: %v", err)
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

			preCrossTxs, err := GetPrepareCrossTxs(block, s.filterEvents)
			if err != nil {
				apply(err)
				break
			}

			s.preTxsCh <- preCrossTxs

			s.blockNum++
			blockTime := time.Unix(preCrossTxs[0].TimeStamp.Seconds, preCrossTxs[0].TimeStamp.Seconds)
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

func (s *BlockSync) ProcessPreTxs() {
	defer s.wg.Done()

	for {
		select {
		case preCrossTxs := <-s.preTxsCh:

			for _, tx := range preCrossTxs {
				//TODO
				var swap contract.Contract
				json.Unmarshal(tx.Payload, &swap)
				fmt.Println(tx)
				fmt.Println(swap)
			}
		case <-s.stopCh:
			return
		}
	}
}
