package courier

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/icodezjb/fabric-study/courier/contractlib"
	"github.com/icodezjb/fabric-study/log"

	"github.com/asdine/storm/v3"
	"github.com/asdine/storm/v3/q"
)

type FieldName = string

const (
	PK             FieldName = "PK"
	CrossIdIndex   FieldName = "CrossID"
	StatusField    FieldName = "Status"
	TimestampField FieldName = "Timestamp"
)

type DB interface {
	Save(txList []*CrossTx) error
	Updates(idList []string, updaters []func(c *CrossTx)) error
	One(fieldName string, value interface{}) *CrossTx
	Set(key string, value uint64) error
	Get(key string) uint64
	Query(pageSize int, startPage int, orderBy []FieldName, reverse bool, filter ...q.Matcher) []*CrossTx
}

type Store struct {
	db storm.Node
}

func OpenStormDB(dataDir string) (*storm.DB, error) {
	var workDir = os.TempDir()

	if dataDir != "" {
		workDir = dataDir
	}
	return storm.Open(filepath.Join(workDir, "rootdb"))
}

func NewStore(root *storm.DB) (*Store, error) {
	s := &Store{}
	s.db = root.From("mychannel").WithBatch(true)
	return s, nil
}

func (s *Store) Set(key string, value uint64) error {
	return s.db.Set("config", key, value)
}

func (s *Store) Get(key string) uint64 {
	var value uint64
	if err := s.db.Get("config", key, &value); err != nil {
		return 0
	}

	return value
}

func (s *Store) Save(txList []*CrossTx) error {
	log.Debug("[Store] to save cross txs", "len(txList)", len(txList))

	withTransaction, err := s.db.Begin(true)
	if err != nil {
		return fmt.Errorf("db begin err: %w", err)
	}
	defer withTransaction.Rollback()

	for _, newTx := range txList {
		var oldTx CrossTx
		err = withTransaction.One(CrossIdIndex, newTx.CrossID, &oldTx)

		if err == storm.ErrNotFound {
			log.Debug("[Store] save new cross tx", "crossID", newTx.CrossID, "status", newTx.GetStatus(), "blockNumber", newTx.BlockNumber)
			if err = withTransaction.Save(newTx); err != nil {
				return fmt.Errorf("db save err: %w", err)
			}
		} else if oldTx.IContract == nil {
			log.Warn("[Store] parse old crossTx failed", "crossID", oldTx.CrossID)
		} else if newTx.GetStatus() == contractlib.Finished {
			log.Debug("[Store] receive Finished crossTx ", "crossID", newTx.CrossID, "txId", newTx.TxID)
			// update old status, discard new
			oldTx.UpdateStatus(contractlib.Completed)
			if err = withTransaction.Update(&oldTx); err != nil {
				return fmt.Errorf("db update err: %w", err)
			}
			log.Info("[Store] update Finished to Completed, cross chain transaction completed", "crossID", newTx.CrossID, "txId", newTx.TxID)
		} else {
			log.Warn("[Store] duplicate crossTx", "crossID", newTx.CrossID, "old.status", oldTx.GetStatus(), "new.status", newTx.GetStatus())
			continue
		}
	}

	return withTransaction.Commit()
}

func (s *Store) One(fieldName string, value interface{}) *CrossTx {
	to := CrossTx{}
	if err := s.db.One(fieldName, value, &to); err != nil {
		return nil
	}

	return &to
}

func (s *Store) Updates(idList []string, updaters []func(c *CrossTx)) error {
	if len(idList) != len(updaters) {
		return fmt.Errorf("invalid update params")
	}

	log.Debug("[Store] update list", "idList", idList)

	withTransaction, err := s.db.Begin(true)
	if err != nil {
		return fmt.Errorf("db begin err: %w", err)
	}
	defer withTransaction.Rollback()

	for i, id := range idList {
		var c CrossTx
		if err = withTransaction.One(CrossIdIndex, id, &c); err != nil {
			return fmt.Errorf("db query err: %w", err)
		}

		updaters[i](&c)

		if err = withTransaction.Update(&c); err != nil {
			return fmt.Errorf("db update err: %w", err)
		}
	}

	log.Debug("[Store] update list", "successes", len(idList))

	return withTransaction.Commit()
}

func (s *Store) Query(pageSize int, startPage int, orderBy []FieldName, reverse bool, filter ...q.Matcher) (crossTxs []*CrossTx) {
	if pageSize > 0 && startPage <= 0 {
		return nil
	}

	query := s.db.Select(filter...)
	if len(orderBy) > 0 {
		query.OrderBy(orderBy...)
	}
	if reverse {
		query.Reverse()
	}
	if pageSize > 0 {
		query.Limit(pageSize).Skip(pageSize * (startPage - 1))
	}
	query.Find(&crossTxs)

	return crossTxs
}
