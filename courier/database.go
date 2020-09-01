package courier

import (
	"fmt"
	"github.com/asdine/storm/v3/q"
	"github.com/icodezjb/fabric-study/courier/contractlib"
	"os"
	"path/filepath"

	"github.com/icodezjb/fabric-study/log"

	"github.com/asdine/storm/v3"
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

func OpenStormDB(dbPath string) (*storm.DB, error) {
	if len(dbPath) == 0 {
		return storm.Open(filepath.Join(os.TempDir(), dbPath))
	}
	return storm.Open(dbPath)
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
	log.Debug("[Store] to save %d cross txs", len(txList))

	withTransaction, err := s.db.Begin(true)
	if err != nil {
		return err
	}
	defer withTransaction.Rollback()

	for _, new := range txList {
		var old CrossTx
		err = withTransaction.One(CrossIdIndex, new.CrossID, &old)

		if err == storm.ErrNotFound {
			log.Debug("[Store] crossID = %s, status = %v, blockNumber = %d", new.CrossID, new.GetStatus(), new.BlockNumber)
			if err = withTransaction.Save(new); err != nil {
				return err
			}
		} else if old.IContract == nil {
			log.Warn("[store] parse old crossTx failed, crossID = %s", old.CrossID)
		} else if old.GetStatus() == contractlib.Executed && new.GetStatus() == contractlib.Finished {
			log.Debug("[Store] complete crossTx: crossID = %s, txId = %s", new.CrossID, new.TxID)
			new.UpdateStatus(contractlib.Completed)
			if err = withTransaction.Update(new); err != nil {
				return err
			}
		} else {
			log.Warn("[Store] duplicate crossTx: crossID = %s, old.status = %v, new.status = %v", new.CrossID, old.GetStatus(), new.GetStatus())
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

	log.Debug("[Store] update list: %v", idList)

	withTransaction, err := s.db.Begin(true)
	if err != nil {
		return err
	}
	defer withTransaction.Rollback()

	for i, id := range idList {
		var c CrossTx
		if err = withTransaction.One(CrossIdIndex, id, &c); err != nil {
			return err
		}

		updaters[i](&c)

		if err = withTransaction.Update(&c); err != nil {
			return err
		}
	}

	log.Debug("[Store] update %d cross txs", len(idList))

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
