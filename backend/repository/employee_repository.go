package repository

import "github.com/kappannotavio/ponto-eletronico/model"

type EmployeeRepository interface {
	Create(employee model.Employee) (model.Employee, error)
	Update(employee model.Employee) (model.Employee, error)
	FindByID(id int64) (model.Employee, bool, error)
	FindByCardTag(cardTag string) (model.Employee, bool, error)
	List() ([]model.Employee, error)
}
