package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/stretchr/testify/require"
)

func TestOnboardingBoardSearchAcrossPaginationAndPIC(t *testing.T) {
	db, service, firstID, picID, actorID := onboardingFixture(t)
	ctx := context.Background()
	require.NoError(t, service.AssignPIC(ctx, firstID, picID, actorID))

	for i := 0; i < 125; i++ {
		name := fmt.Sprintf("Santri Reguler %03d", i)
		if i == 0 { name = "Cari_100%Uji" }
		result, err := db.Exec("INSERT INTO santri (nama,jenis_kelamin,angkatan,status) VALUES (?,'L','AKA38','aktif')", name)
		require.NoError(t, err)
		id, err := result.LastInsertId()
		require.NoError(t, err)
		_, err = db.Exec("INSERT INTO onboarding_cases (santri_id,stage,registered_at,deadline_at,created_at,updated_at) VALUES (?,'onboarding_baru',CURRENT_TIMESTAMP,datetime('now','+15 days'),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)", id)
		require.NoError(t, err)
	}

	board, err := service.Board(ctx, 1)
	require.NoError(t, err)
	require.EqualValues(t, 126, board.Total)
	require.Len(t, board.Cases, 100)
	board2, err := service.Board(ctx, 2)
	require.NoError(t, err)
	require.Len(t, board2.Cases, 26)

	filtered, err := service.BoardFiltered(ctx, 1, queries.OnboardingBoardFilter{Query:"Cari_100%Uji"})
	require.NoError(t, err)
	require.EqualValues(t, 1, filtered.Total, "underscores and percent signs are literal")
	require.Len(t, filtered.Cases, 1)
	require.Equal(t, "Cari_100%Uji", filtered.Cases[0].Nama)
	require.EqualValues(t, 1, filtered.Counts["onboarding_baru"])

	byPIC, err := service.BoardFiltered(ctx, 1, queries.OnboardingBoardFilter{PICID:picID})
	require.NoError(t, err)
	require.EqualValues(t, 1, byPIC.Total)
	unassigned, err := service.BoardFiltered(ctx, 1, queries.OnboardingBoardFilter{UnassignedPIC:true})
	require.NoError(t, err)
	require.EqualValues(t, 125, unassigned.Total)

	empty, err := service.BoardFiltered(ctx, 1, queries.OnboardingBoardFilter{Query:"tidak ada mahasantri ini"})
	require.NoError(t, err)
	require.Zero(t, empty.Total)
	require.Empty(t, empty.Cases)
	require.Empty(t, empty.Counts)
}

func TestOnboardingBoardSLAFiltersExcludeTerminal(t *testing.T) {
	db, service, caseID, _, _ := onboardingFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 10, 10, 0, 0, 0, time.FixedZone("WIB", 7*3600))
	_, err := db.Exec("UPDATE onboarding_cases SET deadline_at=? WHERE id=?", now.Add(-time.Hour).UTC(), caseID)
	require.NoError(t, err)

	overdue, err := service.BoardFiltered(ctx, 1, queries.OnboardingBoardFilter{SLA:"overdue",Now:now})
	require.NoError(t, err)
	require.EqualValues(t, 1, overdue.Total)

	_, err = db.Exec("UPDATE onboarding_cases SET stage='batal' WHERE id=?", caseID)
	require.NoError(t, err)
	excluded, err := service.BoardFiltered(ctx, 1, queries.OnboardingBoardFilter{SLA:"overdue",Now:now})
	require.NoError(t, err)
	require.Zero(t, excluded.Total)
}

func TestOnboardingQuotaFilteredAndClassMemberRoster(t *testing.T) {
	db, service, kelasID, _, _, superID := quotaFixture(t, "Reguler")
	ctx := context.Background()

	missing, err := service.ClassControlsFiltered(ctx, 1, queries.OnboardingQuotaFilter{Status:"needs_target"})
	require.NoError(t, err)
	require.EqualValues(t, 1, missing.Total)
	require.Len(t, missing.Classes, 1)

	require.NoError(t, service.SetClassQuota(ctx, kelasID, superID, 3))
	shortage, err := service.ClassControlsFiltered(ctx, 1, queries.OnboardingQuotaFilter{Status:"shortage"})
	require.NoError(t, err)
	require.EqualValues(t, 1, shortage.Total)
	require.EqualValues(t, 2, shortage.Classes[0].Kekurangan)

	result, err := db.Exec("INSERT INTO santri(nama,jenis_kelamin,angkatan,status,kelas_id) VALUES ('Santri Historis','L','AKA38','cuti',?)", kelasID)
	require.NoError(t, err)
	historicID, err := result.LastInsertId()
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO santri(nama,jenis_kelamin,angkatan,status,kelas_id) VALUES ('Nonaktif','L','AKA38','nonaktif',?)",kelasID)
	require.NoError(t, err)

	roster, err := service.ClassMembers(ctx, kelasID, 1)
	require.NoError(t, err)
	require.EqualValues(t, 2, roster.Total)
	require.Len(t, roster.Members, 2)
	foundHistoric := false
	for _, m := range roster.Members {
		if m.ID == historicID {
			require.Equal(t, "cuti", m.Status)
			require.Empty(t, m.Stage, "historical santri has no onboarding case")
			foundHistoric = true
		}
	}
	require.True(t, foundHistoric)

	visible, err := service.ClassControlsFiltered(ctx, 1, queries.OnboardingQuotaFilter{Query:"Kelas PR3"})
	require.NoError(t, err)
	require.EqualValues(t, roster.Total, visible.Classes[0].JumlahSantri)
	require.EqualValues(t, 1, visible.Classes[0].KandidatAktif)
	require.NoError(t, service.SetClassQuota(ctx, kelasID, superID, 2))
	ready, err := service.ClassControlsFiltered(ctx, 1, queries.OnboardingQuotaFilter{Status:"ready"})
	require.NoError(t, err)
	require.EqualValues(t, 1, ready.Total)
	require.True(t, ready.Classes[0].CanStart)
	require.EqualValues(t, 2, ready.Classes[0].JumlahSantri)
	_, err = service.ClassMembers(ctx, 99999999, 1)
	require.ErrorIs(t, err, ErrOnboardingNotFound)
}

func TestOnboardingQuotaSearchBeyondFormer200ClassLimit(t *testing.T) {
	db, service, kelasID, _, _, _ := quotaFixture(t, "Reguler")
	ctx := context.Background()
	var firstSantri int64
	require.NoError(t, db.QueryRow("SELECT santri_id FROM onboarding_cases LIMIT 1").Scan(&firstSantri))
	_ = kelasID
	_ = firstSantri

	for i := 0; i < 205; i++ {
		name := fmt.Sprintf("Kelas Uji %03d", i)
		r, err := db.Exec("INSERT INTO kelas(kunci_kelas,angkatan,tipe,jenis_kelamin,level,frekuensi,jadwal,sub_index,nama_kelas,kapasitas,jumlah_santri,is_aktif) VALUES (?,?, 'Reguler','L','Dasar','1x','Senin',1,?,10,0,1)",fmt.Sprintf("TEST-CLASS-%03d",i),"AKA38",name)
		require.NoError(t,err)
		id,err:=r.LastInsertId()
		require.NoError(t,err)
		r,err=db.Exec("INSERT INTO santri(nama,jenis_kelamin,angkatan,status,kelas_id) VALUES (?, 'L','AKA38','aktif',?)",fmt.Sprintf("Santri Kelas %03d",i),id)
		require.NoError(t,err)
		sid,err:=r.LastInsertId()
		require.NoError(t,err)
		_,err=db.Exec("INSERT INTO onboarding_cases(santri_id,stage,registered_at,deadline_at,created_at,updated_at) VALUES (?,'onboarding_baru',CURRENT_TIMESTAMP,datetime('now','+15 days'),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)",sid)
		require.NoError(t,err)
	}
	filtered,err:=service.ClassControlsFiltered(ctx, 1, queries.OnboardingQuotaFilter{Query:"Kelas Uji 204"})
	require.NoError(t,err)
	require.EqualValues(t,1,filtered.Total)
	require.Len(t,filtered.Classes,1)
	require.Equal(t,"Kelas Uji 204",filtered.Classes[0].Nama)

	page,err:=service.ClassControlsFiltered(ctx, 9,queries.OnboardingQuotaFilter{})
	require.NoError(t,err)
	require.EqualValues(t,206,page.Total)
	require.Len(t,page.Classes,14)
}
