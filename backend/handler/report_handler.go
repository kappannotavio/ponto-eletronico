package handler

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/kappannotavio/ponto-eletronico/model"
	"github.com/kappannotavio/ponto-eletronico/report"
	"github.com/kappannotavio/ponto-eletronico/service"
)

type ReportHandler struct {
	employees   *service.EmployeeService
	timeRecords *service.TimeRecordService
}

func NewReportHandler(employees *service.EmployeeService, timeRecords *service.TimeRecordService) *ReportHandler {
	return &ReportHandler{
		employees:   employees,
		timeRecords: timeRecords,
	}
}

var statusLabels = map[string]string{
	model.StatusNormal:     "Normal",
	model.StatusInProgress: "Em andamento",
	model.StatusLate:       "Atraso",
	model.StatusOvertime:   "Hora extra",
	model.StatusIncomplete: "Incompleto",
}

// AttendanceCSV gera o relatorio com ";" e BOM UTF-8 para abrir direto no Excel em pt-BR.
func (h *ReportHandler) AttendanceCSV(w http.ResponseWriter, r *http.Request) {
	filter, ok := attendanceFilter(w, r)
	if !ok {
		return
	}

	attendance, err := h.timeRecords.Attendance(filter)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="relatorio-ponto.csv"`)
	w.Write([]byte("\ufeff"))

	writer := csv.NewWriter(w)
	writer.Comma = ';'
	writer.Write([]string{"Funcionario", "Data", "Entrada", "Saida", "Batidas", "Horas trabalhadas", "Situacao"})

	for _, day := range attendance {
		date, _ := time.Parse(service.DateLayout, day.Date)
		writer.Write([]string{
			day.EmployeeName,
			date.Format("02/01/2006"),
			valueOrDash(day.Entry),
			valueOrDash(day.Exit),
			fmt.Sprint(len(day.Records)),
			fmt.Sprintf("%02d:%02d", day.WorkedMinutes/60, day.WorkedMinutes%60),
			statusLabels[day.Status],
		})
	}

	writer.Flush()
}

// AttendancePDF gera o espelho de ponto, uma secao por funcionario.
func (h *ReportHandler) AttendancePDF(w http.ResponseWriter, r *http.Request) {
	filter, ok := attendanceFilter(w, r)
	if !ok {
		return
	}

	attendance, err := h.timeRecords.Attendance(filter)
	if err != nil {
		writeError(w, err)
		return
	}

	byEmployee := make(map[int64][]model.Attendance)
	for _, day := range attendance {
		byEmployee[day.EmployeeID] = append(byEmployee[day.EmployeeID], day)
	}

	// Filtrando um funcionario, ele aparece mesmo sem marcacoes no periodo.
	if filter.EmployeeID != 0 {
		if _, ok := byEmployee[filter.EmployeeID]; !ok {
			byEmployee[filter.EmployeeID] = nil
		}
	}

	employees := make([]report.EmployeeReport, 0, len(byEmployee))
	for id, days := range byEmployee {
		employee, err := h.employees.Get(id)
		if err != nil {
			writeError(w, err)
			return
		}

		sort.Slice(days, func(i, j int) bool { return days[i].Date < days[j].Date })
		employees = append(employees, report.EmployeeReport{Employee: employee, Days: days})
	}

	sort.Slice(employees, func(i, j int) bool {
		return employees[i].Employee.FullName < employees[j].Employee.FullName
	})

	var buffer bytes.Buffer
	err = report.WriteAttendancePDF(&buffer, report.AttendanceReport{
		Period:      periodLabel(filter, attendance),
		GeneratedAt: time.Now(),
		Employees:   employees,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="espelho-de-ponto.pdf"`)
	w.Write(buffer.Bytes())
}

// periodLabel descreve o filtro de datas; sem filtro, usa o intervalo dos dados.
func periodLabel(filter service.AttendanceFilter, attendance []model.Attendance) string {
	const layout = "02/01/2006"

	from, to := filter.From, filter.To
	if !to.IsZero() {
		to = to.AddDate(0, 0, -1)
	}

	if from.IsZero() && to.IsZero() && len(attendance) > 0 {
		// attendance vem do dia mais recente para o mais antigo.
		from, _ = time.Parse(service.DateLayout, attendance[len(attendance)-1].Date)
		to, _ = time.Parse(service.DateLayout, attendance[0].Date)
	}

	switch {
	case from.IsZero() && to.IsZero():
		return "todas as marcações"
	case to.IsZero():
		return "a partir de " + from.Format(layout)
	case from.IsZero():
		return "até " + to.Format(layout)
	case from.Equal(to):
		return from.Format(layout)
	default:
		return from.Format(layout) + " a " + to.Format(layout)
	}
}

func valueOrDash(value *string) string {
	if value == nil {
		return "-"
	}
	return *value
}
