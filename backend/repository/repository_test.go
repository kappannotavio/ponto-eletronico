package repository

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/kappannotavio/ponto-eletronico/model"
)

// Os mesmos cenarios rodam nas duas implementacoes para garantir que o
// SQLite se comporta igual a versao em memoria usada nos testes de service.

type repositories struct {
	employees   EmployeeRepository
	timeRecords TimeRecordRepository
}

func implementations(t *testing.T) map[string]func(t *testing.T) repositories {
	return map[string]func(t *testing.T) repositories{
		"memory": func(t *testing.T) repositories {
			return repositories{NewInMemoryEmployeeRepository(), NewInMemoryTimeRecordRepository()}
		},
		"sqlite": func(t *testing.T) repositories {
			db := openTestDB(t, filepath.Join(t.TempDir(), "ponto.db"))
			return repositories{NewSQLiteEmployeeRepository(db), NewSQLiteTimeRecordRepository(db)}
		},
	}
}

func openTestDB(t *testing.T, path string) *sql.DB {
	t.Helper()

	db, err := OpenSQLite(path)
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	return db
}

var saoPaulo = func() *time.Location {
	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return time.FixedZone("-03", -3*60*60)
	}
	return location
}()

func newEmployee(name, tag string) model.Employee {
	now := time.Date(2026, 11, 10, 9, 0, 0, 123456789, saoPaulo)

	return model.Employee{
		FullName:   name,
		JobTitle:   "Caixa",
		Whatsapp:   "11999999999",
		StartTime:  "08:00",
		DailyHours: 8,
		LunchBreak: 60,
		CardTag:    tag,
		PinHash:    "salt:hash",
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func assertSameEmployee(t *testing.T, got, want model.Employee) {
	t.Helper()

	if !got.CreatedAt.Equal(want.CreatedAt) || !got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Fatalf("datas diferentes: got %v/%v, want %v/%v", got.CreatedAt, got.UpdatedAt, want.CreatedAt, want.UpdatedAt)
	}
	got.CreatedAt, got.UpdatedAt = want.CreatedAt, want.UpdatedAt
	if got != want {
		t.Fatalf("funcionario diferente:\n got %+v\nwant %+v", got, want)
	}
}

func TestEmployeeRepository(t *testing.T) {
	for name, open := range implementations(t) {
		t.Run(name, func(t *testing.T) {
			repos := open(t)

			ana, err := repos.employees.Create(newEmployee("Ana", "AA:BB:CC:01"))
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			bruno, err := repos.employees.Create(newEmployee("Bruno", "AA:BB:CC:02"))
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			if ana.ID == 0 || bruno.ID == ana.ID {
				t.Fatalf("ids invalidos: %d, %d", ana.ID, bruno.ID)
			}

			found, ok, err := repos.employees.FindByID(ana.ID)
			if err != nil || !ok {
				t.Fatalf("FindByID: ok=%v err=%v", ok, err)
			}
			assertSameEmployee(t, found, ana)

			found, ok, err = repos.employees.FindByCardTag("AA:BB:CC:02")
			if err != nil || !ok {
				t.Fatalf("FindByCardTag: ok=%v err=%v", ok, err)
			}
			assertSameEmployee(t, found, bruno)

			if _, ok, err := repos.employees.FindByCardTag("NAO:EXISTE"); err != nil || ok {
				t.Fatalf("FindByCardTag inexistente: ok=%v err=%v", ok, err)
			}
			if _, ok, err := repos.employees.FindByID(999); err != nil || ok {
				t.Fatalf("FindByID inexistente: ok=%v err=%v", ok, err)
			}

			ana.FullName = "Ana Souza"
			ana.IsActive = false
			ana.CardTag = "AA:BB:CC:03"
			ana.UpdatedAt = ana.UpdatedAt.Add(time.Hour)
			if _, err := repos.employees.Update(ana); err != nil {
				t.Fatalf("Update: %v", err)
			}
			found, _, _ = repos.employees.FindByID(ana.ID)
			assertSameEmployee(t, found, ana)

			if _, err := repos.employees.Update(model.Employee{ID: 999}); !errors.Is(err, ErrNotFound) {
				t.Fatalf("Update inexistente: esperado ErrNotFound, veio %v", err)
			}

			list, err := repos.employees.List()
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(list) != 2 || list[0].ID != ana.ID || list[1].ID != bruno.ID {
				t.Fatalf("List fora de ordem: %+v", list)
			}
		})
	}
}

func TestTimeRecordRepository(t *testing.T) {
	for name, open := range implementations(t) {
		t.Run(name, func(t *testing.T) {
			repos := open(t)

			ana, _ := repos.employees.Create(newEmployee("Ana", "AA:BB:CC:01"))
			bruno, _ := repos.employees.Create(newEmployee("Bruno", "AA:BB:CC:02"))

			day := time.Date(2026, 11, 10, 0, 0, 0, 0, saoPaulo)
			save := func(employee model.Employee, at time.Time) model.TimeRecord {
				t.Helper()
				record, err := repos.timeRecords.Save(model.TimeRecord{
					EmployeeID: employee.ID,
					CardTag:    employee.CardTag,
					RecordedAt: at,
					Source:     model.SourceDevice,
					CreatedAt:  at,
				})
				if err != nil {
					t.Fatalf("Save: %v", err)
				}
				return record
			}

			// Salvas fora de ordem de proposito. 22:30 em Sao Paulo ja e o dia
			// seguinte em UTC: o filtro precisa respeitar o instante, nao o texto local.
			exit := save(ana, day.Add(17*time.Hour))
			entry := save(ana, day.Add(8*time.Hour))
			late := save(ana, day.Add(22*time.Hour+30*time.Minute))
			save(bruno, day.Add(9*time.Hour))
			save(ana, day.AddDate(0, 0, 1).Add(8*time.Hour))

			found, ok, err := repos.timeRecords.FindByID(entry.ID)
			if err != nil || !ok {
				t.Fatalf("FindByID: ok=%v err=%v", ok, err)
			}
			if !found.RecordedAt.Equal(entry.RecordedAt) || found.EmployeeID != ana.ID || found.Source != model.SourceDevice {
				t.Fatalf("FindByID diferente: %+v", found)
			}

			records, err := repos.timeRecords.List(TimeRecordFilter{
				EmployeeID: ana.ID,
				From:       day,
				To:         day.AddDate(0, 0, 1),
			})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			wantIDs := []int64{entry.ID, exit.ID, late.ID}
			if len(records) != len(wantIDs) {
				t.Fatalf("List: esperado %d batidas, veio %d (%+v)", len(wantIDs), len(records), records)
			}
			for i, id := range wantIDs {
				if records[i].ID != id {
					t.Fatalf("List fora de ordem: posicao %d esperado id %d, veio %d", i, id, records[i].ID)
				}
			}

			all, _ := repos.timeRecords.List(TimeRecordFilter{})
			if len(all) != 5 {
				t.Fatalf("List sem filtro: esperado 5, veio %d", len(all))
			}

			if err := repos.timeRecords.Delete(exit.ID); err != nil {
				t.Fatalf("Delete: %v", err)
			}
			if err := repos.timeRecords.Delete(exit.ID); !errors.Is(err, ErrNotFound) {
				t.Fatalf("Delete repetido: esperado ErrNotFound, veio %v", err)
			}
			if _, ok, _ := repos.timeRecords.FindByID(exit.ID); ok {
				t.Fatal("batida apagada ainda aparece")
			}

			// Ids nao sao reaproveitados depois de apagar.
			next := save(bruno, day.Add(18*time.Hour))
			if next.ID <= late.ID {
				t.Fatalf("id reaproveitado: %d", next.ID)
			}
		})
	}
}

func TestSQLiteKeepsDataAfterReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dados", "ponto.db")

	db, err := OpenSQLite(path)
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	employee, err := NewSQLiteEmployeeRepository(db).Create(newEmployee("Ana", "AA:BB:CC:01"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	recordedAt := time.Date(2026, 11, 10, 8, 2, 0, 0, saoPaulo)
	if _, err := NewSQLiteTimeRecordRepository(db).Save(model.TimeRecord{
		EmployeeID: employee.ID, CardTag: employee.CardTag, RecordedAt: recordedAt,
		Source: model.SourceDevice, CreatedAt: recordedAt,
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	db.Close()

	db = openTestDB(t, path)
	found, ok, err := NewSQLiteEmployeeRepository(db).FindByCardTag("AA:BB:CC:01")
	if err != nil || !ok || found.FullName != "Ana" {
		t.Fatalf("funcionario nao persistiu: ok=%v err=%v %+v", ok, err, found)
	}
	records, err := NewSQLiteTimeRecordRepository(db).List(TimeRecordFilter{})
	if err != nil || len(records) != 1 || !records[0].RecordedAt.Equal(recordedAt) {
		t.Fatalf("batida nao persistiu: err=%v %+v", err, records)
	}
}

func TestSQLiteRejectsInvalidData(t *testing.T) {
	db := openTestDB(t, filepath.Join(t.TempDir(), "ponto.db"))
	employees := NewSQLiteEmployeeRepository(db)

	if _, err := employees.Create(newEmployee("Ana", "AA:BB:CC:01")); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := employees.Create(newEmployee("Outra", "AA:BB:CC:01")); err == nil {
		t.Fatal("tag duplicada deveria ser recusada pelo banco")
	}

	_, err := NewSQLiteTimeRecordRepository(db).Save(model.TimeRecord{
		EmployeeID: 999, CardTag: "X", RecordedAt: time.Now(), Source: model.SourceDevice, CreatedAt: time.Now(),
	})
	if err == nil {
		t.Fatal("batida de funcionario inexistente deveria ser recusada (foreign key)")
	}
}
