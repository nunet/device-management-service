package repositories

import (
	"errors"
)

// ErrInvalidData represents an error indicating that the provided data is invalid.
var ErrInvalidData = errors.New("invalid data given")

// ErrNotFound represents an error indicating that the requested record was not found.
var ErrNotFound = errors.New("record not found")

// DatabaseError represents a general error related to database operations.
var ErrDatabase = errors.New("database error")

// ErrParsingModel represents an error indicating that there was an issue parsing the model.
var ErrParsingModel = errors.New("error parsing model")
