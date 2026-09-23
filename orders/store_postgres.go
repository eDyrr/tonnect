package tonnect

import (
	"database/sql"
	"fmt"

	_ "embed"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed schema.sql
var schema string

type PgStore struct {
	db *sql.DB
}

func NewPgStore(connString string) (*PgStore, error) {
	db, err := sql.Open("pgx", connString)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping failed: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}
	return &PgStore{db: db}, nil
}

func (s *PgStore) Save(o *order) error {
	_, err := s.db.Exec(
		`insert into orders (id, status, reference, recipient, amount)
		 values ($1, $2, $3, $4, $5)`,
		o.ID, o.Status, o.Reference, o.Recipient, o.Amount,
	)
	return err
}

func (s *PgStore) Get(id string) (*order, error) {
	var o order
	err := s.db.QueryRow(
		`select id, status, reference, recipient, amount from orders where id = $1`,
		id,
	).Scan(&o.ID, &o.Status, &o.Reference, &o.Recipient, &o.Amount)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("order %s not found ", id)
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (s *PgStore) Pending() ([]*order, error) {
	rows, err := s.db.Query(
		`select id, status, reference, recipient, amount from orders where status = $1`,
		Created,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*order
	for rows.Next() {
		var o order
		if err := rows.Scan(&o.ID, &o.Status, &o.Reference, &o.Recipient, &o.Amount); err != nil {
			return nil, err
		}
		out = append(out, &o)
	}
	return out, rows.Err()
}

func (s *PgStore) MarkPaid(id string, txHash string) error {
	res, err := s.db.Exec(
		`update orders set status = $1, paid_tx_hash = $2, paid_at = now() where id = $3 and status != $1`,
		Paid, txHash, id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		// nothing updated: missing ID, or already paid
		if _, err := s.Get(id); err != nil {
			return err // missing "not found"
		}
	}
	return err
}

func (s *PgStore) MarkExpired(id string) error {
	res, err := s.db.Exec(
		`update orders set status = $1 where id = $2 and status = $3`,
		Expired, id, Created,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		if _, err := s.Get(id); err != nil {
			return err
		}
	}
	return nil
}
