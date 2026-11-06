package main

import (
	"time"

	"github.com/kappannotavio/ponto-eletronico/model"
	"github.com/kappannotavio/ponto-eletronico/repository"
	"github.com/kappannotavio/ponto-eletronico/service"
)

// seedDemoData cria funcionarios e batidas de exemplo (SEED_DEMO_DATA=true).
// Todos usam o PIN 1234. As batidas entram como vindas do relogio.
func seedDemoData(employees *service.EmployeeService, timeRecords repository.TimeRecordRepository) error {
	inputs := []service.EmployeeInput{
		{FullName: "Ana Souza", JobTitle: "Recepcionista", Whatsapp: "11900000001", StartTime: "08:00", DailyHours: 8, LunchBreak: 60, CardTag: "A1:B2:C3:D4", Pin: "1234"},
		{FullName: "Bruno Lima", JobTitle: "Auxiliar administrativo", Whatsapp: "11900000002", StartTime: "09:00", DailyHours: 6, LunchBreak: 30, CardTag: "B1:B2:B3:B4", Pin: "1234"},
		{FullName: "Carla Mendes", JobTitle: "Assistente de loja", Whatsapp: "11900000003", StartTime: "08:30", DailyHours: 8, LunchBreak: 60, CardTag: "C1:C2:C3:C4", Pin: "1234"},
	}

	created := make([]model.Employee, 0, len(inputs))
	for _, input := range inputs {
		employee, err := employees.Create(input)
		if err != nil {
			return err
		}
		created = append(created, employee)
	}

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	today := now

	days := []struct {
		employee model.Employee
		date     time.Time
		times    []string
	}{
		{created[0], yesterday, []string{"08:02", "12:00", "13:00", "17:00"}},
		{created[1], yesterday, []string{"09:12", "15:00"}},
		{created[2], yesterday, []string{"08:30", "12:00", "13:00", "17:45"}},
		{created[0], today, []string{"07:58"}},
		{created[2], today, []string{"08:47"}},
	}

	for _, day := range days {
		for _, value := range day.times {
			clock, _ := time.Parse(service.HourLayout, value)
			recordedAt := time.Date(day.date.Year(), day.date.Month(), day.date.Day(), clock.Hour(), clock.Minute(), 0, 0, time.Local)

			_, err := timeRecords.Save(model.TimeRecord{
				EmployeeID: day.employee.ID,
				CardTag:    day.employee.CardTag,
				RecordedAt: recordedAt,
				Source:     model.SourceDevice,
				CreatedAt:  now,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
