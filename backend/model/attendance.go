package model

const (
	StatusNormal     = "normal"
	StatusInProgress = "in_progress"
	StatusLate       = "late"
	StatusOvertime   = "overtime"
	StatusIncomplete = "incomplete"
)

// Attendance e o resumo de um dia de trabalho de um funcionario,
// calculado a partir das batidas (TimeRecord) daquele dia.
type Attendance struct {
	EmployeeID    int64        `json:"employeeId"`
	EmployeeName  string       `json:"employeeName"`
	Date          string       `json:"date"`
	Entry         *string      `json:"entry"`
	Exit          *string      `json:"exit"`
	WorkedMinutes int          `json:"workedMinutes"`
	Late          bool         `json:"late"`
	Status        string       `json:"status"`
	Times         []string     `json:"times"`
	Records       []TimeRecord `json:"records"`
}

type Dashboard struct {
	Date         string       `json:"date"`
	ActiveCount  int          `json:"activeCount"`
	PresentCount int          `json:"presentCount"`
	LateCount    int          `json:"lateCount"`
	AbsentCount  int          `json:"absentCount"`
	Today        []Attendance `json:"today"`
}
