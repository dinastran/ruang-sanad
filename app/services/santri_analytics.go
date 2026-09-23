package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

type SantriAnalyticsService struct {
	querier *queries.Querier
}

func NewSantriAnalyticsService(querier *queries.Querier) *SantriAnalyticsService {
	return &SantriAnalyticsService{querier: querier}
}

func NormalizeSantriAnalyticsFilters(filters models.SantriAnalyticsFilters) (models.SantriAnalyticsFilters, error) {
	filters.DateFrom = strings.TrimSpace(filters.DateFrom)
	filters.DateTo = strings.TrimSpace(filters.DateTo)
	filters.AngkatanPendaftaran = strings.TrimSpace(filters.AngkatanPendaftaran)
	filters.AngkatanKelas = strings.TrimSpace(filters.AngkatanKelas)
	filters.Level = strings.TrimSpace(filters.Level)
	filters.Tipe = strings.TrimSpace(filters.Tipe)
	filters.SegmentAgeBucket = strings.TrimSpace(filters.SegmentAgeBucket)
	filters.SegmentGender = strings.TrimSpace(filters.SegmentGender)
	filters.SegmentDomisili = strings.TrimSpace(filters.SegmentDomisili)
	filters.SegmentStatus = strings.TrimSpace(filters.SegmentStatus)
	filters.SegmentRegistrationMo = strings.TrimSpace(filters.SegmentRegistrationMo)

	for label, raw := range map[string]string{"tanggal mulai": filters.DateFrom, "tanggal akhir": filters.DateTo} {
		if raw == "" {
			continue
		}
		if _, err := time.Parse(time.DateOnly, raw); err != nil {
			return filters, fmt.Errorf("%s tidak valid", label)
		}
	}
	if filters.DateFrom != "" && filters.DateTo != "" && filters.DateFrom > filters.DateTo {
		return filters, errors.New("tanggal mulai tidak boleh setelah tanggal akhir")
	}

	if filters.SegmentAgeBucket != "" {
		valid := map[string]bool{"le17": true, "18-24": true, "25-34": true, "35-44": true, "45-54": true, "55plus": true}
		if !valid[filters.SegmentAgeBucket] {
			return filters, errors.New("filter usia tidak valid")
		}
	}
	if filters.SegmentGender != "" && filters.SegmentGender != "L" && filters.SegmentGender != "P" && filters.SegmentGender != "unknown" {
		return filters, errors.New("filter jenis kelamin tidak valid")
	}
	if filters.SegmentStatus != "" {
		valid := map[string]bool{"aktif": true, "cuti": true, "nonaktif": true, "tidak_lanjut": true}
		if !valid[filters.SegmentStatus] {
			return filters, errors.New("filter status tidak valid")
		}
	}
	if filters.SegmentRegistrationMo != "" {
		if _, err := time.Parse("2006-01", filters.SegmentRegistrationMo); err != nil {
			return filters, errors.New("filter bulan pendaftaran tidak valid")
		}
	}
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.Limit <= 0 || filters.Limit > 100 {
		filters.Limit = 25
	}
	return filters, nil
}

func (s *SantriAnalyticsService) Get(filters models.SantriAnalyticsFilters) (*models.SantriAnalyticsData, error) {
	filters, err := NormalizeSantriAnalyticsFilters(filters)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	summary, err := s.querier.SantriAnalyticsSummary(ctx, filters)
	if err != nil {
		return nil, err
	}
	if summary.Total > 0 {
		summary.ActiveRate = float64(summary.Aktif) / float64(summary.Total) * 100
	}

	growth, err := s.querier.SantriAnalyticsGrowth(ctx, filters)
	if err != nil {
		return nil, err
	}
	growth = fillAnalyticsMonths(growth, filters.DateFrom, filters.DateTo)

	age, err := s.querier.SantriAnalyticsAge(ctx, filters)
	if err != nil {
		return nil, err
	}
	ageLabels := map[string]string{"le17": "≤17", "18-24": "18-24", "25-34": "25-34", "35-44": "35-44", "45-54": "45-54", "55plus": "55+"}
	for i := range age {
		age[i].Label = ageLabels[age[i].Value]
	}

	gender, err := s.querier.SantriAnalyticsGender(ctx, filters)
	if err != nil {
		return nil, err
	}
	genderLabels := map[string]string{"P": "Perempuan", "L": "Laki-laki", "unknown": "Tidak diketahui"}
	for i := range gender {
		gender[i].Label = genderLabels[gender[i].Value]
	}

	quality, err := s.querier.SantriAnalyticsDataQuality(ctx, filters)
	if err != nil {
		return nil, err
	}
	domisili, err := s.querier.SantriAnalyticsDomisili(ctx, filters, 10)
	if err != nil {
		return nil, err
	}
	var topDomisiliTotal int64
	for i := range domisili {
		domisili[i].Label = domisili[i].Value
		topDomisiliTotal += domisili[i].Total
	}
	if other := summary.Total - quality.DomisiliKosong - topDomisiliTotal; other > 0 {
		domisili = append(domisili, models.SantriAnalyticsPoint{Label: "Lainnya", Total: other})
	}

	status, err := s.querier.SantriAnalyticsStatus(ctx, filters)
	if err != nil {
		return nil, err
	}
	statusLabels := map[string]string{"aktif": "Aktif", "cuti": "Cuti", "nonaktif": "Nonaktif", "tidak_lanjut": "Tidak Lanjut", "unknown": "Tidak diketahui"}
	for i := range status {
		if label := statusLabels[status[i].Value]; label != "" {
			status[i].Label = label
		}
	}

	cohortStatus, err := s.querier.SantriAnalyticsStatusByAngkatan(ctx, filters)
	if err != nil {
		return nil, err
	}
	level, err := s.querier.SantriAnalyticsLevel(ctx, filters)
	if err != nil {
		return nil, err
	}
	levelNames := map[string]string{}
	if levels, err := s.querier.ListLevel(ctx); err == nil {
		for _, item := range levels {
			levelNames[item.Kode] = item.Nama
		}
	}
	for i := range level {
		if name := levelNames[level[i].Value]; name != "" {
			level[i].Label = name
		}
	}

	tipe, err := s.querier.SantriAnalyticsTipe(ctx, filters)
	if err != nil {
		return nil, err
	}
	for i := range tipe {
		tipe[i].Label = tipe[i].Value
	}

	rowTotal, err := s.querier.SantriAnalyticsDetailCount(ctx, filters)
	if err != nil {
		return nil, err
	}
	rows, err := s.querier.SantriAnalyticsDetail(ctx, filters)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if name := levelNames[rows[i].Level]; name != "" {
			rows[i].Level = name
		}
	}

	return &models.SantriAnalyticsData{
		Summary: summary, Growth: growth, AgeDistribution: age, GenderDistribution: gender,
		DomisiliDistribution: domisili, StatusDistribution: status, StatusByAngkatan: cohortStatus,
		LevelDistribution: level, TipeDistribution: tipe, DataQuality: quality, Rows: rows, RowTotal: rowTotal,
	}, nil
}

func fillAnalyticsMonths(points []models.SantriAnalyticsGrowthPoint, dateFrom, dateTo string) []models.SantriAnalyticsGrowthPoint {
	if len(points) == 0 && (dateFrom == "" || dateTo == "") {
		return points
	}
	values := make(map[string]int64, len(points))
	for _, point := range points {
		values[point.Month] = point.Total
	}

	var start, end time.Time
	if dateFrom != "" {
		if parsed, err := time.Parse(time.DateOnly, dateFrom); err == nil {
			start = time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC)
		}
	}
	if start.IsZero() && len(points) > 0 {
		start, _ = time.Parse("2006-01", points[0].Month)
	}
	if dateTo != "" {
		if parsed, err := time.Parse(time.DateOnly, dateTo); err == nil {
			end = time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC)
		}
	}
	if end.IsZero() && len(points) > 0 {
		end, _ = time.Parse("2006-01", points[len(points)-1].Month)
	}
	if start.IsZero() || end.IsZero() || start.After(end) {
		return points
	}

	out := make([]models.SantriAnalyticsGrowthPoint, 0)
	for month := start; !month.After(end); month = month.AddDate(0, 1, 0) {
		key := month.Format("2006-01")
		out = append(out, models.SantriAnalyticsGrowthPoint{Month: key, Total: values[key]})
	}
	return out
}
