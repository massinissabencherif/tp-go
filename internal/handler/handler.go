package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/massinissabencherif/tp-go/internal/apperr"
	"github.com/massinissabencherif/tp-go/internal/model"
)

const maxBodyBytes = 1 << 20

type Service interface {
	ListVehicules(ctx context.Context) ([]model.Vehicule, error)
	CreateVehicule(ctx context.Context, v model.Vehicule) (model.Vehicule, error)
	GetVehicule(ctx context.Context, plaque string) (model.Vehicule, error)
	DeleteVehicule(ctx context.Context, plaque string) error
	AddIntervention(ctx context.Context, plaque string, i model.Intervention) (model.Intervention, error)
	ListInterventions(ctx context.Context, plaque string) ([]model.Intervention, error)
	Stats(ctx context.Context) (model.Stats, error)
}

type errorResponse struct {
	Error string `json:"error"`
}

type handler struct {
	svc Service
}

func New(svc Service) http.Handler {
	h := &handler{svc: svc}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /vehicules", h.listVehicules)
	mux.HandleFunc("POST /vehicules", h.createVehicule)
	mux.HandleFunc("GET /vehicules/{plaque}", h.getVehicule)
	mux.HandleFunc("DELETE /vehicules/{plaque}", h.deleteVehicule)
	mux.HandleFunc("POST /vehicules/{plaque}/interventions", h.createIntervention)
	mux.HandleFunc("GET /vehicules/{plaque}/interventions", h.listInterventions)
	mux.HandleFunc("GET /stats", h.stats)

	return recoverPanic(logging(mux))
}

func (h *handler) listVehicules(w http.ResponseWriter, r *http.Request) {
	vehicules, err := h.svc.ListVehicules(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, vehicules)
}

func (h *handler) createVehicule(w http.ResponseWriter, r *http.Request) {
	var in model.Vehicule
	if !decodeBody(w, r, &in) {
		return
	}
	v, err := h.svc.CreateVehicule(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Location", "/vehicules/"+v.Plaque)
	writeJSON(w, http.StatusCreated, v)
}

func (h *handler) getVehicule(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.GetVehicule(r.Context(), r.PathValue("plaque"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (h *handler) deleteVehicule(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteVehicule(r.Context(), r.PathValue("plaque")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) createIntervention(w http.ResponseWriter, r *http.Request) {
	var in model.Intervention
	if !decodeBody(w, r, &in) {
		return
	}
	i, err := h.svc.AddIntervention(r.Context(), r.PathValue("plaque"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, i)
}

func (h *handler) listInterventions(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListInterventions(r.Context(), r.PathValue("plaque"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *handler) stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.Stats(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "corps de requête trop volumineux"})
			return false
		}
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Sprintf("JSON invalide : %v", err)})
		return false
	}

	if dec.More() {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "JSON invalide : données en trop"})
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, err error) {
	var ve apperr.ValidationError
	switch {
	case errors.As(err, &ve):
		writeJSON(w, http.StatusUnprocessableEntity, errorResponse{Error: err.Error()})
	case errors.Is(err, apperr.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
	case errors.Is(err, apperr.ErrAlreadyExists), errors.Is(err, apperr.ErrHasInterventions):
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})
	default:
		log.Printf("erreur interne : %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "erreur interne"})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("écriture de la réponse : %v", err)
	}
}

func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic récupérée : %v", rec)
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "erreur interne"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.RequestURI(), time.Since(start))
	})
}
