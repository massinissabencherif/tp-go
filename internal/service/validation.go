package service

import (
	"regexp"
	"strings"
	"time"

	"github.com/massinissabencherif/tp-go/internal/apperr"
	"github.com/massinissabencherif/tp-go/internal/model"
)

var (
	plaqueRe    = regexp.MustCompile(`^[A-Z]{2}-[0-9]{3}-[A-Z]{2}$`)
	telephoneRe = regexp.MustCompile(`^\+?[0-9][0-9 .-]{5,19}$`)
)

const premiereAnnee = 1886

func validatePlaque(plaque string) error {
	if !plaqueRe.MatchString(plaque) {
		return apperr.Invalid("plaque", "format attendu AA-123-AA")
	}
	return nil
}

func validateVehicule(v model.Vehicule, now time.Time) error {
	if err := validatePlaque(v.Plaque); err != nil {
		return err
	}
	if v.Marque == "" {
		return apperr.Invalid("marque", "obligatoire")
	}
	if v.Modele == "" {
		return apperr.Invalid("modele", "obligatoire")
	}
	if v.Annee < premiereAnnee || v.Annee > now.Year()+1 {
		return apperr.Invalid("annee", "année de mise en service invalide")
	}
	if v.Kilometrage < 0 {
		return apperr.Invalid("kilometrage", "ne peut pas être négatif")
	}
	if v.Nom == "" {
		return apperr.Invalid("proprietaire.nom", "obligatoire")
	}
	if !telephoneRe.MatchString(v.Telephone) {
		return apperr.Invalid("proprietaire.telephone", "numéro de téléphone invalide")
	}
	return nil
}

func validateIntervention(i model.Intervention) error {
	if _, err := time.Parse("2006-01-02", i.Date); err != nil {
		return apperr.Invalid("date", "format attendu YYYY-MM-DD")
	}
	if strings.TrimSpace(i.Description) == "" {
		return apperr.Invalid("description", "obligatoire")
	}
	if !i.Statut.Valid() {
		return apperr.Invalid("statut", "valeurs possibles : en_cours, terminee, annulee")
	}
	return nil
}
