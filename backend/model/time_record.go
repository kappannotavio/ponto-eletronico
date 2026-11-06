package model

import "time"

const (
	SourceDevice = "device"
	SourceManual = "manual"
)

// TimeRecord e uma batida de ponto individual. As batidas de um dia sao
// alternadas: 1a entrada, 2a saida, 3a entrada (volta do almoco), 4a saida...
type TimeRecord struct {
	ID         int64     `json:"id"`
	EmployeeID int64     `json:"employeeId"`
	CardTag    string    `json:"cardTag"`
	RecordedAt time.Time `json:"recordedAt"`
	Source     string    `json:"source"`
	CreatedAt  time.Time `json:"createdAt"`
}
