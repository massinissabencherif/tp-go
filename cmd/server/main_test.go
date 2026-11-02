package main

import "testing"

func TestGetenv(t *testing.T) {
	t.Setenv("TP_GO_TEST_VAR", "valeur")
	tests := []struct {
		name     string
		key      string
		fallback string
		want     string
	}{
		{"variable définie", "TP_GO_TEST_VAR", "defaut", "valeur"},
		{"variable absente", "TP_GO_TEST_ABSENTE", "defaut", "defaut"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getenv(tt.key, tt.fallback); got != tt.want {
				t.Errorf("getenv() = %q, attendu %q", got, tt.want)
			}
		})
	}
}
