package repository

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/kappannotavio/ponto-eletronico/model"
)

const timeRecordColumns = `id, employee_id, card_tag, recorded_at, source, created_at`

type SQLiteTimeRecordRepository struct {
	db *sql.DB
}

func NewSQLiteTimeRecordRepository(db *sql.DB) *SQLiteTimeRecordRepository {
	return &SQLiteTimeRecordRepository{db: db}
}

func (r *SQLiteTimeRecordRepository) Save(record model.TimeRecord) (model.TimeRecord, error) {
	result, err := r.db.Exec(`
		INSERT INTO time_records (employee_id, card_tag, recorded_at, source, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		record.EmployeeID, record.CardTag, formatTime(record.RecordedAt),
		record.Source, formatTime(record.CreatedAt),
	)
	if err != nil {
		return model.TimeRecord{}, err
	}

	record.ID, err = result.LastInsertId()
	if err != nil {
		return model.TimeRecord{}, err
	}

	return record, nil
}

func (r *SQLiteTimeRecordRepository) FindByID(id int64) (model.TimeRecord, bool, error) {
	record, err := scanTimeRecord(r.db.QueryRow(
		`SELECT `+timeRecordColumns+` FROM time_records WHERE id = ?`, id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return model.TimeRecord{}, false, nil
	}
	if err != nil {
		return model.TimeRecord{}, false, err
	}

	return record, true, nil
}

func (r *SQLiteTimeRecordRepository) Delete(id int64) error {
	result, err := r.db.Exec(`DELETE FROM time_records WHERE id = ?`, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

// List retorna as batidas ordenadas da mais antiga para a mais recente.
func (r *SQLiteTimeRecordRepository) List(filter TimeRecordFilter) ([]model.TimeRecord, error) {
	conditions := make([]string, 0, 3)
	args := make([]any, 0, 3)

	if filter.EmployeeID != 0 {
		conditions = append(conditions, "employee_id = ?")
		args = append(args, filter.EmployeeID)
	}
	if !filter.From.IsZero() {
		conditions = append(conditions, "recorded_at >= ?")
		args = append(args, formatTime(filter.From))
	}
	if !filter.To.IsZero() {
		conditions = append(conditions, "recorded_at < ?")
		args = append(args, formatTime(filter.To))
	}

	query := `SELECT ` + timeRecordColumns + ` FROM time_records`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY recorded_at, id"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]model.TimeRecord, 0)
	for rows.Next() {
		record, err := scanTimeRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

func scanTimeRecord(row rowScanner) (model.TimeRecord, error) {
	var (
		record     model.TimeRecord
		recordedAt string
		createdAt  string
	)

	err := row.Scan(&record.ID, &record.EmployeeID, &record.CardTag, &recordedAt, &record.Source, &createdAt)
	if err != nil {
		return model.TimeRecord{}, err
	}

	if record.RecordedAt, err = parseTime(recordedAt); err != nil {
		return model.TimeRecord{}, err
	}
	if record.CreatedAt, err = parseTime(createdAt); err != nil {
		return model.TimeRecord{}, err
	}

	return record, nil
}
