// Relogio virtual: simula o ESP32 no terminal, usando as mesmas rotas
// /api/v1/device/* do firmware. Serve para testar o sistema sem hardware.
//
//	go run ./cmd/relogio-virtual
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const pollInterval = 2 * time.Second

type client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// request devolve o status HTTP e decodifica o corpo em out (se houver).
// status 0 significa que o servidor nao respondeu.
func (c *client) request(method, path string, body, out any) (int, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-Device-Key", c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if out != nil && resp.StatusCode != http.StatusNoContent {
		json.NewDecoder(resp.Body).Decode(out)
	}

	return resp.StatusCode, nil
}

type apiResponse struct {
	Error        string `json:"error"`
	ID           any    `json:"id"`
	FullName     string `json:"fullName"`
	IsActive     bool   `json:"isActive"`
	EmployeeName string `json:"employeeName"`
	Type         string `json:"type"`
	PunchNumber  int    `json:"punchNumber"`
	CardTag      string `json:"cardTag"`
	Record       struct {
		RecordedAt time.Time `json:"recordedAt"`
	} `json:"record"`
	AssignedTo *struct {
		FullName string `json:"fullName"`
	} `json:"assignedTo"`
}

type employee struct {
	FullName string `json:"fullName"`
	CardTag  string `json:"cardTag"`
	IsActive bool   `json:"isActive"`
}

// clock guarda o estado do relogio, compartilhado entre o terminal
// e a goroutine que consulta os pedidos de leitura do painel.
type clock struct {
	mu          sync.Mutex
	api         *client
	pendingRead string
	tag         string
	name        string
	attempts    int
}

func main() {
	api := &client{
		baseURL: env("API_BASE_URL", "http://localhost:8080/api/v1"),
		apiKey:  env("DEVICE_API_KEY", "dev-device-key"),
		http:    &http.Client{Timeout: 5 * time.Second},
	}
	c := &clock{api: api}

	fmt.Println("================================")
	fmt.Println("   RELOGIO DE PONTO VIRTUAL")
	fmt.Println("================================")
	fmt.Println("Servidor:", api.baseURL)

	if status, err := api.request("GET", "/device/ping", nil, nil); err != nil {
		fmt.Println("! Servidor fora do ar. Rode o backend: ./dev.sh backend")
	} else if status == http.StatusUnauthorized {
		fmt.Println("! DEVICE_API_KEY diferente da do backend.")
	} else {
		fmt.Println("Servidor online.")
	}

	printHelp()
	c.prompt()

	go c.pollCardReads()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		c.handle(strings.TrimSpace(scanner.Text()))
	}
}

func printHelp() {
	fmt.Println()
	fmt.Println("Digite a TAG do cartao e depois a SENHA, como no relogio.")
	fmt.Println("Comandos:")
	fmt.Println("  lista   mostra os funcionarios e suas tags")
	fmt.Println("  c       cancela a operacao atual")
	fmt.Println("  ajuda   mostra esta tela")
	fmt.Println("  sair    fecha o relogio")
}

func (c *clock) prompt() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.promptLocked()
}

func (c *clock) promptLocked() {
	fmt.Println()
	switch {
	case c.tag != "":
		fmt.Print("Senha (4 numeros): ")
	case c.pendingRead != "":
		fmt.Print("[PAINEL] Digite a tag a vincular ao funcionario: ")
	default:
		fmt.Print("Aproxime o cartao (digite a tag): ")
	}
}

func (c *clock) handle(input string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer c.promptLocked()

	switch strings.ToLower(input) {
	case "":
		return
	case "ajuda", "help":
		printHelp()
		return
	case "lista":
		c.listEmployees()
		return
	case "c", "cancelar":
		c.reset()
		fmt.Println("Operacao cancelada.")
		return
	case "sair", "exit":
		os.Exit(0)
	}

	switch {
	case c.tag != "":
		c.sendPunch(input)
	case c.pendingRead != "":
		c.sendCardRead(strings.ToUpper(input))
	default:
		c.startPunch(strings.ToUpper(input))
	}
}

func (c *clock) reset() {
	c.tag = ""
	c.name = ""
	c.attempts = 0
}

func (c *clock) startPunch(tag string) {
	var resp apiResponse
	status, err := c.api.request("GET", "/device/employees/"+url.PathEscape(tag), nil, &resp)

	switch {
	case err != nil:
		fmt.Println("x Servidor indisponivel:", err)
		return
	case status == http.StatusOK && !resp.IsActive:
		bell()
		fmt.Println("x Funcionario desativado. Procure o RH.")
		return
	case status != http.StatusOK:
		bell()
		fmt.Println("x", errorText(status, resp))
		return
	}

	c.tag = tag
	c.name = resp.FullName
	c.attempts = 0
	fmt.Printf("Ola, %s!\n", resp.FullName)
}

func (c *clock) sendPunch(pin string) {
	var resp apiResponse
	status, err := c.api.request("POST", "/device/punches", map[string]string{
		"cardTag": c.tag,
		"pin":     pin,
	}, &resp)

	if err != nil {
		fmt.Println("x Servidor indisponivel:", err)
		c.reset()
		return
	}

	if status == http.StatusUnauthorized {
		c.attempts++
		bell()
		if c.attempts < 3 {
			fmt.Printf("x %s. Tente novamente (%d/3).\n", errorText(status, resp), c.attempts)
			return
		}
		fmt.Println("x Muitas tentativas. Passe o cartao de novo.")
		c.reset()
		return
	}

	if status != http.StatusCreated {
		bell()
		fmt.Println("x Ponto nao registrado:", errorText(status, resp))
		c.reset()
		return
	}

	kind := "ENTRADA"
	if resp.Type == "exit" {
		kind = "SAIDA"
	}
	bell()
	fmt.Println("********** PONTO REGISTRADO **********")
	fmt.Printf("  %s - %s as %s (batida %d do dia)\n",
		resp.EmployeeName, kind, resp.Record.RecordedAt.Local().Format("15:04"), resp.PunchNumber)
	fmt.Println("**************************************")
	c.reset()
}

func (c *clock) sendCardRead(tag string) {
	var resp apiResponse
	status, err := c.api.request("POST", "/device/card-reads/"+c.pendingRead, map[string]string{"cardTag": tag}, &resp)
	c.pendingRead = ""

	switch {
	case err != nil:
		fmt.Println("x Servidor indisponivel:", err)
	case status != http.StatusOK:
		fmt.Println("x Nao foi possivel enviar a tag:", errorText(status, resp))
	default:
		bell()
		fmt.Printf("Tag %s enviada ao painel.\n", resp.CardTag)
		if resp.AssignedTo != nil {
			fmt.Printf("Atencao: essa tag ja pertence a %s.\n", resp.AssignedTo.FullName)
		}
	}
}

func (c *clock) listEmployees() {
	var employees []employee
	// /employees e rota do painel; aqui so serve de cola para os testes.
	status, err := c.api.request("GET", "/employees", nil, &employees)
	if err != nil || status != http.StatusOK {
		fmt.Println("x Nao foi possivel listar os funcionarios.")
		return
	}

	fmt.Println()
	if len(employees) == 0 {
		fmt.Println("Nenhum funcionario cadastrado. Cadastre pelo painel.")
	}
	for _, e := range employees {
		situation := ""
		if !e.IsActive {
			situation = " (desativado)"
		}
		fmt.Printf("  %-14s %s%s\n", e.CardTag, e.FullName, situation)
	}
}

// pollCardReads faz o papel do ESP32 consultando o botao "Ler no relogio" do painel.
func (c *clock) pollCardReads() {
	for range time.Tick(pollInterval) {
		var resp apiResponse
		status, err := c.api.request("GET", "/device/card-reads/pending", nil, &resp)
		if err != nil {
			continue
		}

		c.mu.Lock()
		id, _ := resp.ID.(string)

		if status == http.StatusOK && id != c.pendingRead && c.tag == "" {
			c.pendingRead = id
			bell()
			fmt.Println()
			fmt.Println()
			fmt.Println("[PAINEL] Cadastro de cartao solicitado.")
			c.promptLocked()
		} else if status == http.StatusNoContent && c.pendingRead != "" {
			c.pendingRead = ""
			fmt.Println()
			fmt.Println()
			fmt.Println("[PAINEL] Leitura cancelada ou expirada.")
			c.promptLocked()
		}
		c.mu.Unlock()
	}
}

func errorText(status int, resp apiResponse) string {
	if resp.Error != "" {
		return resp.Error
	}
	return fmt.Sprintf("erro HTTP %d", status)
}

// bell faz o terminal apitar, no lugar do buzzer.
func bell() {
	fmt.Print("\a")
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
