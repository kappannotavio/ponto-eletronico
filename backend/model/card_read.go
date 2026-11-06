package model

import "time"

const (
	CardReadWaiting   = "waiting"
	CardReadCompleted = "completed"
	CardReadCancelled = "cancelled"
	CardReadExpired   = "expired"
)

// CardRead e um pedido do painel para que o relogio de ponto leia
// a proxima tag aproximada (usado no cadastro/edicao de funcionario).
type CardRead struct {
	ID         string       `json:"id"`
	Status     string       `json:"status"`
	CardTag    string       `json:"cardTag,omitempty"`
	AssignedTo *EmployeeRef `json:"assignedTo,omitempty"`
	CreatedAt  time.Time    `json:"createdAt"`
	ExpiresAt  time.Time    `json:"expiresAt"`
}

type EmployeeRef struct {
	ID       int64  `json:"id"`
	FullName string `json:"fullName"`
}
