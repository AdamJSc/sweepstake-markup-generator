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

type MultiError struct {
	Errs   []error
	prefix string
}

func (m *MultiError) Error() string {
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

func (m *MultiError) withPrefix(prefix string) *MultiError {
	return &MultiError{
		Errs:   m.Errs,
		prefix: prefix,
	}
}

func (m *MultiError) add(err error) {
	if err != nil {
		if m.prefix != "" {
			err = fmt.Errorf("%s: %w", m.prefix, err)
		}
		m.Errs = append(m.Errs, err)
	}
}
