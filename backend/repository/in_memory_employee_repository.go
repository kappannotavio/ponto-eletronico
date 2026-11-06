package repository

import (
	"sort"
	"sync"

	"github.com/kappannotavio/ponto-eletronico/model"
)

type InMemoryEmployeeRepository struct {
	mu        sync.RWMutex
	nextID    int64
	employees map[int64]model.Employee
}

func NewInMemoryEmployeeRepository() *InMemoryEmployeeRepository {
	return &InMemoryEmployeeRepository{
		nextID:    1,
		employees: make(map[int64]model.Employee),
	}
}

func (r *InMemoryEmployeeRepository) Create(employee model.Employee) (model.Employee, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	employee.ID = r.nextID
	r.nextID++
	r.employees[employee.ID] = employee

	return employee, nil
}

func (r *InMemoryEmployeeRepository) Update(employee model.Employee) (model.Employee, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.employees[employee.ID]; !ok {
		return model.Employee{}, ErrNotFound
	}

	r.employees[employee.ID] = employee

	return employee, nil
}

func (r *InMemoryEmployeeRepository) FindByID(id int64) (model.Employee, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	employee, ok := r.employees[id]

	return employee, ok, nil
}

func (r *InMemoryEmployeeRepository) FindByCardTag(cardTag string) (model.Employee, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, employee := range r.employees {
		if employee.CardTag == cardTag {
			return employee, true, nil
		}
	}

	return model.Employee{}, false, nil
}

func (r *InMemoryEmployeeRepository) List() ([]model.Employee, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	employees := make([]model.Employee, 0, len(r.employees))
	for _, employee := range r.employees {
		employees = append(employees, employee)
	}

	sort.Slice(employees, func(i, j int) bool {
		return employees[i].ID < employees[j].ID
	})

	return employees, nil
}
