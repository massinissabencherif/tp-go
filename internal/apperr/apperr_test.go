package apperr

import (
	"errors"
	"fmt"
	"testing"
)

func TestWrapping(t *testing.T) {
	t.Run("sentinelle enveloppée", func(t *testing.T) {
		err := fmt.Errorf("véhicule AA-123-AA : %w", ErrNotFound)
		if !errors.Is(err, ErrNotFound) {
			t.Error("errors.Is devrait retrouver ErrNotFound")
		}
		if errors.Is(err, ErrAlreadyExists) {
			t.Error("ErrAlreadyExists ne devrait pas correspondre")
		}
	})
	t.Run("erreur de validation enveloppée", func(t *testing.T) {
		err := fmt.Errorf("créer : %w", Invalid("marque", "obligatoire"))
		var ve ValidationError
		if !errors.As(err, &ve) || ve.Field != "marque" {
			t.Errorf("errors.As a échoué : %v", err)
		}
	})
}
