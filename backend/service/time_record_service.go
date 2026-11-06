package service

import (
	"sort"
	"strings"
	"time"

	"github.com/kappannotavio/ponto-eletronico/model"
	"github.com/kappannotavio/ponto-eletronico/repository"
)

const (
	DateLayout = "2006-01-02"
	HourLayout = "15:04"

	// Tolerancias da CLT (art. 58, par. 1o): ate 10 minutos por dia nao contam
	// como atraso nem como hora extra.
	lateTolerance     = 10 * time.Minute
	overtimeTolerance = 10 * time.Minute

	// Batidas repetidas da mesma tag dentro desse intervalo sao ignoradas
	// (evita registro duplo ao encostar o cartao duas vezes ou reenviar a fila offline).
	duplicateWindow = time.Minute

	// Aceita relogios do dispositivo levemente adiantados.
	maxClockSkew = 5 * time.Minute
)

type PunchInput struct {
	CardTag    string     `json:"cardTag"`
	Pin        string     `json:"pin"`
	RecordedAt *time.Time `json:"recordedAt"`
}

type PunchResult struct {
	Record       model.TimeRecord `json:"record"`
	EmployeeName string           `json:"employeeName"`
	Type         string           `json:"type"`
	PunchNumber  int              `json:"punchNumber"`
}

type AttendanceFilter struct {
	EmployeeID int64
	From       time.Time
	To         time.Time
}

type TimeRecordService struct {
	repository repository.TimeRecordRepository
	employees  *EmployeeService
	now        func() time.Time
}

func NewTimeRecordService(
	repository repository.TimeRecordRepository,
	employees *EmployeeService,
) *TimeRecordService {
	return &TimeRecordService{
		repository: repository,
		employees:  employees,
		now:        time.Now,
	}
}

// Punch registra uma batida vinda do relogio de ponto (tag + PIN).
func (s *TimeRecordService) Punch(input PunchInput) (PunchResult, error) {
	employee, err := s.employees.FindByCardTag(input.CardTag)
	if err != nil {
		return PunchResult{}, err
	}
	if !employee.IsActive {
		return PunchResult{}, newError(KindForbidden, "funcionario %s esta desativado", employee.FullName)
	}
	if !verifyPin(employee.PinHash, strings.TrimSpace(input.Pin)) {
		return PunchResult{}, newError(KindUnauthorized, "senha incorreta")
	}

	now := s.now()
	recordedAt := now
	if input.RecordedAt != nil {
		recordedAt = *input.RecordedAt
		if recordedAt.After(now.Add(maxClockSkew)) {
			return PunchResult{}, newError(KindValidation, "horario da batida esta no futuro")
		}
	}
	recordedAt = recordedAt.In(time.Local)

	dayRecords, err := s.dayRecords(employee.ID, recordedAt)
	if err != nil {
		return PunchResult{}, err
	}
	for _, record := range dayRecords {
		if absDuration(record.RecordedAt.Sub(recordedAt)) < duplicateWindow {
			return PunchResult{}, newError(KindConflict, "ponto ja registrado ha menos de 1 minuto")
		}
	}

	record, err := s.repository.Save(model.TimeRecord{
		EmployeeID: employee.ID,
		CardTag:    employee.CardTag,
		RecordedAt: recordedAt,
		Source:     model.SourceDevice,
		CreatedAt:  now,
	})
	if err != nil {
		return PunchResult{}, err
	}

	punchNumber := 1
	for _, existing := range dayRecords {
		if existing.RecordedAt.Before(recordedAt) {
			punchNumber++
		}
	}

	punchType := "entry"
	if punchNumber%2 == 0 {
		punchType = "exit"
	}

	return PunchResult{
		Record:       record,
		EmployeeName: employee.FullName,
		Type:         punchType,
		PunchNumber:  punchNumber,
	}, nil
}

// CreateManual registra uma batida lancada pelo painel (esquecimento, correcao).
func (s *TimeRecordService) CreateManual(employeeID int64, recordedAt time.Time) (model.TimeRecord, error) {
	employee, err := s.employees.Get(employeeID)
	if err != nil {
		return model.TimeRecord{}, err
	}
	if recordedAt.IsZero() {
		return model.TimeRecord{}, newError(KindValidation, "recordedAt e obrigatorio")
	}

	return s.repository.Save(model.TimeRecord{
		EmployeeID: employee.ID,
		CardTag:    employee.CardTag,
		RecordedAt: recordedAt.In(time.Local),
		Source:     model.SourceManual,
		CreatedAt:  s.now(),
	})
}

func (s *TimeRecordService) Delete(id int64) error {
	if err := s.repository.Delete(id); err != nil {
		if err == repository.ErrNotFound {
			return newError(KindNotFound, "batida nao encontrada")
		}
		return err
	}

	return nil
}

func (s *TimeRecordService) List(filter repository.TimeRecordFilter) ([]model.TimeRecord, error) {
	return s.repository.List(filter)
}

// ReplaceDay define as batidas de um funcionario em um dia ("HH:MM").
// Batidas existentes com o mesmo horario sao mantidas (preservando a origem),
// as demais sao removidas e os horarios novos entram como manuais.
// Uma lista vazia apaga o dia.
func (s *TimeRecordService) ReplaceDay(employeeID int64, date string, times []string) (model.Attendance, error) {
	employee, err := s.employees.Get(employeeID)
	if err != nil {
		return model.Attendance{}, err
	}

	day, err := ParseDate(date)
	if err != nil {
		return model.Attendance{}, err
	}

	wanted := make(map[string]int, len(times))
	for _, value := range times {
		value = strings.TrimSpace(value)
		if _, err := time.Parse(HourLayout, value); err != nil {
			return model.Attendance{}, newError(KindValidation, "horario invalido: %q (use HH:MM)", value)
		}
		wanted[value]++
	}

	existing, err := s.dayRecords(employee.ID, day)
	if err != nil {
		return model.Attendance{}, err
	}

	for _, record := range existing {
		key := record.RecordedAt.Format(HourLayout)
		if wanted[key] > 0 {
			wanted[key]--
			continue
		}
		if err := s.repository.Delete(record.ID); err != nil {
			return model.Attendance{}, err
		}
	}

	for value, count := range wanted {
		clock, _ := time.Parse(HourLayout, value)
		recordedAt := time.Date(day.Year(), day.Month(), day.Day(), clock.Hour(), clock.Minute(), 0, 0, time.Local)
		for i := 0; i < count; i++ {
			if _, err := s.CreateManual(employee.ID, recordedAt); err != nil {
				return model.Attendance{}, err
			}
		}
	}

	records, err := s.dayRecords(employee.ID, day)
	if err != nil {
		return model.Attendance{}, err
	}

	return s.summarize(employee, day.Format(DateLayout), records), nil
}

// Attendance agrupa as batidas por funcionario e dia, do dia mais recente ao mais antigo.
func (s *TimeRecordService) Attendance(filter AttendanceFilter) ([]model.Attendance, error) {
	records, err := s.repository.List(repository.TimeRecordFilter{
		EmployeeID: filter.EmployeeID,
		From:       filter.From,
		To:         filter.To,
	})
	if err != nil {
		return nil, err
	}

	type dayKey struct {
		employeeID int64
		date       string
	}

	grouped := make(map[dayKey][]model.TimeRecord)
	for _, record := range records {
		key := dayKey{record.EmployeeID, record.RecordedAt.In(time.Local).Format(DateLayout)}
		grouped[key] = append(grouped[key], record)
	}

	result := make([]model.Attendance, 0, len(grouped))
	for key, dayRecords := range grouped {
		employee, err := s.employees.Get(key.employeeID)
		if err != nil {
			continue
		}
		result = append(result, s.summarize(employee, key.date, dayRecords))
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Date != result[j].Date {
			return result[i].Date > result[j].Date
		}
		return result[i].EmployeeName < result[j].EmployeeName
	})

	return result, nil
}

func (s *TimeRecordService) Dashboard() (model.Dashboard, error) {
	today := startOfDay(s.now())

	attendance, err := s.Attendance(AttendanceFilter{
		From: today,
		To:   today.AddDate(0, 0, 1),
	})
	if err != nil {
		return model.Dashboard{}, err
	}

	active := true
	employees, err := s.employees.List(EmployeeFilter{Active: &active})
	if err != nil {
		return model.Dashboard{}, err
	}

	present := make(map[int64]bool, len(attendance))
	dashboard := model.Dashboard{
		Date:        today.Format(DateLayout),
		ActiveCount: len(employees),
		Today:       attendance,
	}

	for _, day := range attendance {
		present[day.EmployeeID] = true
		if day.Late {
			dashboard.LateCount++
		}
	}
	for _, employee := range employees {
		if present[employee.ID] {
			dashboard.PresentCount++
		} else {
			dashboard.AbsentCount++
		}
	}

	return dashboard, nil
}

func (s *TimeRecordService) dayRecords(employeeID int64, day time.Time) ([]model.TimeRecord, error) {
	start := startOfDay(day)

	return s.repository.List(repository.TimeRecordFilter{
		EmployeeID: employeeID,
		From:       start,
		To:         start.AddDate(0, 0, 1),
	})
}

// summarize calcula entrada, saida, horas trabalhadas e situacao de um dia.
// Regras:
//   - batidas alternam entrada/saida; horas = soma dos pares completos;
//   - com so um par (sem batida de almoco), desconta o intervalo de almoco cadastrado;
//   - atraso quando a 1a batida passa do horario de entrada + tolerancia;
//   - hora extra quando as horas passam da jornada + tolerancia;
//   - numero impar de batidas: "em andamento" hoje, "incompleto" em dias passados.
func (s *TimeRecordService) summarize(employee model.Employee, date string, records []model.TimeRecord) model.Attendance {
	sort.Slice(records, func(i, j int) bool {
		return records[i].RecordedAt.Before(records[j].RecordedAt)
	})

	attendance := model.Attendance{
		EmployeeID:   employee.ID,
		EmployeeName: employee.FullName,
		Date:         date,
		Records:      records,
		Status:       model.StatusNormal,
		Times:        make([]string, 0, len(records)),
	}

	for _, record := range records {
		attendance.Times = append(attendance.Times, record.RecordedAt.In(time.Local).Format(HourLayout))
	}

	if len(records) == 0 {
		return attendance
	}

	entry := records[0].RecordedAt.In(time.Local).Format(HourLayout)
	attendance.Entry = &entry

	if len(records)%2 == 0 {
		exit := records[len(records)-1].RecordedAt.In(time.Local).Format(HourLayout)
		attendance.Exit = &exit
	}

	var worked time.Duration
	for i := 0; i+1 < len(records); i += 2 {
		worked += records[i+1].RecordedAt.Sub(records[i].RecordedAt)
	}
	if len(records) == 2 && worked > time.Duration(employee.LunchBreak)*time.Minute {
		worked -= time.Duration(employee.LunchBreak) * time.Minute
	}
	attendance.WorkedMinutes = int(worked / time.Minute)

	if expected, err := time.Parse(HourLayout, employee.StartTime); err == nil {
		first := records[0].RecordedAt.In(time.Local)
		expectedAt := time.Date(first.Year(), first.Month(), first.Day(), expected.Hour(), expected.Minute(), 0, 0, time.Local)
		attendance.Late = first.After(expectedAt.Add(lateTolerance))
	}

	isToday := date == s.now().Format(DateLayout)
	overtime := worked > time.Duration(employee.DailyHours)*time.Hour+overtimeTolerance

	switch {
	case len(records)%2 == 1 && !isToday:
		attendance.Status = model.StatusIncomplete
	case attendance.Late:
		attendance.Status = model.StatusLate
	case len(records)%2 == 1:
		attendance.Status = model.StatusInProgress
	case overtime:
		attendance.Status = model.StatusOvertime
	}

	return attendance
}

// ParseDate interpreta "YYYY-MM-DD" no fuso local do servidor.
func ParseDate(value string) (time.Time, error) {
	day, err := time.ParseInLocation(DateLayout, strings.TrimSpace(value), time.Local)
	if err != nil {
		return time.Time{}, newError(KindValidation, "data invalida: %q (use YYYY-MM-DD)", value)
	}

	return day, nil
}

func startOfDay(t time.Time) time.Time {
	t = t.In(time.Local)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
