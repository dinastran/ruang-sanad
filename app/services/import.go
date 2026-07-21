package services

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

type ImportService struct {
	querier *queries.Querier
	engine  *KelasEngineService
}

func NewImportService(querier *queries.Querier, engine *KelasEngineService) *ImportService {
	return &ImportService{querier: querier, engine: engine}
}

func (s *ImportService) ProcessCSV(r io.Reader, userID int64, namaFile string) (*models.ImportResult, error) {
	reader := csv.NewReader(r)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("baca CSV gagal: %w", err)
	}

	if len(records) < 1 {
		return &models.ImportResult{Total: 0, Berhasil: 0, Gagal: 0, Catatan: "File kosong"}, nil
	}

	dataRows := records[1:]
	total := len(dataRows)
	berhasil := 0
	gagal := 0
	var catatanParts []string

	for i, row := range dataRows {
		if len(row) < 8 {
			gagal++
			catatanParts = append(catatanParts, fmt.Sprintf("Baris %d: kolom tidak lengkap (%d)", i+2, len(row)))
			continue
		}

		jenisKelamin := strings.TrimSpace(strings.ToUpper(row[2]))
		if jenisKelamin != "L" && jenisKelamin != "P" {
			gagal++
			catatanParts = append(catatanParts, fmt.Sprintf("Baris %d: jenis kelamin '%s' tidak valid", i+2, row[2]))
			continue
		}

		now := time.Now()

		result, err := s.querier.CreateSantri(context.Background(), queries.CreateSantriParams{
			IDMahasantri:  "",
			KelasKode:     strings.TrimSpace(row[0]),
			Nama:          strings.TrimSpace(row[1]),
			JenisKelamin:  jenisKelamin,
			Nominal:       parseNominal(strings.TrimSpace(row[3])),
			TanggalDaftar: parseDate(strings.TrimSpace(row[4])),
			Angkatan:      strings.TrimSpace(row[5]),
			Usia:          parseInt64(strings.TrimSpace(row[6])),
			Domisili:      strings.TrimSpace(row[7]),
			Tipe:          "",
			Frekuensi:     "",
			IsLengkap:     0,
			KelasID:       sql.NullInt64{Valid: false},
			Status:        "aktif",
			CreatedBy:     sql.NullInt64{Int64: userID, Valid: true},
			CreatedAt:     now,
			UpdatedAt:     now,
		})
		if err != nil {
			gagal++
			catatanParts = append(catatanParts, fmt.Sprintf("Baris %d: %s", i+2, err.Error()))
			continue
		}

		id, err := result.LastInsertId()
		if err != nil {
			gagal++
			catatanParts = append(catatanParts, fmt.Sprintf("Baris %d: last insert id gagal", i+2))
			continue
		}

		if td := parseDate(strings.TrimSpace(row[4])); td.Valid {
			idMahasantri := fmt.Sprintf("MHS.%s.%04d.%s", strings.TrimSpace(row[5]), id, td.Time.Format("012006"))
			_ = s.querier.UpdateSantriIdMahasantri(context.Background(), queries.UpdateSantriIdMahasantriParams{
				IDMahasantri: idMahasantri,
				ID:           id,
			})
		}

		santri, err := s.querier.GetSantriByID(context.Background(), id)
		if err != nil {
			gagal++
			catatanParts = append(catatanParts, fmt.Sprintf("Baris %d: get santri gagal", i+2))
			continue
		}

		if err := s.engine.ProcessSantri(context.Background(), &santri); err != nil {
			gagal++
			catatanParts = append(catatanParts, fmt.Sprintf("Baris %d: engine process gagal", i+2))
			continue
		}

		berhasil++
	}

	_ = s.querier.CreateImportLog(context.Background(), userID, namaFile, int64(total), int64(berhasil), int64(gagal), strings.Join(catatanParts, "; "))

	result := &models.ImportResult{
		Total:    total,
		Berhasil: berhasil,
		Gagal:    gagal,
		Catatan:  strings.Join(catatanParts, "\n"),
	}
	return result, nil
}

func parseNominal(s string) int64 {
	n := int64(0)
	fmt.Sscanf(s, "%d", &n)
	return n
}

func parseDate(s string) sql.NullTime {
	if s == "" {
		return sql.NullTime{Valid: false}
	}
	formats := []string{"2006-01-02", "02/01/2006", "1/2/2006", "2006/01/02"}
	for _, f := range formats {
		t, err := time.Parse(f, s)
		if err == nil {
			return sql.NullTime{Time: t, Valid: true}
		}
	}
	return sql.NullTime{Valid: false}
}

func parseInt64(s string) sql.NullInt64 {
	n := int64(0)
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: n, Valid: n > 0}
}
