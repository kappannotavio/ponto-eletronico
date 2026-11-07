package repository

import (
	"sort"
	"sync"

	"github.com/kappannotavio/ponto-eletronico/model"
)

type InMemoryTimeRecordRepository struct {
	mu      sync.RWMutex
	nextID  int64
	records map[int64]model.TimeRecord
}

func NewInMemoryTimeRecordRepository() *InMemoryTimeRecordRepository {
	return &InMemoryTimeRecordRepository{
		nextID:  1,
		records: make(map[int64]model.TimeRecord),
	}
}

func (r *InMemoryTimeRecordRepository) Save(record model.TimeRecord) (model.TimeRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record.ID = r.nextID
	r.nextID++
	r.records[record.ID] = record

	return record, nil
}

func (r *InMemoryTimeRecordRepository) FindByID(id int64) (model.TimeRecord, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, ok := r.records[id]

	return record, ok, nil
}

func (r *InMemoryTimeRecordRepository) Delete(id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.records[id]; !ok {
		return ErrNotFound
	}

	delete(r.records, id)

	return nil
}

// List retorna as batidas ordenadas da mais antiga para a mais recente.
func (r *InMemoryTimeRecordRepository) List(filter TimeRecordFilter) ([]model.TimeRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	records := make([]model.TimeRecord, 0)
	for _, record := range r.records {
		if filter.EmployeeID != 0 && record.EmployeeID != filter.EmployeeID {
			continue
		}
		if !filter.From.IsZero() && record.RecordedAt.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && !record.RecordedAt.Before(filter.To) {
			continue
		}
		records = append(records, record)
	}

	sort.Slice(records, func(i, j int) bool {
		if records[i].RecordedAt.Equal(records[j].RecordedAt) {
			return records[i].ID < records[j].ID
		}
		return records[i].RecordedAt.Before(records[j].RecordedAt)
	})

	return records, nil
}
