package service

import (
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kappannotavio/ponto-eletronico/model"
	"github.com/kappannotavio/ponto-eletronico/repository"
)

var (
	pinPattern       = regexp.MustCompile(`^[0-9]{4}$`)
	startTimePattern = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
)

type EmployeeInput struct {
	FullName   string `json:"fullName"`
	JobTitle   string `json:"jobTitle"`
	Whatsapp   string `json:"whatsapp"`
	StartTime  string `json:"startTime"`
	DailyHours int    `json:"dailyHours"`
	LunchBreak int    `json:"lunchBreak"`
	CardTag    string `json:"cardTag"`
	Pin        string `json:"pin"`
}

type EmployeeFilter struct {
	Search string
	Active *bool
}

type EmployeeService struct {
	// mu serializa escritas para a checagem de tag duplicada ser confiavel.
	mu         sync.Mutex
	repository repository.EmployeeRepository
	now        func() time.Time
}

func NewEmployeeService(repository repository.EmployeeRepository) *EmployeeService {
	return &EmployeeService{
		repository: repository,
		now:        time.Now,
	}
}

// NormalizeCardTag deixa a tag no formato usado pelo ESP32: "AA:BB:CC:DD".
func NormalizeCardTag(cardTag string) string {
	return strings.ToUpper(strings.TrimSpace(cardTag))
}

func (s *EmployeeService) List(filter EmployeeFilter) ([]model.Employee, error) {
	employees, err := s.repository.List()
	if err != nil {
		return nil, err
	}

	search := strings.ToLower(strings.TrimSpace(filter.Search))
	result := make([]model.Employee, 0, len(employees))

	for _, employee := range employees {
		if filter.Active != nil && employee.IsActive != *filter.Active {
			continue
		}
		if search != "" &&
			!strings.Contains(strings.ToLower(employee.FullName), search) &&
			!strings.Contains(strings.ToLower(employee.JobTitle), search) &&
			!strings.Contains(strings.ToLower(employee.CardTag), search) {
			continue
		}
		result = append(result, employee)
	}

	return result, nil
}

func (s *EmployeeService) Get(id int64) (model.Employee, error) {
	employee, ok, err := s.repository.FindByID(id)
	if err != nil {
		return model.Employee{}, err
	}
	if !ok {
		return model.Employee{}, newError(KindNotFound, "funcionario nao encontrado")
	}

	return employee, nil
}

func (s *EmployeeService) Create(input EmployeeInput) (model.Employee, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if input.StartTime == "" {
		input.StartTime = "08:00"
	}
	if input.DailyHours == 0 {
		input.DailyHours = 8
	}

	input = normalizeInput(input)

	if err := validateInput(input, true); err != nil {
		return model.Employee{}, err
	}
	if err := s.ensureCardTagAvailable(input.CardTag, 0); err != nil {
		return model.Employee{}, err
	}

	pinHash, err := hashPin(input.Pin)
	if err != nil {
		return model.Employee{}, err
	}

	now := s.now()
	employee := model.Employee{
		FullName:   input.FullName,
		JobTitle:   input.JobTitle,
		Whatsapp:   input.Whatsapp,
		StartTime:  input.StartTime,
		DailyHours: input.DailyHours,
		LunchBreak: input.LunchBreak,
		CardTag:    input.CardTag,
		PinHash:    pinHash,
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	return s.repository.Create(employee)
}

// Update substitui os dados do funcionario. Pin vazio mantem o PIN atual.
func (s *EmployeeService) Update(id int64, input EmployeeInput) (model.Employee, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	employee, err := s.Get(id)
	if err != nil {
		return model.Employee{}, err
	}

	input = normalizeInput(input)

	if err := validateInput(input, false); err != nil {
		return model.Employee{}, err
	}
	if err := s.ensureCardTagAvailable(input.CardTag, id); err != nil {
		return model.Employee{}, err
	}

	if input.Pin != "" {
		pinHash, err := hashPin(input.Pin)
		if err != nil {
			return model.Employee{}, err
		}
		employee.PinHash = pinHash
	}

	employee.FullName = input.FullName
	employee.JobTitle = input.JobTitle
	employee.Whatsapp = input.Whatsapp
	employee.StartTime = input.StartTime
	employee.DailyHours = input.DailyHours
	employee.LunchBreak = input.LunchBreak
	employee.CardTag = input.CardTag
	employee.UpdatedAt = s.now()

	return s.repository.Update(employee)
}

func (s *EmployeeService) SetActive(id int64, active bool) (model.Employee, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	employee, err := s.Get(id)
	if err != nil {
		return model.Employee{}, err
	}

	employee.IsActive = active
	employee.UpdatedAt = s.now()

	return s.repository.Update(employee)
}

func (s *EmployeeService) FindByCardTag(cardTag string) (model.Employee, error) {
	employee, ok, err := s.repository.FindByCardTag(NormalizeCardTag(cardTag))
	if err != nil {
		return model.Employee{}, err
	}
	if !ok {
		return model.Employee{}, newError(KindNotFound, "tag nao cadastrada")
	}

	return employee, nil
}

func (s *EmployeeService) ensureCardTagAvailable(cardTag string, currentID int64) error {
	owner, ok, err := s.repository.FindByCardTag(cardTag)
	if err != nil {
		return err
	}
	if ok && owner.ID != currentID {
		return newError(KindConflict, "a tag %s ja pertence a %s", cardTag, owner.FullName)
	}

	return nil
}

func normalizeInput(input EmployeeInput) EmployeeInput {
	input.FullName = strings.TrimSpace(input.FullName)
	input.JobTitle = strings.TrimSpace(input.JobTitle)
	input.Whatsapp = strings.TrimSpace(input.Whatsapp)
	input.StartTime = strings.TrimSpace(input.StartTime)
	input.CardTag = NormalizeCardTag(input.CardTag)
	input.Pin = strings.TrimSpace(input.Pin)

	return input
}

func validateInput(input EmployeeInput, pinRequired bool) error {
	switch {
	case input.FullName == "":
		return newError(KindValidation, "nome completo e obrigatorio")
	case input.CardTag == "":
		return newError(KindValidation, "cartao ou tag e obrigatorio")
	case !startTimePattern.MatchString(input.StartTime):
		return newError(KindValidation, "horario de entrada deve estar no formato HH:MM")
	case input.DailyHours < 1 || input.DailyHours > 12:
		return newError(KindValidation, "horas por dia deve estar entre 1 e 12")
	case input.LunchBreak < 0 || input.LunchBreak > 180:
		return newError(KindValidation, "intervalo de almoco deve estar entre 0 e 180 minutos")
	case pinRequired && input.Pin == "":
		return newError(KindValidation, "senha e obrigatoria")
	case input.Pin != "" && !pinPattern.MatchString(input.Pin):
		return newError(KindValidation, "senha deve ter exatamente 4 numeros")
	}

	return nil
}
