package repository

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	// Driver SQLite escrito em Go puro: nao precisa de CGO, gcc nem de
	// instalar o SQLite. O banco e um unico arquivo no disco.
	_ "modernc.org/sqlite"
)

// schemaVersion e gravado em PRAGMA user_version. Ao mudar o schema,
// incremente e adicione o passo de migracao em migrate.
const schemaVersion = 1

const schemaV1 = `
CREATE TABLE IF NOT EXISTS employees (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	full_name   TEXT    NOT NULL,
	job_title   TEXT    NOT NULL DEFAULT '',
	whatsapp    TEXT    NOT NULL DEFAULT '',
	start_time  TEXT    NOT NULL,
	daily_hours INTEGER NOT NULL,
	lunch_break INTEGER NOT NULL,
	card_tag    TEXT    NOT NULL UNIQUE,
	pin_hash    TEXT    NOT NULL,
	is_active   INTEGER NOT NULL DEFAULT 1,
	created_at  TEXT    NOT NULL,
	updated_at  TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS time_records (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	employee_id INTEGER NOT NULL REFERENCES employees(id),
	card_tag    TEXT    NOT NULL,
	recorded_at TEXT    NOT NULL,
	source      TEXT    NOT NULL,
	created_at  TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_time_records_employee_recorded
	ON time_records (employee_id, recorded_at);

CREATE INDEX IF NOT EXISTS idx_time_records_recorded
	ON time_records (recorded_at);
`

// OpenSQLite abre (ou cria) o banco no arquivo path e garante que as
// tabelas existem. A pasta do arquivo e criada se nao existir.
func OpenSQLite(path string) (*sql.DB, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("criar pasta do banco: %w", err)
		}
	}

	// Os pragmas vao na DSN para valerem em toda conexao aberta pelo pool.
	// Sem o prefixo "file:" o caminho e usado como esta (funciona com C:\...).
	dsn := path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("abrir banco %s: %w", path, err)
	}

	// SQLite aceita um escritor por vez. Uma conexao so evita "database is
	// locked" e e mais que suficiente para o volume de um relogio de ponto.
	db.SetMaxOpenConns(1)

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("preparar banco %s: %w", path, err)
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}

	if version > schemaVersion {
		return fmt.Errorf("banco na versao %d, mas este programa conhece ate a %d (atualize o programa)", version, schemaVersion)
	}

	if version < 1 {
		if _, err := db.Exec(schemaV1); err != nil {
			return err
		}
	}

	_, err := db.Exec(fmt.Sprintf("PRAGMA user_version = %d", schemaVersion))
	return err
}

// Datas sao gravadas em UTC com largura fixa: legiveis em qualquer visualizador
// de SQLite e comparaveis como texto (necessario para os filtros por periodo).
const sqliteTimeLayout = "2006-01-02T15:04:05.000000000Z"

func formatTime(t time.Time) string {
	return t.UTC().Format(sqliteTimeLayout)
}

func parseTime(value string) (time.Time, error) {
	t, err := time.Parse(sqliteTimeLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("data invalida no banco %q: %w", value, err)
	}

	return t.In(time.Local), nil
}
