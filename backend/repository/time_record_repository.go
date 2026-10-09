package repository

import "github.com/kappannotavio/ponto-eletronico/model"

// depois eu ajeito melhor isso aqui, acho que ta faltando coisa, nao tem banco ainda
type TimeRecordRepository interface {
	Save(record model.TimeRecord) (model.TimeRecord, error)
}
