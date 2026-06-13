package domain

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrLocked        = errors.New("locked")
	ErrNoSession     = errors.New("no active session")
)
