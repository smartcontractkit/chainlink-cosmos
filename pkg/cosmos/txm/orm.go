package txm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/smartcontractkit/sqlx"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/db"
)

type DB interface {
	BeginTxx(context.Context, *sql.TxOptions) (*sqlx.Tx, error)
}

type Queryer interface {
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

// ORM manages the data model for cosmos tx management.
type ORM struct {
	chainID string
	db      Queryer
}

// NewORM creates an ORM scoped to chainID.
func NewORM(chainID string, db Queryer) *ORM {
	return &ORM{
		chainID: chainID,
		db:      db,
	}
}

// TODO this is a simplified version of https://github.com/smartcontractkit/chainlink/blob/5c0ac3879bef46b828d460960702c1587f730909/core/services/pg/transaction.go#L88
// we should have a common shared impl instead, maybe in relay/pkg/db
// TODO make generic
func (o *ORM) Transaction(ctx context.Context, fn func(*ORM) error) (err error) {
	db, ok := o.db.(DB)
	if !ok {
		return fn(NewORM(o.chainID, o.db))
	}
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			if err2 := tx.Rollback(); err2 != nil {
				err = errors.Join(err, err2)
			}
			return
		}
		err = tx.Commit()
	}()
	// TODO set timeouts?
	err = fn(NewORM(o.chainID, tx))
	return
}

// InsertMsg inserts a cosmos msg, assumed to be a serialized cosmos ExecuteContractMsg.
func (o *ORM) InsertMsg(ctx context.Context, contractID, typeURL string, msg []byte) (int64, error) {
	var tm adapters.Msg

	err := o.db.GetContext(ctx, &tm, `INSERT INTO cosmos_msgs (contract_id, type, raw, state, cosmos_chain_id, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING *`, contractID, typeURL, msg, db.Unstarted, o.chainID)
	if err != nil {
		return 0, err
	}
	return tm.ID, nil
}

// UpdateMsgsContract updates messages for the given contract.
func (o *ORM) UpdateMsgsContract(ctx context.Context, contractID string, from, to db.State) error {
	_, err := o.db.ExecContext(ctx, `UPDATE cosmos_msgs SET state = $1, updated_at = NOW()
	WHERE cosmos_chain_id = $2 AND contract_id = $3 AND state = $4`, to, o.chainID, contractID, from)
	if err != nil {
		return err
	}
	return nil
}

// GetMsgsState returns the oldest messages with a given state up to limit.
func (o *ORM) GetMsgsState(ctx context.Context, state db.State, limit int64) (adapters.Msgs, error) {
	if limit < 1 {
		return adapters.Msgs{}, errors.New("limit must be greater than 0")
	}
	var msgs adapters.Msgs
	if err := o.db.SelectContext(ctx, &msgs, `SELECT * FROM cosmos_msgs WHERE state = $1 AND cosmos_chain_id = $2 ORDER BY id ASC LIMIT $3`, state, o.chainID, limit); err != nil {
		return nil, err
	}
	return msgs, nil
}

// GetMsgs returns any messages matching ids.
func (o *ORM) GetMsgs(ctx context.Context, ids ...int64) (adapters.Msgs, error) {
	var msgs adapters.Msgs
	if err := o.db.SelectContext(ctx, &msgs, `SELECT * FROM cosmos_msgs WHERE id = ANY($1)`, ids); err != nil {
		return nil, err
	}
	return msgs, nil
}

// UpdateMsgs updates msgs with the given ids.
// Note state transitions are validated at the db level.
func (o *ORM) UpdateMsgs(ctx context.Context, ids []int64, state db.State, txHash *string) error {
	if state == db.Broadcasted && txHash == nil {
		return errors.New("txHash is required when updating to broadcasted")
	}
	var res sql.Result
	var err error
	if state == db.Broadcasted {
		res, err = o.db.ExecContext(ctx, `UPDATE cosmos_msgs SET state = $1, updated_at = NOW(), tx_hash = $2 WHERE id = ANY($3)`, state, *txHash, ids)
	} else {
		res, err = o.db.ExecContext(ctx, `UPDATE cosmos_msgs SET state = $1, updated_at = NOW() WHERE id = ANY($2)`, state, ids)
	}
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if int(count) != len(ids) {
		return fmt.Errorf("expected %d records updated, got %d", len(ids), count)
	}
	return nil
}
