package main

import (
	"testing"
	"time"
)

func TestNextMetaAdsSyncTimeWIB(t *testing.T) {
	cases := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{"before two WIB", time.Date(2026, 11, 6, 18, 0, 0, 0, time.UTC), time.Date(2026, 11, 6, 19, 0, 0, 0, time.UTC)},
		{"exactly two WIB", time.Date(2026, 11, 6, 19, 0, 0, 0, time.UTC), time.Date(2026, 11, 7, 19, 0, 0, 0, time.UTC)},
		{"after two WIB", time.Date(2026, 11, 7, 2, 30, 0, 0, time.UTC), time.Date(2026, 11, 7, 19, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := nextMetaAdsSyncTime(tc.now); !got.Equal(tc.want) {
				t.Fatalf("nextMetaAdsSyncTime(%v) = %v, want %v", tc.now, got, tc.want)
			}
		})
	}
}
