package repositories

import (
	"errors"
)

// InvalidDataError represents an error indicating that the provided data is invalid.
var InvalidDataError = errors.New("Invalid data given")

// NotFoundError represents an error indicating that the requested record was not found.
var NotFoundError = errors.New("Record not found")

// DatabaseError represents a general error related to database operations.
var DatabaseError = errors.New("Database error")

// ErrParsingModel represents an error indicating that there was an issue parsing the model.
var ErrParsingModel = errors.New("Error parsing model")
