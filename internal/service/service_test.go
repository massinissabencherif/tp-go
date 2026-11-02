package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/massinissabencherif/tp-go/internal/apperr"
	"github.com/massinissabencherif/tp-go/internal/model"
)

type fakeRepo struct {
	vehicules     map[string]model.Vehicule
	interventions []model.Intervention
	failCounts    bool
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{vehicules: map[string]model.Vehicule{}}
}

func (f *fakeRepo) ListVehicules(context.Context) ([]model.Vehicule, error) {
	list := []model.Vehicule{}
	for _, v := range f.vehicules {
		list = append(list, v)
	}
	return list, nil
}

func (f *fakeRepo) GetVehicule(_ context.Context, plaque string) (model.Vehicule, error) {
	v, ok := f.vehicules[plaque]
	if !ok {
		return model.Vehicule{}, apperr.ErrNotFound
	}
	return v, nil
}

func (f *fakeRepo) CreateVehicule(_ context.Context, v model.Vehicule) error {
	f.vehicules[v.Plaque] = v
	return nil
}

func (f *fakeRepo) DeleteVehicule(_ context.Context, plaque string) error {
	delete(f.vehicules, plaque)
	return nil
}

func (f *fakeRepo) CountInterventions(_ context.Context, plaque string) (int, error) {
	n := 0
	for _, i := range f.interventions {
		if i.Plaque == plaque {
			n++
		}
	}
	return n, nil
}

func (f *fakeRepo) CreateIntervention(_ context.Context, i *model.Intervention) error {
	i.ID = int64(len(f.interventions) + 1)
	f.interventions = append(f.interventions, *i)
	return nil
}

func (f *fakeRepo) ListInterventions(ctx context.Context, plaque string) ([]model.Intervention, error) {
	list := []model.Intervention{}
	for _, i := range f.interventions {
		if i.Plaque == plaque {
			list = append(list, i)
		}
	}
	return list, nil
}

func (f *fakeRepo) CountVehicules(context.Context) (int, error) {
	if f.failCounts {
		return 0, errors.New("base indisponible")
	}
	return len(f.vehicules), nil
}

func (f *fakeRepo) CountInterventionsByStatut(_ context.Context, statut model.Statut) (int, error) {
	n := 0
	for _, i := range f.interventions {
		if i.Statut == statut {
			n++
		}
	}
	return n, nil
}

func validVehicule() model.Vehicule {
	return model.Vehicule{
		Plaque: "AB-123-CD", Marque: "Renault", Modele: "Clio", Annee: 2020, Kilometrage: 5000,
		Proprietaire: model.Proprietaire{Nom: "Alice", Telephone: "06 12 34 56 78"},
	}
}

func newService(repo *fakeRepo) *Service {
	s := New(repo)
	s.now = func() time.Time { return time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC) }
	return s
}

func TestCreateVehicule(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*model.Vehicule)
		want   string
	}{
		{"valide", func(*model.Vehicule) {}, ""},
		{"plaque en minuscules normalisée", func(v *model.Vehicule) { v.Plaque = "ab-123-cd" }, ""},
		{"plaque invalide", func(v *model.Vehicule) { v.Plaque = "ABC-12-DE" }, "plaque"},
		{"marque vide", func(v *model.Vehicule) { v.Marque = "  " }, "marque"},
		{"modèle vide", func(v *model.Vehicule) { v.Modele = "" }, "modele"},
		{"année trop ancienne", func(v *model.Vehicule) { v.Annee = 1800 }, "annee"},
		{"année dans le futur", func(v *model.Vehicule) { v.Annee = 2030 }, "annee"},
		{"kilométrage négatif", func(v *model.Vehicule) { v.Kilometrage = -1 }, "kilometrage"},
		{"propriétaire sans nom", func(v *model.Vehicule) { v.Nom = "" }, "proprietaire.nom"},
		{"téléphone invalide", func(v *model.Vehicule) { v.Telephone = "abc" }, "proprietaire.telephone"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newService(newFakeRepo())
			v := validVehicule()
			tt.modify(&v)

			got, err := svc.CreateVehicule(context.Background(), v)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("erreur inattendue : %v", err)
				}
				if got.Plaque != "AB-123-CD" {
					t.Errorf("plaque = %q", got.Plaque)
				}
				return
			}
			var ve apperr.ValidationError
			if !errors.As(err, &ve) || ve.Field != tt.want {
				t.Errorf("erreur = %v, champ attendu %q", err, tt.want)
			}
		})
	}
}

func TestCreateVehiculeDoublon(t *testing.T) {
	svc := newService(newFakeRepo())
	ctx := context.Background()
	if _, err := svc.CreateVehicule(ctx, validVehicule()); err != nil {
		t.Fatal(err)
	}
	_, err := svc.CreateVehicule(ctx, validVehicule())
	if !errors.Is(err, apperr.ErrAlreadyExists) {
		t.Errorf("erreur = %v", err)
	}
}

func TestDeleteVehicule(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	svc := newService(repo)
	for _, p := range []string{"AA-111-AA", "BB-222-BB"} {
		v := validVehicule()
		v.Plaque = p
		if _, err := svc.CreateVehicule(ctx, v); err != nil {
			t.Fatal(err)
		}
	}
	_, err := svc.AddIntervention(ctx, "BB-222-BB", model.Intervention{Date: "2025-01-10", Description: "Freins"})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		plaque  string
		wantErr error
	}{
		{"sans intervention", "AA-111-AA", nil},
		{"avec intervention", "BB-222-BB", apperr.ErrHasInterventions},
		{"inexistant", "CC-333-CC", apperr.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := svc.DeleteVehicule(ctx, tt.plaque); !errors.Is(err, tt.wantErr) {
				t.Errorf("erreur = %v, attendu %v", err, tt.wantErr)
			}
		})
	}
}

func TestAddIntervention(t *testing.T) {
	ctx := context.Background()
	svc := newService(newFakeRepo())
	if _, err := svc.CreateVehicule(ctx, validVehicule()); err != nil {
		t.Fatal(err)
	}

	t.Run("statut par défaut", func(t *testing.T) {
		got, err := svc.AddIntervention(ctx, "AB-123-CD", model.Intervention{Date: "2025-03-01", Description: "Vidange"})
		if err != nil || got.Statut != model.StatutEnCours || got.ID == 0 || got.Plaque != "AB-123-CD" {
			t.Errorf("got %+v, %v", got, err)
		}
	})

	invalid := []struct {
		name string
		in   model.Intervention
		want string
	}{
		{"date invalide", model.Intervention{Date: "01/03/2025", Description: "x"}, "date"},
		{"description vide", model.Intervention{Date: "2025-03-01", Description: " "}, "description"},
		{"statut inconnu", model.Intervention{Date: "2025-03-01", Description: "x", Statut: "fini"}, "statut"},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.AddIntervention(ctx, "AB-123-CD", tt.in)
			var ve apperr.ValidationError
			if !errors.As(err, &ve) || ve.Field != tt.want {
				t.Errorf("erreur = %v, champ attendu %q", err, tt.want)
			}
		})
	}

	t.Run("véhicule inconnu", func(t *testing.T) {
		_, err := svc.AddIntervention(ctx, "ZZ-999-ZZ", model.Intervention{Date: "2025-03-01", Description: "x"})
		if !errors.Is(err, apperr.ErrNotFound) {
			t.Errorf("erreur = %v", err)
		}
	})
}

func TestStats(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	svc := newService(repo)
	if _, err := svc.CreateVehicule(ctx, validVehicule()); err != nil {
		t.Fatal(err)
	}
	for _, st := range []model.Statut{model.StatutEnCours, model.StatutTerminee, model.StatutTerminee} {
		_, err := svc.AddIntervention(ctx, "AB-123-CD", model.Intervention{Date: "2025-03-01", Description: "x", Statut: st})
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Run("comptages", func(t *testing.T) {
		stats, err := svc.Stats(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if stats.NombreVehicules != 1 ||
			stats.InterventionsParStatut[model.StatutEnCours] != 1 ||
			stats.InterventionsParStatut[model.StatutTerminee] != 2 ||
			stats.InterventionsParStatut[model.StatutAnnulee] != 0 {
			t.Errorf("stats = %+v", stats)
		}
	})
	t.Run("erreur propagée", func(t *testing.T) {
		repo.failCounts = true
		if _, err := svc.Stats(ctx); err == nil {
			t.Error("une erreur était attendue")
		}
	})
}
