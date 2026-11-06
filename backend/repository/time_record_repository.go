package repository

import (
	"time"

	"github.com/kappannotavio/ponto-eletronico/model"
)

// TimeRecordFilter filtra batidas. Campos zerados nao filtram.
// From e inclusivo e To e exclusivo.
type TimeRecordFilter struct {
	EmployeeID int64
	From       time.Time
	To         time.Time
}

type TimeRecordRepository interface {
	Save(record model.TimeRecord) (model.TimeRecord, error)
	FindByID(id int64) (model.TimeRecord, bool, error)
	Delete(id int64) error
	List(filter TimeRecordFilter) ([]model.TimeRecord, error)
}
