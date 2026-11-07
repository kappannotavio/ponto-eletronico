package service

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/kappannotavio/ponto-eletronico/model"
)

const (
	cardReadTTL       = 60 * time.Second
	cardReadRetention = 10 * time.Minute
)

// CardReadService coordena o pedido "leia a proxima tag" entre o painel
// e o relogio de ponto. Fica so em memoria porque e efemero.
type CardReadService struct {
	mu        sync.Mutex
	reads     map[string]*model.CardRead
	employees *EmployeeService
	now       func() time.Time
}

func NewCardReadService(employees *EmployeeService) *CardReadService {
	return &CardReadService{
		reads:     make(map[string]*model.CardRead),
		employees: employees,
		now:       time.Now,
	}
}

// Create abre um novo pedido de leitura e cancela os que ainda estavam esperando,
// ja que o relogio so consegue atender um por vez.
func (s *CardReadService) Create() (model.CardRead, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	for id, read := range s.reads {
		s.refresh(read)
		if read.Status == model.CardReadWaiting {
			read.Status = model.CardReadCancelled
		}
		if now.Sub(read.CreatedAt) > cardReadRetention {
			delete(s.reads, id)
		}
	}

	id, err := randomID()
	if err != nil {
		return model.CardRead{}, err
	}

	read := &model.CardRead{
		ID:        id,
		Status:    model.CardReadWaiting,
		CreatedAt: now,
		ExpiresAt: now.Add(cardReadTTL),
	}
	s.reads[id] = read

	return *read, nil
}

func (s *CardReadService) Get(id string) (model.CardRead, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	read, ok := s.reads[id]
	if !ok {
		return model.CardRead{}, newError(KindNotFound, "leitura nao encontrada")
	}
	s.refresh(read)

	return *read, nil
}

func (s *CardReadService) Cancel(id string) (model.CardRead, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	read, ok := s.reads[id]
	if !ok {
		return model.CardRead{}, newError(KindNotFound, "leitura nao encontrada")
	}
	s.refresh(read)
	if read.Status == model.CardReadWaiting {
		read.Status = model.CardReadCancelled
	}

	return *read, nil
}

// Pending devolve o pedido que o relogio deve atender, se houver.
func (s *CardReadService) Pending() (model.CardRead, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, read := range s.reads {
		s.refresh(read)
		if read.Status == model.CardReadWaiting {
			return *read, true
		}
	}

	return model.CardRead{}, false
}

func (s *CardReadService) Complete(id, cardTag string) (model.CardRead, error) {
	cardTag = NormalizeCardTag(cardTag)
	if cardTag == "" {
		return model.CardRead{}, newError(KindValidation, "cardTag e obrigatorio")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	read, ok := s.reads[id]
	if !ok {
		return model.CardRead{}, newError(KindNotFound, "leitura nao encontrada")
	}
	s.refresh(read)
	if read.Status != model.CardReadWaiting {
		return model.CardRead{}, newError(KindConflict, "leitura nao esta mais aguardando (%s)", read.Status)
	}

	read.Status = model.CardReadCompleted
	read.CardTag = cardTag
	if owner, err := s.employees.FindByCardTag(cardTag); err == nil {
		read.AssignedTo = &model.EmployeeRef{ID: owner.ID, FullName: owner.FullName}
	}

	return *read, nil
}

func (s *CardReadService) refresh(read *model.CardRead) {
	if read.Status == model.CardReadWaiting && s.now().After(read.ExpiresAt) {
		read.Status = model.CardReadExpired
	}
}

func randomID() (string, error) {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return hex.EncodeToString(buffer), nil
}
