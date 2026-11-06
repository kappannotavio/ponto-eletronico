package service

import (
	"errors"
	"testing"
	"time"

	"github.com/kappannotavio/ponto-eletronico/model"
	"github.com/kappannotavio/ponto-eletronico/repository"
)

func newTestServices(t *testing.T, now time.Time) (*EmployeeService, *TimeRecordService, *CardReadService) {
	t.Helper()

	clock := func() time.Time { return now }

	employees := NewEmployeeService(repository.NewInMemoryEmployeeRepository())
	employees.now = clock
	timeRecords := NewTimeRecordService(repository.NewInMemoryTimeRecordRepository(), employees)
	timeRecords.now = clock
	cardReads := NewCardReadService(employees)
	cardReads.now = clock

	return employees, timeRecords, cardReads
}

func createEmployee(t *testing.T, employees *EmployeeService) model.Employee {
	t.Helper()

	employee, err := employees.Create(EmployeeInput{
		FullName:   "Ana Souza",
		StartTime:  "08:00",
		DailyHours: 8,
		LunchBreak: 60,
		CardTag:    "a1:b2:c3:d4",
		Pin:        "1234",
	})
	if err != nil {
		t.Fatalf("create employee: %v", err)
	}

	return employee
}

func assertKind(t *testing.T, err error, kind ErrorKind) {
	t.Helper()

	var serviceError *Error
	if !errors.As(err, &serviceError) || serviceError.Kind != kind {
		t.Fatalf("expected error kind %d, got %v", kind, err)
	}
}

func TestEmployeeCreateNormalizesTagAndRejectsDuplicates(t *testing.T) {
	employees, _, _ := newTestServices(t, time.Now())
	employee := createEmployee(t, employees)

	if employee.CardTag != "A1:B2:C3:D4" {
		t.Fatalf("expected normalized tag, got %q", employee.CardTag)
	}

	_, err := employees.Create(EmployeeInput{FullName: "Outra", CardTag: "A1:B2:C3:D4 ", Pin: "1111"})
	assertKind(t, err, KindConflict)

	_, err = employees.Create(EmployeeInput{FullName: "Outra", CardTag: "FF", Pin: "12"})
	assertKind(t, err, KindValidation)
}

func TestEmployeeUpdateKeepsPinWhenEmpty(t *testing.T) {
	employees, timeRecords, _ := newTestServices(t, time.Date(2026, 11, 6, 8, 0, 0, 0, time.Local))
	employee := createEmployee(t, employees)

	_, err := employees.Update(employee.ID, EmployeeInput{
		FullName: "Ana Souza Lima", StartTime: "08:00", DailyHours: 8, LunchBreak: 60, CardTag: employee.CardTag,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	if _, err := timeRecords.Punch(PunchInput{CardTag: employee.CardTag, Pin: "1234"}); err != nil {
		t.Fatalf("punch with old pin: %v", err)
	}
}

func TestPunchValidatesPinStatusAndDuplicates(t *testing.T) {
	now := time.Date(2026, 11, 6, 8, 5, 0, 0, time.Local)
	employees, timeRecords, _ := newTestServices(t, now)
	employee := createEmployee(t, employees)

	_, err := timeRecords.Punch(PunchInput{CardTag: "00:00", Pin: "1234"})
	assertKind(t, err, KindNotFound)

	_, err = timeRecords.Punch(PunchInput{CardTag: employee.CardTag, Pin: "9999"})
	assertKind(t, err, KindUnauthorized)

	result, err := timeRecords.Punch(PunchInput{CardTag: "a1:b2:c3:d4", Pin: "1234"})
	if err != nil {
		t.Fatalf("punch: %v", err)
	}
	if result.Type != "entry" || result.PunchNumber != 1 || result.EmployeeName != "Ana Souza" {
		t.Fatalf("unexpected result: %+v", result)
	}

	_, err = timeRecords.Punch(PunchInput{CardTag: employee.CardTag, Pin: "1234"})
	assertKind(t, err, KindConflict)

	later := now.Add(4 * time.Hour)
	timeRecords.now = func() time.Time { return later }
	result, err = timeRecords.Punch(PunchInput{CardTag: employee.CardTag, Pin: "1234", RecordedAt: &later})
	if err != nil {
		t.Fatalf("second punch: %v", err)
	}
	if result.Type != "exit" || result.PunchNumber != 2 {
		t.Fatalf("expected exit, got %+v", result)
	}

	future := later.Add(time.Hour)
	_, err = timeRecords.Punch(PunchInput{CardTag: employee.CardTag, Pin: "1234", RecordedAt: &future})
	assertKind(t, err, KindValidation)

	if _, err := employees.SetActive(employee.ID, false); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	_, err = timeRecords.Punch(PunchInput{CardTag: employee.CardTag, Pin: "1234"})
	assertKind(t, err, KindForbidden)
}

func TestAttendanceSummary(t *testing.T) {
	now := time.Date(2026, 11, 6, 18, 0, 0, 0, time.Local)
	employees, timeRecords, _ := newTestServices(t, now)
	employee := createEmployee(t, employees)

	tests := []struct {
		name          string
		date          string
		times         []string
		status        string
		workedMinutes int
		exit          bool
	}{
		{"normal day with lunch", "2026-11-02", []string{"08:00", "12:00", "13:00", "17:00"}, model.StatusNormal, 480, true},
		{"single pair deducts lunch", "2026-11-03", []string{"08:05", "17:05"}, model.StatusNormal, 480, true},
		{"late", "2026-11-04", []string{"08:11", "12:00", "13:00", "17:00"}, model.StatusLate, 469, true},
		{"overtime", "2026-11-05", []string{"08:00", "12:00", "13:00", "18:30"}, model.StatusOvertime, 570, true},
		{"incomplete past day", "2026-11-01", []string{"08:00"}, model.StatusIncomplete, 0, false},
		{"in progress today", "2026-11-06", []string{"08:00", "12:00", "13:00"}, model.StatusInProgress, 240, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			day, err := timeRecords.ReplaceDay(employee.ID, test.date, test.times)
			if err != nil {
				t.Fatalf("replace day: %v", err)
			}
			if day.Status != test.status {
				t.Errorf("status = %s, want %s", day.Status, test.status)
			}
			if day.WorkedMinutes != test.workedMinutes {
				t.Errorf("worked = %d, want %d", day.WorkedMinutes, test.workedMinutes)
			}
			if (day.Exit != nil) != test.exit {
				t.Errorf("exit = %v, want present=%v", day.Exit, test.exit)
			}
			if *day.Entry != test.times[0] {
				t.Errorf("entry = %s, want %s", *day.Entry, test.times[0])
			}
		})
	}

	all, err := timeRecords.Attendance(AttendanceFilter{})
	if err != nil {
		t.Fatalf("attendance: %v", err)
	}
	if len(all) != len(tests) || all[0].Date != "2026-11-06" {
		t.Fatalf("expected %d days sorted desc, got %d (first %s)", len(tests), len(all), all[0].Date)
	}

	dashboard, err := timeRecords.Dashboard()
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if dashboard.PresentCount != 1 || dashboard.AbsentCount != 0 || len(dashboard.Today) != 1 {
		t.Fatalf("unexpected dashboard: %+v", dashboard)
	}
}

func TestReplaceDayKeepsMatchingDeviceRecords(t *testing.T) {
	now := time.Date(2026, 11, 6, 8, 0, 0, 0, time.Local)
	employees, timeRecords, _ := newTestServices(t, now)
	employee := createEmployee(t, employees)

	if _, err := timeRecords.Punch(PunchInput{CardTag: employee.CardTag, Pin: "1234"}); err != nil {
		t.Fatalf("punch: %v", err)
	}

	day, err := timeRecords.ReplaceDay(employee.ID, "2026-11-06", []string{"08:00", "17:00"})
	if err != nil {
		t.Fatalf("replace day: %v", err)
	}
	if len(day.Records) != 2 || day.Records[0].Source != model.SourceDevice || day.Records[1].Source != model.SourceManual {
		t.Fatalf("unexpected records: %+v", day.Records)
	}

	day, err = timeRecords.ReplaceDay(employee.ID, "2026-11-06", nil)
	if err != nil || len(day.Records) != 0 {
		t.Fatalf("expected empty day, got %+v (%v)", day, err)
	}
}

func TestCardReadFlow(t *testing.T) {
	now := time.Date(2026, 11, 6, 8, 0, 0, 0, time.Local)
	employees, _, cardReads := newTestServices(t, now)
	employee := createEmployee(t, employees)

	first, _ := cardReads.Create()
	second, _ := cardReads.Create()

	if read, _ := cardReads.Get(first.ID); read.Status != model.CardReadCancelled {
		t.Fatalf("first read should be cancelled, got %s", read.Status)
	}

	pending, ok := cardReads.Pending()
	if !ok || pending.ID != second.ID {
		t.Fatalf("expected pending %s, got %+v", second.ID, pending)
	}

	read, err := cardReads.Complete(second.ID, "a1:b2:c3:d4")
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if read.CardTag != employee.CardTag || read.AssignedTo == nil || read.AssignedTo.ID != employee.ID {
		t.Fatalf("unexpected read: %+v", read)
	}

	if _, ok := cardReads.Pending(); ok {
		t.Fatal("no read should be pending")
	}

	_, err = cardReads.Complete(second.ID, "FF")
	assertKind(t, err, KindConflict)

	third, _ := cardReads.Create()
	cardReads.now = func() time.Time { return now.Add(2 * time.Minute) }
	if read, _ := cardReads.Get(third.ID); read.Status != model.CardReadExpired {
		t.Fatalf("expected expired, got %s", read.Status)
	}
}
