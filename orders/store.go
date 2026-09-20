package tonnect

import (
	"fmt"
	"sync"
)

type Store struct {
	mu   sync.Mutex
	data map[string]*order
}

func NewStore() *Store {
	return &Store{data: make(map[string]*order)}
}

func (s *Store) Save(o *order) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[o.ID] = o
}

func (s *Store) Get(id string) (*order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.data[id]
	if !ok {
		return nil, fmt.Errorf("order %s not found", id)
	}
	return o, nil
}

func (s *Store) Pending() ([]*order, error) {
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

func (s *Store) MarkPaid(id string, txHash string) error {
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
	return nil
}
