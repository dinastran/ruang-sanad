package services

import (
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func TestOnboardingDeadlineCalendarWIB(t *testing.T) {
	start := time.Date(2026, 10, 29, 23, 59, 0, 0, onboardingWIB)
	require.Equal(t, "2026-11-13T00:00:00+07:00", OnboardingDeadline(start).In(onboardingWIB).Format(time.RFC3339))
}

func TestSantriCreationOpensOnboardingAtomically(t *testing.T) {
	db, _, service, userID := setupSantriIntegrityService(t)
	santri, err := service.Create(validCreateSantriRequest(), userID)
	require.NoError(t, err)
	var id int64
	var stage string
	require.NoError(t, db.QueryRow("SELECT id, stage FROM onboarding_cases WHERE santri_id = ?", santri.ID).Scan(&id, &stage))
	require.Equal(t, OnboardingStageBaru, stage)
	require.NotZero(t, id)
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM onboarding_activity_logs WHERE case_id = ? AND event_type = 'onboarding.created'", id).Scan(&count))
	require.Equal(t, 1, count)
	_, err = db.Exec("INSERT INTO onboarding_cases (santri_id, registered_at, deadline_at, created_at, updated_at) VALUES (?, CURRENT_TIMESTAMP, datetime('now', '+15 days'), CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)", santri.ID)
	require.Error(t, err, "unique santri must not have duplicate cases")
}

func TestInvalidSantriCreationDoesNotOpenOnboarding(t *testing.T) {
	db, _, service, userID := setupSantriIntegrityService(t)
	req := models.CreateSantriRequest{Nama: "", TanggalDaftar: "2026-10-29", Angkatan: "AKA38"}
	_, err := service.Create(req, userID)
	require.Error(t, err)
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM onboarding_cases").Scan(&count))
	require.Zero(t, count)
}

func TestOnboardingListDoesNotBackfillOldSantri(t *testing.T) {
	db, q, service, userID := setupSantriIntegrityService(t)
	_, err := db.Exec("INSERT INTO santri (nama, jenis_kelamin, angkatan, status) VALUES ('Historis','L','AKA38','aktif')")
	require.NoError(t, err)
	created, err := service.Create(validCreateSantriRequest(), userID)
	require.NoError(t, err)
	rows, err := NewOnboardingService(q).List(t.Context())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, created.ID, rows[0].SantriID)
}

func TestOnboardingInsertFailureRollsBackSantri(t *testing.T) {
	db, _, service, userID := setupSantriIntegrityService(t)
	_, err := db.Exec(`CREATE TRIGGER reject_onboarding BEFORE INSERT ON onboarding_cases
 BEGIN SELECT RAISE(ABORT, 'onboarding unavailable'); END`)
	require.NoError(t, err)
	_, err = service.Create(validCreateSantriRequest(), userID)
	require.ErrorContains(t, err, "onboarding unavailable")
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM santri").Scan(&count))
	require.Zero(t, count, "santri write must roll back when onboarding fails")
}

func TestOnboardingRegisteredAtUsesActualCreationNotHistoricalInput(t *testing.T) {
	db, _, service, userID := setupSantriIntegrityService(t)
	before := time.Now().UTC().Add(-time.Second)
	santri, err := service.Create(validCreateSantriRequest(), userID)
	require.NoError(t, err)
	var registered time.Time
	require.NoError(t, db.QueryRow("SELECT registered_at FROM onboarding_cases WHERE santri_id = ?", santri.ID).Scan(&registered))
	require.False(t, registered.Before(before))
	require.True(t, registered.Before(time.Now().UTC().Add(time.Second)))
}

func TestImportedHistoricalSantriSkipsOnboarding(t *testing.T) {
	db, _, service, userID := setupSantriIntegrityService(t)
	santri, err := service.CreateImported(validCreateSantriRequest(), userID)
	require.NoError(t, err)
	require.NotZero(t, santri.ID)
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM onboarding_cases").Scan(&count))
	require.Zero(t, count)
}
