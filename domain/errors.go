package domain

import (
	"errors"
)

var (
	ErrIsEmpty     = errors.New("is empty")
	ErrIsDuplicate = errors.New("is duplicate")
)
