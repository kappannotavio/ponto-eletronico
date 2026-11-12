package repository

import (
	"database/sql"
	"errors"

	"github.com/kappannotavio/ponto-eletronico/model"
)

const employeeColumns = `id, full_name, job_title, whatsapp, start_time, daily_hours,
	lunch_break, card_tag, pin_hash, is_active, created_at, updated_at`

type SQLiteEmployeeRepository struct {
	db *sql.DB
}

func NewSQLiteEmployeeRepository(db *sql.DB) *SQLiteEmployeeRepository {
	return &SQLiteEmployeeRepository{db: db}
}

func (r *SQLiteEmployeeRepository) Create(employee model.Employee) (model.Employee, error) {
	result, err := r.db.Exec(`
		INSERT INTO employees (full_name, job_title, whatsapp, start_time, daily_hours,
			lunch_break, card_tag, pin_hash, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		employee.FullName, employee.JobTitle, employee.Whatsapp, employee.StartTime,
		employee.DailyHours, employee.LunchBreak, employee.CardTag, employee.PinHash,
		employee.IsActive, formatTime(employee.CreatedAt), formatTime(employee.UpdatedAt),
	)
	if err != nil {
		return model.Employee{}, err
	}

	employee.ID, err = result.LastInsertId()
	if err != nil {
		return model.Employee{}, err
	}

	return employee, nil
}

func (r *SQLiteEmployeeRepository) Update(employee model.Employee) (model.Employee, error) {
	result, err := r.db.Exec(`
		UPDATE employees SET full_name = ?, job_title = ?, whatsapp = ?, start_time = ?,
			daily_hours = ?, lunch_break = ?, card_tag = ?, pin_hash = ?, is_active = ?,
			created_at = ?, updated_at = ?
		WHERE id = ?`,
		employee.FullName, employee.JobTitle, employee.Whatsapp, employee.StartTime,
		employee.DailyHours, employee.LunchBreak, employee.CardTag, employee.PinHash,
		employee.IsActive, formatTime(employee.CreatedAt), formatTime(employee.UpdatedAt),
		employee.ID,
	)
	if err != nil {
		return model.Employee{}, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return model.Employee{}, err
	}
	if affected == 0 {
		return model.Employee{}, ErrNotFound
	}

	return employee, nil
}

func (r *SQLiteEmployeeRepository) FindByID(id int64) (model.Employee, bool, error) {
	return r.findOne(`SELECT `+employeeColumns+` FROM employees WHERE id = ?`, id)
}

func (r *SQLiteEmployeeRepository) FindByCardTag(cardTag string) (model.Employee, bool, error) {
	return r.findOne(`SELECT `+employeeColumns+` FROM employees WHERE card_tag = ?`, cardTag)
}

func (r *SQLiteEmployeeRepository) List() ([]model.Employee, error) {
	rows, err := r.db.Query(`SELECT ` + employeeColumns + ` FROM employees ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	employees := make([]model.Employee, 0)
	for rows.Next() {
		employee, err := scanEmployee(rows)
		if err != nil {
			return nil, err
		}
		employees = append(employees, employee)
	}

	return employees, rows.Err()
}

func (r *SQLiteEmployeeRepository) findOne(query string, arg any) (model.Employee, bool, error) {
	employee, err := scanEmployee(r.db.QueryRow(query, arg))
	if errors.Is(err, sql.ErrNoRows) {
		return model.Employee{}, false, nil
	}
	if err != nil {
		return model.Employee{}, false, err
	}

	return employee, true, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEmployee(row rowScanner) (model.Employee, error) {
	var (
		employee  model.Employee
		createdAt string
		updatedAt string
	)

	err := row.Scan(
		&employee.ID, &employee.FullName, &employee.JobTitle, &employee.Whatsapp,
		&employee.StartTime, &employee.DailyHours, &employee.LunchBreak, &employee.CardTag,
		&employee.PinHash, &employee.IsActive, &createdAt, &updatedAt,
	)
	if err != nil {
		return model.Employee{}, err
	}

	if employee.CreatedAt, err = parseTime(createdAt); err != nil {
		return model.Employee{}, err
	}
	if employee.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return model.Employee{}, err
	}

	return employee, nil
}
