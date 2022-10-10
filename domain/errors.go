package domain

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrIsEmpty     = errors.New("is empty")
	ErrIsDuplicate = errors.New("is duplicate")
)

type MultiError interface {
	error
	Add(err error)
	IsEmpty() bool
}

func NewMultiError() *multiErr {
	return &multiErr{}
}

type multiErr struct {
	Errs []error
}

func (m *multiErr) Error() string {
	var msg string
	switch len(m.Errs) {
	case 0:
		return "0 errors"
	case 1:
		msg = "1 error:\n"
	default:
		msg = fmt.Sprintf("%d errors:\n", len(m.Errs))
	}

	for _, err := range m.Errs {
		if err != nil && err.Error() != "" {
			msg += fmt.Sprintf("- %s\n", err.Error())
		}
	}

	return strings.Trim(msg, "\n")
}

func (m *multiErr) Add(err error) {
	if err != nil {
		m.Errs = append(m.Errs, err)
	}
}

func (m *multiErr) IsEmpty() bool {
	return len(m.Errs) == 0
}

func (m *multiErr) WithPrefix(prefix string) MultiError {
	return &multiErrWithPrefix{
		MultiError: m,
		prefix:     prefix,
	}
}

type multiErrWithPrefix struct {
	MultiError
	prefix string
}

func (m *multiErrWithPrefix) Add(err error) {
	if err != nil && m.prefix != "" {
		err = fmt.Errorf("%s: %w", m.prefix, err)
	}

	m.MultiError.Add(err)
}
