package service

import (
	"errors"
	"time"

	"github.com/kappannotavio/ponto-eletronico/model"
	"github.com/kappannotavio/ponto-eletronico/repository"
)

type TimeRecordService struct {
	repository repository.TimeRecordRepository
}

func NewTimeRecordService(repository repository.TimeRecordRepository) *TimeRecordService {

	return &TimeRecordService{
		repository: repository,
	}

}

func (s *TimeRecordService) CreateTimeRecord(tagID string, timestamp time.Time) (model.TimeRecord, error) {
	if tagID == "" {
		return model.TimeRecord{}, errors.New("tag_id is required")
	}

	record := model.TimeRecord{
		TagID:     tagID,
		Timestamp: timestamp,
	}

	return s.repository.Save(record)
}
