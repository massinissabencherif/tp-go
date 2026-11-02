package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/massinissabencherif/tp-go/internal/apperr"
	"github.com/massinissabencherif/tp-go/internal/model"
)

type Repository interface {
	ListVehicules(ctx context.Context) ([]model.Vehicule, error)
	GetVehicule(ctx context.Context, plaque string) (model.Vehicule, error)
	CreateVehicule(ctx context.Context, v model.Vehicule) error
	DeleteVehicule(ctx context.Context, plaque string) error
	CountInterventions(ctx context.Context, plaque string) (int, error)
	CreateIntervention(ctx context.Context, i *model.Intervention) error
	ListInterventions(ctx context.Context, plaque string) ([]model.Intervention, error)
	CountVehicules(ctx context.Context) (int, error)
	CountInterventionsByStatut(ctx context.Context, statut model.Statut) (int, error)
}

type Service struct {
	repo Repository
	now  func() time.Time
}

func New(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

func (s *Service) ListVehicules(ctx context.Context) ([]model.Vehicule, error) {
	return s.repo.ListVehicules(ctx)
}

func (s *Service) CreateVehicule(ctx context.Context, v model.Vehicule) (model.Vehicule, error) {
	v.Normalize()
	if err := validateVehicule(v, s.now()); err != nil {
		return model.Vehicule{}, err
	}

	_, err := s.repo.GetVehicule(ctx, v.Plaque)
	switch {
	case err == nil:
		return model.Vehicule{}, fmt.Errorf("véhicule %s : %w", v.Plaque, apperr.ErrAlreadyExists)
	case !errors.Is(err, apperr.ErrNotFound):
		return model.Vehicule{}, err
	}

	if err := s.repo.CreateVehicule(ctx, v); err != nil {
		return model.Vehicule{}, err
	}
	return v, nil
}

func (s *Service) GetVehicule(ctx context.Context, plaque string) (model.Vehicule, error) {
	plaque = model.NormalizePlaque(plaque)
	if err := validatePlaque(plaque); err != nil {
		return model.Vehicule{}, err
	}
	return s.repo.GetVehicule(ctx, plaque)
}

func (s *Service) DeleteVehicule(ctx context.Context, plaque string) error {
	if _, err := s.GetVehicule(ctx, plaque); err != nil {
		return err
	}
	plaque = model.NormalizePlaque(plaque)

	n, err := s.repo.CountInterventions(ctx, plaque)
	if err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("véhicule %s (%d interventions) : %w", plaque, n, apperr.ErrHasInterventions)
	}
	return s.repo.DeleteVehicule(ctx, plaque)
}

func (s *Service) AddIntervention(ctx context.Context, plaque string, i model.Intervention) (model.Intervention, error) {
	v, err := s.GetVehicule(ctx, plaque)
	if err != nil {
		return model.Intervention{}, err
	}

	i.ID = 0
	i.Plaque = v.Plaque
	i.Description = strings.TrimSpace(i.Description)
	if i.Statut == "" {
		i.Statut = model.StatutEnCours
	}
	if err := validateIntervention(i); err != nil {
		return model.Intervention{}, err
	}

	if err := s.repo.CreateIntervention(ctx, &i); err != nil {
		return model.Intervention{}, err
	}
	return i, nil
}

func (s *Service) ListInterventions(ctx context.Context, plaque string) ([]model.Intervention, error) {
	v, err := s.GetVehicule(ctx, plaque)
	if err != nil {
		return nil, err
	}
	return s.repo.ListInterventions(ctx, v.Plaque)
}

func (s *Service) Stats(ctx context.Context) (model.Stats, error) {
	stats := model.Stats{InterventionsParStatut: make(map[model.Statut]int, len(model.Statuts))}

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)

	done := func(err error, store func()) {
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return
		}
		store()
	}

	wg.Add(1 + len(model.Statuts))
	go func() {
		defer wg.Done()
		n, err := s.repo.CountVehicules(ctx)
		done(err, func() { stats.NombreVehicules = n })
	}()
	for _, statut := range model.Statuts {
		go func() {
			defer wg.Done()
			n, err := s.repo.CountInterventionsByStatut(ctx, statut)
			done(err, func() { stats.InterventionsParStatut[statut] = n })
		}()
	}
	wg.Wait()

	if firstErr != nil {
		return model.Stats{}, fmt.Errorf("calculer les statistiques : %w", firstErr)
	}
	return stats, nil
}
