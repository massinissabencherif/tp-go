package apperr

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound         = errors.New("introuvable")
	ErrAlreadyExists    = errors.New("existe déjà")
	ErrHasInterventions = errors.New("suppression impossible : le véhicule a des interventions")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s : %s", e.Field, e.Message)
}

func Invalid(field, message string) error {
	return ValidationError{Field: field, Message: message}
}
