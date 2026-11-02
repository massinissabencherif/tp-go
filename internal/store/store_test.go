package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/massinissabencherif/tp-go/internal/apperr"
	"github.com/massinissabencherif/tp-go/internal/model"
)

func newStore(t *testing.T) *SQLite {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s
}

func vehicule(plaque string) model.Vehicule {
	return model.Vehicule{
		Plaque: plaque, Marque: "Renault", Modele: "Clio", Annee: 2020, Kilometrage: 1000,
		Proprietaire: model.Proprietaire{Nom: "Alice", Telephone: "0612345678"},
	}
}

func TestVehicules(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	t.Run("création puis lecture", func(t *testing.T) {
		want := vehicule("AA-123-AA")
		if err := s.CreateVehicule(ctx, want); err != nil {
			t.Fatal(err)
		}
		got, err := s.GetVehicule(ctx, "AA-123-AA")
		if err != nil || got != want {
			t.Errorf("got %+v, %v", got, err)
		}
	})
	t.Run("doublon", func(t *testing.T) {
		err := s.CreateVehicule(ctx, vehicule("AA-123-AA"))
		if !errors.Is(err, apperr.ErrAlreadyExists) {
			t.Errorf("erreur = %v", err)
		}
	})
	t.Run("introuvable", func(t *testing.T) {
		_, err := s.GetVehicule(ctx, "ZZ-999-ZZ")
		if !errors.Is(err, apperr.ErrNotFound) {
			t.Errorf("erreur = %v", err)
		}
	})
	t.Run("liste", func(t *testing.T) {
		list, err := s.ListVehicules(ctx)
		if err != nil || len(list) != 1 {
			t.Errorf("got %v, %v", list, err)
		}
	})
}

func TestDeleteVehicule(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	for _, p := range []string{"AA-123-AA", "BB-456-BB"} {
		if err := s.CreateVehicule(ctx, vehicule(p)); err != nil {
			t.Fatal(err)
		}
	}
	i := &model.Intervention{Plaque: "BB-456-BB", Date: "2025-01-15", Description: "Vidange", Statut: model.StatutEnCours}
	if err := s.CreateIntervention(ctx, i); err != nil || i.ID == 0 {
		t.Fatalf("intervention : %+v, %v", i, err)
	}

	tests := []struct {
		name    string
		plaque  string
		wantErr error
	}{
		{"sans intervention", "AA-123-AA", nil},
		{"avec intervention", "BB-456-BB", apperr.ErrHasInterventions},
		{"inexistant", "ZZ-999-ZZ", apperr.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.DeleteVehicule(ctx, tt.plaque)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("erreur = %v, attendu %v", err, tt.wantErr)
			}
		})
	}
}

func TestInterventionsAndCounts(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	if err := s.CreateVehicule(ctx, vehicule("AA-123-AA")); err != nil {
		t.Fatal(err)
	}
	for _, st := range []model.Statut{model.StatutEnCours, model.StatutTerminee, model.StatutTerminee} {
		i := &model.Intervention{Plaque: "AA-123-AA", Date: "2025-01-15", Description: "Travaux", Statut: st}
		if err := s.CreateIntervention(ctx, i); err != nil {
			t.Fatal(err)
		}
	}

	list, err := s.ListInterventions(ctx, "AA-123-AA")
	if err != nil || len(list) != 3 {
		t.Fatalf("got %v, %v", list, err)
	}
	n, err := s.CountInterventions(ctx, "AA-123-AA")
	if err != nil || n != 3 {
		t.Errorf("CountInterventions = %d, %v", n, err)
	}
	n, err = s.CountInterventionsByStatut(ctx, model.StatutTerminee)
	if err != nil || n != 2 {
		t.Errorf("CountInterventionsByStatut = %d, %v", n, err)
	}
	n, err = s.CountVehicules(ctx)
	if err != nil || n != 1 {
		t.Errorf("CountVehicules = %d, %v", n, err)
	}

	orphan := &model.Intervention{Plaque: "ZZ-999-ZZ", Date: "2025-01-15", Description: "x", Statut: model.StatutEnCours}
	if err := s.CreateIntervention(ctx, orphan); err == nil {
		t.Error("une intervention orpheline devrait être refusée")
	}
}
