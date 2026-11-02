package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/massinissabencherif/tp-go/internal/model"
	"github.com/massinissabencherif/tp-go/internal/service"
	"github.com/massinissabencherif/tp-go/internal/store"
)

func newAPI(t *testing.T) http.Handler {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Error(err)
		}
	})
	return New(service.New(st))
}

func do(t *testing.T, api http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	return rec
}

const clio = `{"plaque":"AB-123-CD","marque":"Renault","modele":"Clio","annee":2020,"kilometrage":42000,
	"proprietaire":{"nom":"Alice","telephone":"0612345678"}}`

func TestVehicules(t *testing.T) {
	api := newAPI(t)

	tests := []struct {
		name       string
		method     string
		target     string
		body       string
		wantStatus int
		wantBody   string
	}{
		{"liste vide", "GET", "/vehicules", "", 200, "[]"},
		{"création", "POST", "/vehicules", clio, 201, `"plaque":"AB-123-CD"`},
		{"doublon", "POST", "/vehicules", clio, 409, "existe déjà"},
		{"détails", "GET", "/vehicules/AB-123-CD", "", 200, `"modele":"Clio"`},
		{"détails en minuscules", "GET", "/vehicules/ab-123-cd", "", 200, `"plaque":"AB-123-CD"`},
		{"liste", "GET", "/vehicules", "", 200, `"marque":"Renault"`},
		{"introuvable", "GET", "/vehicules/ZZ-999-ZZ", "", 404, "introuvable"},
		{"plaque mal formée", "GET", "/vehicules/abc", "", 422, "plaque"},
		{"JSON invalide", "POST", "/vehicules", `{`, 400, "JSON invalide"},
		{"champ inconnu", "POST", "/vehicules", `{"couleur":"rouge"}`, 400, "JSON invalide"},
		{"validation", "POST", "/vehicules", `{"plaque":"AA-111-AA"}`, 422, "marque"},
		{"suppression", "DELETE", "/vehicules/AB-123-CD", "", 204, ""},
		{"supprimé", "GET", "/vehicules/AB-123-CD", "", 404, "introuvable"},
		{"méthode non prévue", "PUT", "/vehicules/AB-123-CD", "", 405, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, api, tt.method, tt.target, tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, attendu %d (corps : %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("corps = %s, attendu : %s", rec.Body, tt.wantBody)
			}
			if tt.wantStatus < 400 && tt.wantStatus != 204 {
				if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
					t.Errorf("Content-Type = %q", ct)
				}
			}
		})
	}
}

func TestInterventionsEtStats(t *testing.T) {
	api := newAPI(t)
	if rec := do(t, api, "POST", "/vehicules", clio); rec.Code != 201 {
		t.Fatalf("création : %d %s", rec.Code, rec.Body)
	}

	t.Run("création", func(t *testing.T) {
		rec := do(t, api, "POST", "/vehicules/AB-123-CD/interventions",
			`{"date":"2025-03-01","description":"Vidange"}`)
		if rec.Code != 201 {
			t.Fatalf("status = %d, corps : %s", rec.Code, rec.Body)
		}
		var i model.Intervention
		if err := json.Unmarshal(rec.Body.Bytes(), &i); err != nil {
			t.Fatal(err)
		}
		if i.ID == 0 || i.Plaque != "AB-123-CD" || i.Statut != model.StatutEnCours {
			t.Errorf("intervention = %+v", i)
		}
	})
	t.Run("statut terminée", func(t *testing.T) {
		rec := do(t, api, "POST", "/vehicules/AB-123-CD/interventions",
			`{"date":"2025-03-02","description":"Freins","statut":"terminee"}`)
		if rec.Code != 201 {
			t.Fatalf("status = %d", rec.Code)
		}
	})
	t.Run("liste", func(t *testing.T) {
		rec := do(t, api, "GET", "/vehicules/AB-123-CD/interventions", "")
		var list []model.Intervention
		if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list) != 2 {
			t.Errorf("liste = %v, %v", list, err)
		}
	})
	t.Run("statut invalide", func(t *testing.T) {
		rec := do(t, api, "POST", "/vehicules/AB-123-CD/interventions",
			`{"date":"2025-03-01","description":"x","statut":"fini"}`)
		if rec.Code != 422 {
			t.Errorf("status = %d", rec.Code)
		}
	})
	t.Run("véhicule inconnu", func(t *testing.T) {
		rec := do(t, api, "POST", "/vehicules/ZZ-999-ZZ/interventions",
			`{"date":"2025-03-01","description":"x"}`)
		if rec.Code != 404 {
			t.Errorf("status = %d", rec.Code)
		}
	})
	t.Run("suppression refusée", func(t *testing.T) {
		rec := do(t, api, "DELETE", "/vehicules/AB-123-CD", "")
		if rec.Code != 409 || !strings.Contains(rec.Body.String(), "interventions") {
			t.Errorf("status = %d, corps : %s", rec.Code, rec.Body)
		}
	})
	t.Run("stats", func(t *testing.T) {
		rec := do(t, api, "GET", "/stats", "")
		var stats model.Stats
		if err := json.Unmarshal(rec.Body.Bytes(), &stats); err != nil {
			t.Fatal(err)
		}
		if stats.NombreVehicules != 1 ||
			stats.InterventionsParStatut[model.StatutEnCours] != 1 ||
			stats.InterventionsParStatut[model.StatutTerminee] != 1 ||
			stats.InterventionsParStatut[model.StatutAnnulee] != 0 {
			t.Errorf("stats = %+v", stats)
		}
	})
}

func TestCorpsTropGros(t *testing.T) {
	api := newAPI(t)
	body := `{"marque":"` + strings.Repeat("a", maxBodyBytes) + `"}`
	if rec := do(t, api, "POST", "/vehicules", body); rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, attendu 413", rec.Code)
	}
}

type panicService struct{ Service }

func (panicService) ListVehicules(context.Context) ([]model.Vehicule, error) {
	panic("boum")
}

func TestPanicRecuperee(t *testing.T) {
	rec := do(t, New(panicService{}), "GET", "/vehicules", "")
	if rec.Code != 500 || !strings.Contains(rec.Body.String(), "erreur interne") {
		t.Errorf("status = %d, corps : %s", rec.Code, rec.Body)
	}
}

type failService struct{ Service }

func (failService) ListVehicules(context.Context) ([]model.Vehicule, error) {
	return nil, errors.New("disque plein : /var/db/secret")
}

func TestErreurInterneMasquee(t *testing.T) {
	rec := do(t, New(failService{}), "GET", "/vehicules", "")
	if rec.Code != 500 || strings.Contains(rec.Body.String(), "secret") {
		t.Errorf("status = %d, corps : %s", rec.Code, rec.Body)
	}
}
