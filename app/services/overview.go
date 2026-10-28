package services

import (
	"context"
	"errors"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

type OverviewService struct {
	querier *queries.Querier
	finance *KeuanganDashboardService
	riayah  *RiayahSantriService
	now     func() time.Time
}

func NewOverviewService(q *queries.Querier, finance *KeuanganDashboardService, riayah *RiayahSantriService) *OverviewService {
	return &OverviewService{querier: q, finance: finance, riayah: riayah, now: HariIniRiayah}
}

// These are the same roles as kuReadRole in routes/web.go. Financial data is
// excluded at the server boundary, not merely hidden by the component.
func OverviewCanFinance(role string) bool {
	return role == "super_admin" || role == "keuangan" || role == "admin"
}

func (s *OverviewService) Get(ctx context.Context, month, role string) (*models.Overview, error) {
	if !OverviewCanFinance(role) {
		return nil, errors.New("akses dashboard tidak diizinkan")
	}
	now := s.now().In(ZonaWaktuRiayah)
	if month == "" {
		month = now.Format("2006-01")
	}
	start, err := time.ParseInLocation("2006-01", month, ZonaWaktuRiayah)
	if err != nil || start.Format("2006-01") != month || month > now.Format("2006-01") || start.Year() < 2000 {
		return nil, errors.New("pilih bulan yang valid sampai bulan berjalan")
	}
	end := start.AddDate(0, 1, -1)
	if end.After(now) {
		end = now
	}
	from, until, today := start.Format(time.DateOnly), end.Format(time.DateOnly), now.Format(time.DateOnly)
	out := &models.Overview{Bulan: month, BulanLabel: labelBulanID(start), UpdatedAt: now.Format(time.RFC3339), CurrentMonth: month == now.Format("2006-01"), Errors: []string{}, Priorities: []models.OverviewPriority{}, Growth: []models.SantriAnalyticsGrowthPoint{}, Attendance: []models.OverviewAttendanceMonth{}, Programs: []models.OverviewProgram{}}
	add := func(title, detail, href string, count int64) {
		if count > 0 {
			out.Priorities = append(out.Priorities, models.OverviewPriority{Title: title, Detail: detail, Href: href, Count: count})
		}
	}
	fail := func(source string) { out.Errors = append(out.Errors, source) }
	// Management-only sources mirror each module's existing super_admin gate.
	if role == "super_admin" {
		summary, e := s.querier.SantriAnalyticsSummary(ctx, models.SantriAnalyticsFilters{})
		if e != nil {
			fail("Status santri")
		} else {
			if summary.Total > 0 {
				summary.ActiveRate = float64(summary.Aktif) * 100 / float64(summary.Total)
			}
			out.Santri = &summary
		}
		growthFrom := start.AddDate(0, -5, 0).Format(time.DateOnly)
		growth, e := s.querier.SantriAnalyticsGrowth(ctx, models.SantriAnalyticsFilters{DateFrom: growthFrom, DateTo: until})
		if e != nil {
			fail("Tren pendaftaran")
		} else {
			out.Growth = fillAnalyticsMonths(growth, growthFrom, until)
		}
		attendance, e := s.querier.OverviewAttendance(ctx, growthFrom, until)
		if e != nil {
			fail("Kehadiran santri")
		} else {
			out.Attendance = fillAttendanceMonths(attendance, start.AddDate(0, -5, 0), start)
		}
		out.Leads, e = s.overviewLeads(ctx, start)
		if e != nil {
			fail("Lead WhatsApp")
		}
		out.Programs, e = s.querier.OverviewPrograms(ctx)
		if e != nil {
			fail("Performa program")
		}
		out.Operations, e = s.querier.OverviewOperations(ctx, today, from, until)
		if e != nil {
			fail("Operasional kelas dan guru")
		} else {
			o := out.Operations
			add("Data santri belum lengkap", "Lengkapi data santri aktif melalui Admin Kelas agar penempatan dan tindak lanjut tidak terhambat.", "/app/perlu-dilengkapi", o.Incomplete)
			add("Kelas membutuhkan guru", "Guru belum ditetapkan atau sudah tidak aktif. Periksa penugasan sebelum kelas berlangsung.", "/app/kelas", o.WithoutTeacher)
			add("Santri aktif belum punya kelas aktif", "Pastikan penempatan dan jadwal belajarnya bersama Admin Kelas.", "/app/santri", o.WithoutClass)
			add("Pertemuan lama belum ditutup", "Pertemuan berstatus berlangsung sebelum hari ini. Minta guru memeriksa dan melengkapi pencatatan.", "/app/koordinator-guru", o.OpenMeetings)
		}
		_, riayah, e := s.riayah.ListPerhatian(models.RiayahViewer{}, now)
		if e != nil {
			fail("Riayah santri")
		} else {
			out.Riayah = &riayah
			add("Santri membutuhkan perhatian", "Penanda riayah: kehadiran, kontak, atau progres. Periksa penyebab sebelum menyimpulkan risiko berhenti.", "/app/guru/riayah", int64(riayah.PerluPerhatian))
		}
		if out.Operations != nil {
			add("Tindak lanjut guru jatuh tempo", "Tindak lanjut kunjungan yang masih terbuka dan bertarget hari ini atau sebelumnya.", "/app/koordinator-guru/kunjungan", out.Operations.FollowUps)
			add("Tugas koordinator jatuh tempo", "Tugas belum selesai dengan deadline hari ini atau sebelumnya.", "/app/koordinator-guru/todo", out.Operations.Todos)
		}
		products, e := s.querier.ProductDashboardSummary(ctx, models.ProductCRMDashboardFilters{DateFrom: from, DateTo: until})
		if e != nil {
			fail("Produk dan persediaan")
		} else {
			out.Products = &products
			add("Stok produk perlu diperiksa", "Produk aktif telah mencapai batas minimum stok.", "/app/produk-crm", products.LowStockProducts)
		}
	}
	if OverviewCanFinance(role) {
		out.Finance, err = s.finance.Get(month)
		if err != nil {
			fail("Keuangan")
		} else {
			add("Tagihan melewati jatuh tempo", "Prioritaskan follow-up berdasarkan umur tunggakan dan nominal di menu Keuangan.", "/app/keuangan?bulan="+month, out.Finance.JumlahTerlambat)
		}
	}
	return out, nil
}

// fillAttendanceMonths returns one row per month from first to last month
// (inclusive), with zero rows for months without finished meetings.
func fillAttendanceMonths(rows []models.OverviewAttendanceMonth, first, last time.Time) []models.OverviewAttendanceMonth {
	byMonth := make(map[string]models.OverviewAttendanceMonth, len(rows))
	for _, r := range rows {
		byMonth[r.Month] = r
	}
	out := make([]models.OverviewAttendanceMonth, 0, 6)
	for m := first; !m.After(last); m = m.AddDate(0, 1, 0) {
		key := m.Format("2006-01")
		r, ok := byMonth[key]
		if !ok {
			r = models.OverviewAttendanceMonth{Month: key}
		}
		out = append(out, r)
	}
	return out
}

// overviewLeads counts leads and closings in the month starting at start and
// in the month before, with WIB month boundaries like the leads dashboard.
func (s *OverviewService) overviewLeads(ctx context.Context, start time.Time) (*models.OverviewLeads, error) {
	end, prev := start.AddDate(0, 1, 0), start.AddDate(0, -1, 0)
	var out models.OverviewLeads
	var err error
	if out.Masuk, err = s.querier.LeadDashboardCreatedCount(ctx, start, end); err != nil {
		return nil, err
	}
	if out.MasukSebelumnya, err = s.querier.LeadDashboardCreatedCount(ctx, prev, start); err != nil {
		return nil, err
	}
	closings, err := s.querier.ListLeadDashboardClosings(ctx, prev, end)
	if err != nil {
		return nil, err
	}
	for _, c := range closings {
		if c.Before(start) {
			out.ClosingSebelumnya++
		} else {
			out.Closing++
		}
	}
	return &out, nil
}
