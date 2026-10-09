package model

import "time"

type TimeRecord struct {
	TagID     string    `json:"tag_id"`
	Timestamp time.Time `json:"timestamp"`
}
