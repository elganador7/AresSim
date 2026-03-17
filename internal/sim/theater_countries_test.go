package sim

import "testing"

func TestCountryCodeForPoint(t *testing.T) {
	if got := CountryCodeForPoint(31.5, 35.0); got != "ISR" {
		t.Fatalf("expected ISR, got %q", got)
	}
}

func TestCountriesAlongSegment(t *testing.T) {
	countries := CountriesAlongSegment(31.50, 35.00, 31.40, 36.00)
	foundISR := false
	foundJOR := false
	for _, code := range countries {
		if code == "ISR" {
			foundISR = true
		}
		if code == "JOR" {
			foundJOR = true
		}
	}
	if !foundISR {
		t.Fatalf("expected route to include ISR, got %#v", countries)
	}
	if !foundJOR {
		t.Fatalf("expected route to include JOR, got %#v", countries)
	}
}
