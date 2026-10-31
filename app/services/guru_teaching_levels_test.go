package services

import (
	"errors"
	"reflect"
	"testing"
)

func TestTeachingLevelsNormalize(t *testing.T) {
	master := map[string]bool{"1": true, "2": true}
	codes, err := normalizeApprovedLevels([]string{" 2 ", "1"}, master)
	if err != nil || !reflect.DeepEqual(codes, []string{"1", "2"}) {
		t.Fatalf("valid level set: %v, %v", codes, err)
	}
	for _, tc := range []struct {
		name  string
		codes []string
	}{
		{"unknown", []string{"bad"}},
		{"duplicate", []string{"1", "1"}},
		{"empty", []string{""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := normalizeApprovedLevels(tc.codes, master)
			if !errors.Is(err, ErrTeachingLevelInvalid) {
				t.Fatalf("expected invalid: %v", err)
			}
		})
	}
	codes, err = normalizeApprovedLevels([]string{}, master)
	if err != nil || len(codes) != 0 {
		t.Fatalf("revocation should accept empty set: %v, %v", codes, err)
	}
}
