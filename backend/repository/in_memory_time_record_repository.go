package repository

import "github.com/kappannotavio/ponto-eletronico/model"

type InMemoryTimeRecordRepository struct {
	records []model.TimeRecord
}

func NewInMemoryTimeRecordRepository() *InMemoryTimeRecordRepository {
	return &InMemoryTimeRecordRepository{
		records: make([]model.TimeRecord, 0),
	}
}

func (r *InMemoryTimeRecordRepository) Save(record model.TimeRecord) (model.TimeRecord, error) {
	r.records = append(r.records, record)

	return record, nil
}
