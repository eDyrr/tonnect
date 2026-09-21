package tonnect

import (
	"fmt"
	"sync"
)

type Store interface {
	Save(o *order) error
	Get(id string) (*order, error)
	Pending() ([]*order, error)
	MarkPaid(id string, txHash string) error
}

type MemStore struct {
	mu   sync.Mutex
	data map[string]*order
}

var (
	_ Store = (*MemStore)(nil)
	_ Store = (*PgStore)(nil)
)

func NewMemStore() *MemStore {
	return &MemStore{data: make(map[string]*order)}
}

func (s *MemStore) Save(o *order) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[o.ID] = o
	return nil
}

func (s *MemStore) Get(id string) (*order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.data[id]
	if !ok {
		return nil, fmt.Errorf("order %s not found", id)
	}
	return o, nil
}

func (s *MemStore) Pending() ([]*order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*order
	for _, o := range s.data {
		if o.Status == Created {
			out = append(out, o)
		}
	}
	return out, nil
}

func (s *MemStore) MarkPaid(id string, txHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.data[id]
	if !ok {
		return fmt.Errorf("order %s not found", id)
	}
	if o.Status == Paid {
		return nil
	}
	o.Status = Paid
	o.PaidTxHash = txHash
	return nil
}
