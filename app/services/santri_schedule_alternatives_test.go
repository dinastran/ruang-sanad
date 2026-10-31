package services

import (
	"errors"
	"testing"
)

func TestValidateScheduleAlternativeConsent(t *testing.T) {
	cases := []struct {
		name  string
		in    SantriScheduleAlternativeInput
		valid bool
	}{
		{"proposed", SantriScheduleAlternativeInput{JadwalKode: "S1", Status: "proposed"}, true},
		{"approved", SantriScheduleAlternativeInput{JadwalKode: "S1", Status: "approved", ConsentChannel: "whatsapp", ConsentNote: "Konfirmasi tertulis di chat tanggal 31 Oktober"}, true},
		{"revoked", SantriScheduleAlternativeInput{JadwalKode: "S1", Status: "revoked", ConsentNote: "Santri membatalkan pilihan alternatif"}, true},
		{"approved without evidence", SantriScheduleAlternativeInput{JadwalKode: "S1", Status: "approved", ConsentChannel: "phone"}, false},
		{"approved without channel", SantriScheduleAlternativeInput{JadwalKode: "S1", Status: "approved", ConsentNote: "Setuju"}, false},
		{"revoked without reason", SantriScheduleAlternativeInput{JadwalKode: "S1", Status: "revoked"}, false},
		{"empty offer", SantriScheduleAlternativeInput{Status: "proposed"}, false},
		{"invalid status", SantriScheduleAlternativeInput{JadwalKode: "S1", Status: "confirmed"}, false},
		{"unconfirmed cannot have confirmed channel", SantriScheduleAlternativeInput{JadwalKode: "S1", Status: "proposed", ConsentChannel: "whatsapp"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := tc.in
			err := validateAlternative(&in)
			if tc.valid && err != nil {
				t.Fatalf("rejected valid input: %v", err)
			}
			if !tc.valid && !errors.Is(err, ErrAlternativeInvalid) {
				t.Fatalf("expected invalid error: %v", err)
			}
		})
	}
}

func TestScheduleAlternativeTransitionNeedsRevocation(t *testing.T) {
	if err := validateAlternativeTransition("approved", "proposed"); !errors.Is(err, ErrAlternativeInvalid) {
		t.Fatalf("must not silently downgrade approved consent: %v", err)
	}
	if err := validateAlternativeTransition("approved", "revoked"); err != nil {
		t.Fatal(err)
	}
	if err := validateAlternativeTransition("", "revoked"); !errors.Is(err, ErrAlternativeInvalid) {
		t.Fatalf("unknown alternative cannot be revoked: %v", err)
	}
}
