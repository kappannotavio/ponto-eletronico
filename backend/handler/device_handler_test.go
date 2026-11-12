package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kappannotavio/ponto-eletronico/middleware"
	"github.com/kappannotavio/ponto-eletronico/repository"
	"github.com/kappannotavio/ponto-eletronico/service"
)

func TestDeviceListEmployees(t *testing.T) {
	employees := service.NewEmployeeService(repository.NewInMemoryEmployeeRepository())
	ana, err := employees.Create(service.EmployeeInput{FullName: "Ana", CardTag: "a1:b2:c3:d4", Pin: "1234"})
	if err != nil {
		t.Fatal(err)
	}
	bruno, err := employees.Create(service.EmployeeInput{FullName: "Bruno", CardTag: "B1:B2:B3:B4", Pin: "4321"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := employees.SetActive(bruno.ID, false); err != nil {
		t.Fatal(err)
	}

	h := NewDeviceHandler(employees, nil, nil)
	route := middleware.DeviceAuth("chave", h.ListEmployees)

	get := func(key, ifNoneMatch string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/device/employees", nil)
		req.Header.Set("X-Device-Key", key)
		if ifNoneMatch != "" {
			req.Header.Set("If-None-Match", ifNoneMatch)
		}
		rec := httptest.NewRecorder()
		route(rec, req)
		return rec
	}

	rec := get("chave", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}

	var body deviceEmployeeListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Employees) != 2 {
		t.Fatalf("esperado 2 funcionarios (ativos e desativados), veio %+v", body.Employees)
	}
	first := body.Employees[0]
	if first.CardTag != "A1:B2:C3:D4" || first.FullName != "Ana" || !first.IsActive || first.PinHash != ana.PinHash {
		t.Fatalf("funcionario diferente: %+v", first)
	}
	if body.Employees[1].IsActive {
		t.Fatal("Bruno deveria vir desativado")
	}

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("resposta sem ETag")
	}

	if rec := get("chave", etag); rec.Code != http.StatusNotModified || rec.Body.Len() != 0 {
		t.Fatalf("mesma lista deveria dar 304 sem corpo, veio %d (%q)", rec.Code, rec.Body)
	}

	if _, err := employees.Update(ana.ID, service.EmployeeInput{FullName: "Ana Souza", CardTag: ana.CardTag, StartTime: "08:00", DailyHours: 8}); err != nil {
		t.Fatal(err)
	}
	if rec := get("chave", etag); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Ana Souza") {
		t.Fatalf("lista alterada deveria dar 200 com o nome novo, veio %d", rec.Code)
	}

	rec = get("errada", "")
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), `"code":"invalid_device_key"`) {
		t.Fatalf("chave errada deveria dar 401 com code, veio %d %s", rec.Code, rec.Body)
	}
}
