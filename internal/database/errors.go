package database

import "errors"

var (
	ErrNoConnection    = errors.New("no active connection")
	ErrInvalidConfig   = errors.New("invalid connection config")
	ErrNotSupported    = errors.New("operation not supported")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrTimeout         = errors.New("database operation timeout")
	ErrInvalidQuery    = errors.New("invalid query")
	ErrInvalidDatabase = errors.New("invalid database")
)
