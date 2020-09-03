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
	filterEvents map[string]struct{}
	fClient      client.FabricClient
	wg           sync.WaitGroup
	stopCh       chan struct{}
	safeClose    sync.Once
	preTxsCh     chan []*PrepareCrossTx
	txm          *TxManager
	errCh        chan error

	//for test
	syncTestHook func([]*CrossTx)
}

func NewBlockSync(c client.FabricClient, txm *TxManager) *BlockSync {
	startNum := txm.Get("number")
	if startNum == 0 {
		// skip genesis
		startNum = 1
	}

	s := &BlockSync{
		blockNum:     startNum,
		filterEvents: make(map[string]struct{}),
		fClient:      c,
		stopCh:       make(chan struct{}),
		preTxsCh:     make(chan []*PrepareCrossTx),
		errCh:        make(chan error),
		txm:          txm,
	}

	for _, ev := range c.FilterEvents() {
		switch ev {
		case "precommit":
			s.filterEvents[ev] = struct{}{}
		case "commit":
			s.filterEvents[ev] = struct{}{}
		default:
			log.Crit(fmt.Sprintf("[Syncer] unsupported filter event type: %s", ev))
		}
	}

	return s
}

func (s *BlockSync) Start() {
	s.wg.Add(2)
	go s.syncBlock()
	go s.processPreTxs()

	log.Info("[BlockSync] started")
}

func (s *BlockSync) Stop() {
	log.Info("[BlockSync] stopping")

	s.safeClose.Do(func() {
		close(s.stopCh)
	})

	s.wg.Wait()
	log.Info("[BlockSync] stopped")
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
			log.Debug(fmt.Sprintf("[BlockSync] handle %v", err))
			s.blockNum++
			blockTimer.Reset(blockInterval)
		default:
			log.Error("[BlockSync] sync block", "err", err)
			go s.Stop()
		}
	}

	for {
		select {
		case <-blockTimer.C:
			log.Debug("[BlockSync] sync block", "blockNumber", s.blockNum)
			if err := s.txm.Set("number", s.blockNum); err != nil {
				apply(err)
				break
			}

			block, err := s.fClient.QueryBlockByNum(s.blockNum)
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
		case err := <-s.errCh:
			apply(err)
			break
		case <-s.stopCh:
			return
		}
	}
}

func (s *BlockSync) processPreTxs() {
	defer s.wg.Done()

	for {
		select {
		case preCrossTxs := <-s.preTxsCh:

			var crossTxs = make([]*CrossTx, len(preCrossTxs))
			for i, tx := range preCrossTxs {
				var c contractlib.Contract
				err := json.Unmarshal(tx.Payload, &c)
				if err != nil {
					log.Error("[BlockSync] processPreTxs parse Contract", " event", tx.EventName, "err", err)
					s.errCh <- err
					break
				}

				crossTx := &CrossTx{
					Contract:    c,
					TxID:        tx.TxID,
					BlockNumber: tx.BlockNumber,
					TimeStamp:   tx.TimeStamp,
					CrossID:     c.GetContractID(),
				}

				crossTxs = append(crossTxs[:i], crossTx)
			}

			log.Debug("[BlockSync] processPreTxs", "len(crossTxs)", len(crossTxs))

			if s.syncTestHook != nil {
				s.syncTestHook(crossTxs)
				break
			}

			if err := s.txm.AddCrossTxs(crossTxs); err != nil {
				log.Error("[BlockSync] processPreTxs", "err", err)
				s.errCh <- err
				break
			}
		case <-s.stopCh:
			return
		}
	}
}
