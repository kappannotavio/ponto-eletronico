package model

import "time"

type Employee struct {
	ID         int64     `json:"id"`
	FullName   string    `json:"fullName"`
	JobTitle   string    `json:"jobTitle"`
	Whatsapp   string    `json:"whatsapp"`
	StartTime  string    `json:"startTime"`
	DailyHours int       `json:"dailyHours"`
	LunchBreak int       `json:"lunchBreak"`
	CardTag    string    `json:"cardTag"`
	PinHash    string    `json:"-"`
	IsActive   bool      `json:"isActive"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}
