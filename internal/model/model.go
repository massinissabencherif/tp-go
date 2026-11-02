package model

import "strings"

type Statut string

const (
	StatutEnCours  Statut = "en_cours"
	StatutTerminee Statut = "terminee"
	StatutAnnulee  Statut = "annulee"
)

var Statuts = []Statut{StatutEnCours, StatutTerminee, StatutAnnulee}

func (s Statut) Valid() bool {
	for _, v := range Statuts {
		if s == v {
			return true
		}
	}
	return false
}

type Proprietaire struct {
	Nom       string `json:"nom"`
	Telephone string `json:"telephone"`
}

type Vehicule struct {
	Plaque       string `json:"plaque"`
	Marque       string `json:"marque"`
	Modele       string `json:"modele"`
	Annee        int    `json:"annee"`
	Kilometrage  int    `json:"kilometrage"`
	Proprietaire `json:"proprietaire"`
}

func (v *Vehicule) Normalize() {
	v.Plaque = NormalizePlaque(v.Plaque)
	v.Marque = strings.TrimSpace(v.Marque)
	v.Modele = strings.TrimSpace(v.Modele)
	v.Nom = strings.TrimSpace(v.Nom)
	v.Telephone = strings.TrimSpace(v.Telephone)
}

func NormalizePlaque(plaque string) string {
	return strings.ToUpper(strings.TrimSpace(plaque))
}

type Intervention struct {
	ID          int64  `json:"id"`
	Plaque      string `json:"plaque"`
	Date        string `json:"date"`
	Description string `json:"description"`
	Statut      Statut `json:"statut"`
}

type Stats struct {
	NombreVehicules        int            `json:"nombre_vehicules"`
	InterventionsParStatut map[Statut]int `json:"interventions_par_statut"`
}
