package service

import "fmt"

type ErrorKind int

const (
	KindValidation ErrorKind = iota
	KindNotFound
	KindConflict
	KindUnauthorized
	KindForbidden
)

// Error e o erro de negocio devolvido pelos services. O handler converte
// Kind em status HTTP e Message vai direto para o cliente (frontend/ESP32).
type Error struct {
	Kind    ErrorKind
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func newError(kind ErrorKind, format string, args ...any) *Error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...)}
}
