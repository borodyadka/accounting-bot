package accounting_bot

import (
	"fmt"
)

type UnknownCommandError struct {
	Command string
}

func (e *UnknownCommandError) Error() string {
	return e.String()
}

func (e *UnknownCommandError) String() string {
	if e.Command != "" {
		return fmt.Sprintf(`unknown command "%s"`, e.Command)
	}
	return "unknown command"
}

type InternalError struct {
	Err error
}

func (e *InternalError) Error() string {
	if e.Err != nil {
		return e.String() + ": " + e.Err.Error()
	}
	return e.String()
}

func (*InternalError) String() string {
	return "internal error"
}

func NewInternalError(e error) *InternalError {
	return &InternalError{Err: e}
}

type UserNotFoundError struct{}

func (e *UserNotFoundError) Error() string {
	return e.String()
}

func (*UserNotFoundError) String() string {
	return "user not found or not enabled"
}

type InvalidCurrencyError struct {
	Currency string
}

func (e *InvalidCurrencyError) Error() string {
	return e.String()
}

func (e *InvalidCurrencyError) String() string {
	if e.Currency != "" {
		return fmt.Sprintf(`invalid currency "%s"`, e.Currency)
	}
	return "invalid currency"
}