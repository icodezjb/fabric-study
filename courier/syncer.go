package courier

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/icodezjb/fabric-study/courier/client"
	"github.com/icodezjb/fabric-study/courier/contractlib"
	"github.com/icodezjb/fabric-study/log"
)

const blockInterval = 2 * time.Second

type BlockSync struct {
	blockNum     uint64
	filterEvents map[string]WithUnmarshal
	client       *client.Client
	wg           sync.WaitGroup
	stopCh       chan struct{}
	preTxsCh     chan []*PrepareCrossTx
}

type WithUnmarshal func([]byte) (contractlib.IContract, error)

func NewBlockSync(c *client.Client) *BlockSync {
	s := &BlockSync{
		// skip genesis
		blockNum:     1,
		filterEvents: make(map[string]WithUnmarshal),
		client:       c,
		stopCh:       make(chan struct{}),
		preTxsCh:     make(chan []*PrepareCrossTx),
	}

	for _, ev := range c.FilterEvents() {
		switch ev {
		case "precommit":
			s.filterEvents[ev] = func(bytes []byte) (contractlib.IContract, error) {
				contract := &contractlib.Contract{}
				if err := json.Unmarshal(bytes, &contract); err != nil {
					return nil, err
				}
				return contract, nil
			}
		case "commit":
			s.filterEvents[ev] = func(bytes []byte) (contractlib.IContract, error) {
				contract := &contractlib.CommittedContract{}
				if err := json.Unmarshal(bytes, &contract); err != nil {
					return nil, err
				}
				return contract, nil
			}
		default:
			panic(fmt.Sprintf("unsupported filter event type: %s", ev))
		}
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

			preCrossTxs, err := GetPrepareCrossTxs(block, func(eventName string) bool {
				if _, ok := s.filterEvents[eventName]; ok {
					return true
				}
				return false
			})

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

			var crossTxs = make([]CrossTx, len(preCrossTxs))
			for i, tx := range preCrossTxs {
				//TODO
				var crossTx CrossTx
				c, err := s.filterEvents[tx.EventName](tx.Payload)
				if err != nil {
					log.Error("ProcessPreTxs event: %s, err:%v, f=%v", tx.EventName, err, s.filterEvents[tx.EventName])
				}

				crossTx.BlockNumber = tx.BlockNumber
				crossTx.TxID = tx.TxID
				crossTx.TimeStamp = tx.TimeStamp

				crossTx.IContract = c

				crossTxs = append(crossTxs[:i], crossTx)
			}

			log.Info("%v", crossTxs)
		case <-s.stopCh:
			return
		}
	}
}
