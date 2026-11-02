package model

import "testing"

func TestStatutValid(t *testing.T) {
	tests := []struct {
		statut Statut
		want   bool
	}{
		{StatutEnCours, true},
		{StatutTerminee, true},
		{StatutAnnulee, true},
		{"", false},
		{"fini", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.statut), func(t *testing.T) {
			if got := tt.statut.Valid(); got != tt.want {
				t.Errorf("Valid() = %v, attendu %v", got, tt.want)
			}
		})
	}
}

func TestVehiculeNormalize(t *testing.T) {
	v := Vehicule{
		Plaque:       "  ab-123-cd ",
		Marque:       " Renault ",
		Proprietaire: Proprietaire{Nom: " Alice ", Telephone: " 0612345678 "},
	}
	v.Normalize()
	if v.Plaque != "AB-123-CD" || v.Marque != "Renault" || v.Nom != "Alice" || v.Telephone != "0612345678" {
		t.Errorf("Normalize() = %+v", v)
	}
}
