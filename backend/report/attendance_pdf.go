package report

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	"github.com/kappannotavio/ponto-eletronico/model"
)

// Mesma tolerancia usada no calculo de atraso e hora extra.
const toleranceMinutes = 10

type EmployeeReport struct {
	Employee model.Employee
	// Days em ordem cronologica.
	Days []model.Attendance
}

type AttendanceReport struct {
	Period      string
	GeneratedAt time.Time
	Employees   []EmployeeReport
}

const (
	pageMarginX   = 15.0
	pageMarginTop = 15.0
	pageBottom    = 20.0
	rowHeight     = 6.0
)

type column struct {
	title string
	width float64
	align string
}

var columns = []column{
	{"Data", 21, "C"},
	{"Dia", 11, "C"},
	{"Entrada", 18, "C"},
	{"Intervalo", 40, "C"},
	{"Saída", 18, "C"},
	{"Trabalhado", 22, "C"},
	{"Situação", 50, "L"},
}

var weekdays = []string{"dom", "seg", "ter", "qua", "qui", "sex", "sáb"}

// writer junta o documento e o tradutor UTF-8 -> cp1252 das fontes padrao.
type writer struct {
	pdf *fpdf.Fpdf
	tr  func(string) string
}

// WriteAttendancePDF gera o espelho de ponto: uma secao por funcionario,
// cada uma comecando em pagina nova.
func WriteAttendancePDF(out io.Writer, report AttendanceReport) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	w := &writer{pdf: pdf, tr: pdf.UnicodeTranslatorFromDescriptor("")}

	pdf.SetTitle(w.tr("Espelho de ponto"), false)
	pdf.SetMargins(pageMarginX, pageMarginTop, pageMarginX)
	pdf.SetAutoPageBreak(false, pageBottom)
	pdf.AliasNbPages("{nb}")
	pdf.SetFooterFunc(func() { w.footer(report.GeneratedAt) })

	if len(report.Employees) == 0 {
		pdf.AddPage()
		w.title(report.Period)
		w.setFont("", 9)
		pdf.CellFormat(0, 8, w.tr("Nenhuma marcação encontrada para o filtro selecionado."), "", 1, "L", false, 0, "")
	}

	for _, employee := range report.Employees {
		w.employeeSection(employee, report.Period, report.GeneratedAt)
	}

	return pdf.Output(out)
}

func (w *writer) setFont(style string, size float64) {
	w.pdf.SetFont("Helvetica", style, size)
	w.pdf.SetTextColor(0, 0, 0)
}

func (w *writer) title(period string) {
	pdf := w.pdf

	w.setFont("B", 12)
	pdf.CellFormat(100, 6, w.tr("ESPELHO DE PONTO"), "", 0, "L", false, 0, "")
	w.setFont("", 9)
	pdf.CellFormat(0, 6, w.tr("Período: "+period), "", 1, "R", false, 0, "")

	pdf.SetLineWidth(0.4)
	y := pdf.GetY() + 1.5
	pdf.Line(pageMarginX, y, 210-pageMarginX, y)
	pdf.SetY(y + 3)
}

func (w *writer) employeeSection(report EmployeeReport, period string, now time.Time) {
	pdf := w.pdf
	employee := report.Employee

	pdf.AddPage()
	w.title(period)

	jobTitle := employee.JobTitle
	if jobTitle == "" {
		jobTitle = "—"
	}
	schedule := fmt.Sprintf("entrada %s, %dh por dia, intervalo de %d min",
		employee.StartTime, employee.DailyHours, employee.LunchBreak)

	w.field("Funcionário", employee.FullName, 95)
	w.field("Cartão", employee.CardTag, 0)
	pdf.Ln(5)
	w.field("Função", jobTitle, 95)
	w.field("Jornada", schedule, 0)
	pdf.Ln(8)

	w.tableHeader()

	if len(report.Days) == 0 {
		w.setFont("I", 8.5)
		pdf.CellFormat(0, rowHeight, w.tr("Nenhuma marcação no período."), "LRB", 1, "C", false, 0, "")
	}

	today := now.Format("2006-01-02")
	var totals summary

	for _, day := range report.Days {
		if pdf.GetY()+rowHeight > 297-pageBottom {
			pdf.AddPage()
			w.setFont("", 8)
			pdf.CellFormat(0, 5, w.tr(employee.FullName+" (continuação)"), "", 1, "L", false, 0, "")
			w.tableHeader()
		}

		totals.add(employee, day, today)
		w.dayRow(employee, day, today)
	}

	w.summary(totals)
	w.signatures(employee.FullName)
}

func (w *writer) field(label, value string, width float64) {
	pdf := w.pdf

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(90, 90, 90)
	labelText := w.tr(label + ": ")
	labelWidth := pdf.GetStringWidth(labelText) + 0.5
	pdf.CellFormat(labelWidth, 5, labelText, "", 0, "L", false, 0, "")

	w.setFont("B", 9)
	if width > 0 {
		width -= labelWidth
	}
	pdf.CellFormat(width, 5, w.tr(value), "", 0, "L", false, 0, "")
}

func (w *writer) tableHeader() {
	pdf := w.pdf

	w.setFont("B", 8)
	pdf.SetFillColor(225, 225, 225)
	pdf.SetDrawColor(120, 120, 120)
	pdf.SetLineWidth(0.2)

	for _, col := range columns {
		title := col.title
		if col.align == "L" {
			title = " " + title
		}
		pdf.CellFormat(col.width, rowHeight, w.tr(title), "1", 0, col.align, true, 0, "")
	}
	pdf.Ln(-1)
}

func (w *writer) dayRow(employee model.Employee, day model.Attendance, today string) {
	pdf := w.pdf
	times := markedTimes(day)

	entry, exit, middle := "—", "—", ""
	if len(times) > 0 {
		entry = times[0]
	}
	if len(times) >= 2 && len(times)%2 == 0 {
		exit = times[len(times)-1]
		middle = joinBreaks(times[1 : len(times)-1])
	} else if len(times) > 1 {
		middle = joinBreaks(times[1:])
	}
	if middle == "" {
		middle = "—"
	}

	date, _ := time.ParseInLocation("2006-01-02", day.Date, time.Local)
	worked := "—"
	if day.WorkedMinutes > 0 {
		worked = hours(day.WorkedMinutes)
	}

	values := []string{
		date.Format("02/01/2006"),
		weekdays[date.Weekday()],
		entry,
		middle,
		exit,
		worked,
		situation(employee, day, today),
	}

	w.setFont("", 8.5)
	for i, col := range columns {
		if col.align == "L" {
			values[i] = " " + values[i]
		}
		pdf.CellFormat(col.width, rowHeight, w.tr(values[i]), "1", 0, col.align, false, 0, "")
	}
	pdf.Ln(-1)
}

type summary struct {
	days        int
	worked      int
	lateDays    int
	lateMinutes int
	overtime    int
	incomplete  int
	manual      bool
}

func (s *summary) add(employee model.Employee, day model.Attendance, today string) {
	s.days++
	s.worked += day.WorkedMinutes
	if late := lateMinutes(employee, day); late > 0 {
		s.lateDays++
		s.lateMinutes += late
	}
	s.overtime += overtimeMinutes(employee, day)
	if len(day.Times)%2 == 1 && day.Date != today {
		s.incomplete++
	}
	for _, record := range day.Records {
		if record.Source == model.SourceManual {
			s.manual = true
		}
	}
}

func (w *writer) summary(s summary) {
	pdf := w.pdf

	if pdf.GetY()+48 > 297-pageBottom {
		pdf.AddPage()
	}

	pdf.Ln(5)
	w.setFont("B", 9)
	pdf.CellFormat(0, 5, w.tr("Resumo do período"), "", 1, "L", false, 0, "")
	pdf.Ln(1)

	late := fmt.Sprint(s.lateDays)
	if s.lateDays > 0 {
		late += fmt.Sprintf(" (%d min)", s.lateMinutes)
	}

	items := [][2]string{
		{"Dias com marcação", fmt.Sprint(s.days)},
		{"Horas trabalhadas", hours(s.worked)},
		{"Horas extras", hours(s.overtime)},
		{"Atrasos", late},
		{"Dias incompletos", fmt.Sprint(s.incomplete)},
	}

	pdf.SetDrawColor(120, 120, 120)
	for _, item := range items {
		w.setFont("", 8.5)
		pdf.CellFormat(45, 5.5, w.tr(" "+item[0]), "1", 0, "L", false, 0, "")
		w.setFont("B", 8.5)
		pdf.CellFormat(25, 5.5, w.tr(item[1]), "1", 1, "C", false, 0, "")
	}

	pdf.Ln(2)
	pdf.SetFont("Helvetica", "", 7.5)
	pdf.SetTextColor(90, 90, 90)
	notes := fmt.Sprintf("Atraso e hora extra consideram tolerância de %d minutos por dia.", toleranceMinutes)
	if s.manual {
		notes = "* Marcação incluída ou corrigida manualmente.   " + notes
	}
	pdf.CellFormat(0, 4, w.tr(notes), "", 1, "L", false, 0, "")
}

func (w *writer) signatures(employeeName string) {
	pdf := w.pdf

	if pdf.GetY()+30 > 297-pageBottom {
		pdf.AddPage()
	}

	y := pdf.GetY() + 20
	lineWidth := 75.0
	rightX := 210 - pageMarginX - lineWidth

	pdf.SetDrawColor(0, 0, 0)
	pdf.SetLineWidth(0.2)
	pdf.Line(pageMarginX, y, pageMarginX+lineWidth, y)
	pdf.Line(rightX, y, rightX+lineWidth, y)

	w.setFont("", 8)
	pdf.SetXY(pageMarginX, y+1)
	pdf.CellFormat(lineWidth, 4, w.tr(employeeName), "", 0, "C", false, 0, "")
	pdf.SetXY(rightX, y+1)
	pdf.CellFormat(lineWidth, 4, w.tr("Responsável"), "", 1, "C", false, 0, "")
}

func (w *writer) footer(generatedAt time.Time) {
	pdf := w.pdf

	pdf.SetY(-12)
	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(110, 110, 110)
	pdf.CellFormat(90, 4, w.tr("Emitido em "+generatedAt.Format("02/01/2006 às 15:04")), "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 4, w.tr(fmt.Sprintf("Página %d de {nb}", pdf.PageNo())), "", 0, "R", false, 0, "")
}

// markedTimes devolve os horarios do dia, com "*" nas batidas manuais.
func markedTimes(day model.Attendance) []string {
	times := make([]string, len(day.Times))
	for i, value := range day.Times {
		times[i] = value
		if i < len(day.Records) && day.Records[i].Source == model.SourceManual {
			times[i] += "*"
		}
	}
	return times
}

// joinBreaks agrupa as batidas do meio em pares: "12:00 - 13:00".
func joinBreaks(times []string) string {
	parts := make([]string, 0, (len(times)+1)/2)
	for i := 0; i < len(times); i += 2 {
		if i+1 < len(times) {
			parts = append(parts, times[i]+" - "+times[i+1])
		} else {
			parts = append(parts, times[i])
		}
	}
	return strings.Join(parts, " / ")
}

func situation(employee model.Employee, day model.Attendance, today string) string {
	parts := make([]string, 0, 2)

	if late := lateMinutes(employee, day); late > 0 {
		parts = append(parts, fmt.Sprintf("Atraso de %d min", late))
	}
	if extra := overtimeMinutes(employee, day); extra > 0 {
		parts = append(parts, "Hora extra "+hours(extra))
	}
	if len(day.Times)%2 == 1 {
		if day.Date == today {
			parts = append(parts, "Em andamento")
		} else {
			parts = append(parts, "Sem saída")
		}
	}

	if len(parts) == 0 {
		return "Normal"
	}
	return strings.Join(parts, ", ")
}

func lateMinutes(employee model.Employee, day model.Attendance) int {
	if !day.Late || day.Entry == nil {
		return 0
	}
	return clockMinutes(*day.Entry) - clockMinutes(employee.StartTime)
}

func overtimeMinutes(employee model.Employee, day model.Attendance) int {
	extra := day.WorkedMinutes - employee.DailyHours*60
	if len(day.Times)%2 == 1 || extra <= toleranceMinutes {
		return 0
	}
	return extra
}

func clockMinutes(value string) int {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0
	}
	return parsed.Hour()*60 + parsed.Minute()
}

func hours(minutes int) string {
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}
