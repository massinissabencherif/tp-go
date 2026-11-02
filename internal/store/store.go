package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"modernc.org/sqlite"

	"github.com/massinissabencherif/tp-go/internal/apperr"
	"github.com/massinissabencherif/tp-go/internal/model"
)

const (
	sqlitePrimaryKey = 1555
	sqliteUnique     = 2067
)

const schema = `
CREATE TABLE IF NOT EXISTS vehicules (
	plaque                 TEXT PRIMARY KEY,
	marque                 TEXT NOT NULL,
	modele                 TEXT NOT NULL,
	annee                  INTEGER NOT NULL,
	kilometrage            INTEGER NOT NULL,
	proprietaire_nom       TEXT NOT NULL,
	proprietaire_telephone TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS interventions (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	plaque      TEXT NOT NULL REFERENCES vehicules(plaque),
	date        TEXT NOT NULL,
	description TEXT NOT NULL,
	statut      TEXT NOT NULL CHECK (statut IN ('en_cours', 'terminee', 'annulee'))
);
CREATE INDEX IF NOT EXISTS idx_interventions_plaque ON interventions(plaque);
`

type SQLite struct {
	db *sql.DB
}

func Open(path string) (*SQLite, error) {
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("ouvrir la base %s : %w", path, err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, errors.Join(fmt.Errorf("créer le schéma : %w", err), db.Close())
	}
	return &SQLite{db: db}, nil
}

func (s *SQLite) Close() error {
	return s.db.Close()
}

func (s *SQLite) ListVehicules(ctx context.Context) ([]model.Vehicule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT plaque, marque, modele, annee, kilometrage, proprietaire_nom, proprietaire_telephone
		FROM vehicules ORDER BY plaque`)
	if err != nil {
		return nil, fmt.Errorf("lister les véhicules : %w", err)
	}
	defer rows.Close()

	vehicules := []model.Vehicule{}
	for rows.Next() {
		var v model.Vehicule
		if err := rows.Scan(&v.Plaque, &v.Marque, &v.Modele, &v.Annee, &v.Kilometrage, &v.Nom, &v.Telephone); err != nil {
			return nil, fmt.Errorf("lire un véhicule : %w", err)
		}
		vehicules = append(vehicules, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("parcourir les véhicules : %w", err)
	}
	return vehicules, nil
}

func (s *SQLite) GetVehicule(ctx context.Context, plaque string) (model.Vehicule, error) {
	var v model.Vehicule
	err := s.db.QueryRowContext(ctx, `
		SELECT plaque, marque, modele, annee, kilometrage, proprietaire_nom, proprietaire_telephone
		FROM vehicules WHERE plaque = ?`, plaque).
		Scan(&v.Plaque, &v.Marque, &v.Modele, &v.Annee, &v.Kilometrage, &v.Nom, &v.Telephone)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Vehicule{}, fmt.Errorf("véhicule %s : %w", plaque, apperr.ErrNotFound)
	}
	if err != nil {
		return model.Vehicule{}, fmt.Errorf("lire le véhicule %s : %w", plaque, err)
	}
	return v, nil
}

func (s *SQLite) CreateVehicule(ctx context.Context, v model.Vehicule) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO vehicules (plaque, marque, modele, annee, kilometrage, proprietaire_nom, proprietaire_telephone)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		v.Plaque, v.Marque, v.Modele, v.Annee, v.Kilometrage, v.Nom, v.Telephone)
	if isUniqueViolation(err) {
		return fmt.Errorf("véhicule %s : %w", v.Plaque, apperr.ErrAlreadyExists)
	}
	if err != nil {
		return fmt.Errorf("insérer le véhicule %s : %w", v.Plaque, err)
	}
	return nil
}

func (s *SQLite) DeleteVehicule(ctx context.Context, plaque string) error {
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM vehicules
		WHERE plaque = ? AND NOT EXISTS (SELECT 1 FROM interventions WHERE plaque = ?)`,
		plaque, plaque)
	if err != nil {
		return fmt.Errorf("supprimer le véhicule %s : %w", plaque, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("supprimer le véhicule %s : %w", plaque, err)
	}
	if n > 0 {
		return nil
	}

	if _, err := s.GetVehicule(ctx, plaque); err != nil {
		return err
	}
	return fmt.Errorf("véhicule %s : %w", plaque, apperr.ErrHasInterventions)
}

func (s *SQLite) CountInterventions(ctx context.Context, plaque string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM interventions WHERE plaque = ?`, plaque).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("compter les interventions de %s : %w", plaque, err)
	}
	return n, nil
}

func (s *SQLite) CreateIntervention(ctx context.Context, i *model.Intervention) error {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO interventions (plaque, date, description, statut) VALUES (?, ?, ?, ?)`,
		i.Plaque, i.Date, i.Description, string(i.Statut))
	if err != nil {
		return fmt.Errorf("insérer l'intervention de %s : %w", i.Plaque, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("lire l'identifiant de l'intervention : %w", err)
	}
	i.ID = id
	return nil
}

func (s *SQLite) ListInterventions(ctx context.Context, plaque string) ([]model.Intervention, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, plaque, date, description, statut
		FROM interventions WHERE plaque = ? ORDER BY date, id`, plaque)
	if err != nil {
		return nil, fmt.Errorf("lister les interventions de %s : %w", plaque, err)
	}
	defer rows.Close()

	interventions := []model.Intervention{}
	for rows.Next() {
		var i model.Intervention
		if err := rows.Scan(&i.ID, &i.Plaque, &i.Date, &i.Description, &i.Statut); err != nil {
			return nil, fmt.Errorf("lire une intervention : %w", err)
		}
		interventions = append(interventions, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("parcourir les interventions : %w", err)
	}
	return interventions, nil
}

func (s *SQLite) CountVehicules(ctx context.Context) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vehicules`).Scan(&n); err != nil {
		return 0, fmt.Errorf("compter les véhicules : %w", err)
	}
	return n, nil
}

func (s *SQLite) CountInterventionsByStatut(ctx context.Context, statut model.Statut) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM interventions WHERE statut = ?`, string(statut)).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("compter les interventions %s : %w", statut, err)
	}
	return n, nil
}

func isUniqueViolation(err error) bool {
	var se *sqlite.Error
	if !errors.As(err, &se) {
		return false
	}
	code := se.Code()
	return code == sqlitePrimaryKey || code == sqliteUnique
}
