package services

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

var legacySantriCSVHeader = []string{"kelas_kode", "nama", "jenis_kelamin", "nominal", "tanggal_daftar", "angkatan", "usia", "domisili"}
var currentSantriCSVHeader = []string{"kelas_kode", "nama", "no_whatsapp", "email", "jenis_kelamin", "nominal", "tanggal_daftar", "angkatan", "usia", "domisili"}

type ImportService struct {
	querier *queries.Querier
	santri  *SantriService
}

func NewImportService(querier *queries.Querier, santri *SantriService) *ImportService {
	return &ImportService{querier: querier, santri: santri}
}

func (s *ImportService) ProcessCSV(r io.Reader, userID int64, namaFile string) (*models.ImportResult, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("baca CSV gagal: %w", err)
	}

	if len(records) < 1 {
		return &models.ImportResult{Total: 0, Berhasil: 0, Gagal: 0, Catatan: "File kosong"}, nil
	}
	header := make([]string, len(records[0]))
	for i, cell := range records[0] {
		header[i] = strings.ToLower(strings.TrimSpace(cell))
	}
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\ufeff")
	}
	if !slices.Equal(header, legacySantriCSVHeader) && !slices.Equal(header, currentSantriCSVHeader) {
		return nil, errors.New("header CSV tidak didukung; gunakan format template 8 atau 10 kolom")
	}
	expectedColumns := len(header)

	dataRows := records[1:]
	total := len(dataRows)
	berhasil := 0
	gagal := 0
	var catatanParts []string

	for i, row := range dataRows {
		if len(row) != expectedColumns {
			gagal++
			catatanParts = append(catatanParts, fmt.Sprintf("Baris %d: jumlah kolom %d, seharusnya %d", i+2, len(row), expectedColumns))
			continue
		}

		noWA, email := "", ""
		genderIndex, nominalIndex, tanggalDaftarIndex := 2, 3, 4
		angkatanIndex, usiaIndex, domisiliIndex := 5, 6, 7
		if expectedColumns == 10 {
			noWA, email = strings.TrimSpace(row[2]), strings.TrimSpace(row[3])
			genderIndex, nominalIndex, tanggalDaftarIndex = 4, 5, 6
			angkatanIndex, usiaIndex, domisiliIndex = 7, 8, 9
		}

		usia := parseInt64(strings.TrimSpace(row[usiaIndex]))
		_, err := s.santri.Create(models.CreateSantriRequest{
			KelasKode:     strings.TrimSpace(row[0]),
			Nama:          row[1],
			JenisKelamin:  row[genderIndex],
			Nominal:       parseNominal(strings.TrimSpace(row[nominalIndex])),
			TanggalDaftar: row[tanggalDaftarIndex],
			Angkatan:      row[angkatanIndex],
			Usia:          usia,
			Domisili:      strings.TrimSpace(row[domisiliIndex]),
			NoWA:          noWA,
			Email:         email,
		}, userID)
		if err != nil {
			gagal++
			catatanParts = append(catatanParts, fmt.Sprintf("Baris %d: %s", i+2, err.Error()))
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

func parseInt64(s string) int64 {
	n := int64(0)
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0
	}
	if n < 1 {
		return 0
	}
	return n
}
