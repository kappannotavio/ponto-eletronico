package main

import (
	"log"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
	_ "time/tzdata"

	"github.com/kappannotavio/ponto-eletronico/config"
	"github.com/kappannotavio/ponto-eletronico/handler"
	"github.com/kappannotavio/ponto-eletronico/middleware"
	"github.com/kappannotavio/ponto-eletronico/repository"
	"github.com/kappannotavio/ponto-eletronico/service"
	"github.com/kappannotavio/ponto-eletronico/web"
)

func main() {
	cfg := config.Load()

	location, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.Fatalf("invalid APP_TIMEZONE %q: %v", cfg.Timezone, err)
	}
	time.Local = location

	db, err := repository.OpenSQLite(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()
	log.Printf("database: %s", absPath(cfg.DatabasePath))

	employeeRepository := repository.NewSQLiteEmployeeRepository(db)
	timeRecordRepository := repository.NewSQLiteTimeRecordRepository(db)

	employeeService := service.NewEmployeeService(employeeRepository)
	timeRecordService := service.NewTimeRecordService(timeRecordRepository, employeeService)
	cardReadService := service.NewCardReadService(employeeService)

	if cfg.SeedDemoData {
		seeded, err := seedDemoData(employeeService, timeRecordRepository)
		if err != nil {
			log.Fatalf("seed demo data: %v", err)
		}
		if seeded {
			log.Println("demo data loaded")
		} else {
			log.Println("demo data skipped: database already has employees")
		}
	}

	employeeHandler := handler.NewEmployeeHandler(employeeService)
	timeRecordHandler := handler.NewTimeRecordHandler(timeRecordService)
	reportHandler := handler.NewReportHandler(employeeService, timeRecordService)
	cardReadHandler := handler.NewCardReadHandler(cardReadService)
	deviceHandler := handler.NewDeviceHandler(employeeService, timeRecordService, cardReadService)

	device := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware.DeviceAuth(cfg.DeviceAPIKey, next)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Painel (frontend)
	mux.HandleFunc("GET /api/v1/dashboard", timeRecordHandler.Dashboard)

	mux.HandleFunc("GET /api/v1/employees", employeeHandler.List)
	mux.HandleFunc("POST /api/v1/employees", employeeHandler.Create)
	mux.HandleFunc("GET /api/v1/employees/{id}", employeeHandler.Get)
	mux.HandleFunc("PUT /api/v1/employees/{id}", employeeHandler.Update)
	mux.HandleFunc("PATCH /api/v1/employees/{id}/status", employeeHandler.UpdateStatus)

	mux.HandleFunc("GET /api/v1/time-records", timeRecordHandler.List)
	mux.HandleFunc("POST /api/v1/time-records", timeRecordHandler.Create)
	mux.HandleFunc("DELETE /api/v1/time-records/{id}", timeRecordHandler.Delete)

	mux.HandleFunc("GET /api/v1/attendance", timeRecordHandler.Attendance)
	mux.HandleFunc("PUT /api/v1/attendance/{employeeId}/{date}", timeRecordHandler.ReplaceDay)
	mux.HandleFunc("GET /api/v1/reports/attendance.csv", reportHandler.AttendanceCSV)
	mux.HandleFunc("GET /api/v1/reports/attendance.pdf", reportHandler.AttendancePDF)

	mux.HandleFunc("POST /api/v1/card-reads", cardReadHandler.Create)
	mux.HandleFunc("GET /api/v1/card-reads/{id}", cardReadHandler.Get)
	mux.HandleFunc("DELETE /api/v1/card-reads/{id}", cardReadHandler.Cancel)

	// Relogio de ponto (ESP32) - exigem o header X-Device-Key
	mux.HandleFunc("GET /api/v1/device/ping", device(deviceHandler.Ping))
	mux.HandleFunc("GET /api/v1/device/employees", device(deviceHandler.ListEmployees))
	mux.HandleFunc("GET /api/v1/device/employees/{cardTag}", device(deviceHandler.FindEmployeeByTag))
	mux.HandleFunc("POST /api/v1/device/punches", device(deviceHandler.Punch))
	mux.HandleFunc("GET /api/v1/device/card-reads/pending", device(deviceHandler.PendingCardRead))
	mux.HandleFunc("POST /api/v1/device/card-reads/{id}", device(deviceHandler.CompleteCardRead))

	// Painel embutido (executavel gerado com ./dev.sh build). Qualquer caminho
	// que nao seja uma rota da API acima cai aqui.
	mux.Handle("/", web.Handler())

	app := middleware.Logging(middleware.CORS(cfg.CORSAllowedOrigins)(mux))

	address := ":" + cfg.Port
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("nao foi possivel usar a porta %s (o sistema ja esta aberto?): %v", cfg.Port, err)
	}

	log.Printf("server running on %s (timezone %s)", address, cfg.Timezone)
	mostrarEnderecos(cfg.Port)

	if web.Disponivel() && cfg.OpenBrowser {
		abrirNavegador("http://localhost:" + cfg.Port)
	}

	if err := http.Serve(listener, app); err != nil {
		log.Fatal(err)
	}
}

// mostrarEnderecos imprime o endereco do painel e o que deve ser configurado
// no relogio (comando "servidor" do ESP32 ou config.txt do cartao SD).
func mostrarEnderecos(port string) {
	if web.Disponivel() {
		log.Printf("painel: http://localhost:%s", port)
	} else {
		log.Printf("painel: use ./dev.sh e abra http://localhost:4200 (este backend nao tem o painel embutido)")
	}

	interfaces, err := net.InterfaceAddrs()
	if err != nil {
		return
	}
	for _, endereco := range interfaces {
		ip, ok := endereco.(*net.IPNet)
		if !ok || ip.IP.IsLoopback() || ip.IP.To4() == nil {
			continue
		}
		log.Printf("relogio (ESP32): servidor http://%s:%s/api/v1", ip.IP, port)
	}
}

// abrirNavegador abre o painel no navegador padrao (no Windows, ao dar dois
// cliques no ponto.exe). Desligue com ABRIR_NAVEGADOR=false.
func abrirNavegador(url string) {
	var comando *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		comando = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		comando = exec.Command("open", url)
	default:
		comando = exec.Command("xdg-open", url)
	}

	if err := comando.Start(); err != nil {
		log.Printf("abra o painel no navegador: %s", url)
		return
	}
	go comando.Wait()
}

func absPath(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}
