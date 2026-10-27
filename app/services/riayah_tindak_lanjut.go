package services

import (
	"sort"
	"sync"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
)

const (
	// riayahSudahDihubungiHari: santri yang dihubungi dalam rentang ini tampil di tab "Sudah dihubungi".
	riayahSudahDihubungiHari = 14
	// jumlahAlpaTTL membatasi hitung ulang badge menu yang dimuat di setiap halaman.
	jumlahAlpaTTL = time.Minute
)

// jumlahAlpaCache menyimpan jumlah antrian tindak lanjut alpa sebentar.
// Direset setiap kontak dicatat atau dihapus agar badge langsung berkurang.
type jumlahAlpaCache struct {
	mu     sync.Mutex
	jumlah int
	sampai time.Time
}

func (c *jumlahAlpaCache) reset() {
	c.mu.Lock()
	c.sampai = time.Time{}
	c.mu.Unlock()
}

// ListTindakLanjutAlpa mengembalikan antrian Admin Kelas: santri alpa
// berturut-turut yang belum dihubungi, dan yang sudah dihubungi belakangan ini.
func (s *RiayahSantriService) ListTindakLanjutAlpa(today time.Time) (perlu, sudah []models.RiayahSantriItem, err error) {
	items, _, err := s.ListPerhatian(models.RiayahViewer{}, today)
	if err != nil {
		return nil, nil, err
	}
	batas := today.AddDate(0, 0, -riayahSudahDihubungiHari).Format("2006-01-02")
	perlu = []models.RiayahSantriItem{}
	sudah = []models.RiayahSantriItem{}
	for _, it := range items {
		tl := it.TindakLanjutAlpa
		switch {
		case tl == nil:
		case tl.Status == models.TindakLanjutPerlu:
			perlu = append(perlu, it)
		case tl.KontakTanggal >= batas:
			sudah = append(sudah, it)
		}
	}
	// Paling lama tidak hadir dulu: alpa terbanyak, lalu alpa terakhir terlama.
	sort.SliceStable(perlu, func(i, j int) bool {
		a, b := perlu[i].TindakLanjutAlpa, perlu[j].TindakLanjutAlpa
		if a.AlpaBeruntun != b.AlpaBeruntun {
			return a.AlpaBeruntun > b.AlpaBeruntun
		}
		return a.AlpaTerakhir < b.AlpaTerakhir
	})
	sort.SliceStable(sudah, func(i, j int) bool {
		return sudah[i].TindakLanjutAlpa.KontakTanggal > sudah[j].TindakLanjutAlpa.KontakTanggal
	})
	s.jumlahAlpa.mu.Lock()
	s.jumlahAlpa.jumlah = len(perlu)
	s.jumlahAlpa.sampai = time.Now().Add(jumlahAlpaTTL)
	s.jumlahAlpa.mu.Unlock()
	return perlu, sudah, nil
}

// JumlahTindakLanjutAlpa adalah jumlah santri yang perlu dihubungi Admin Kelas.
func (s *RiayahSantriService) JumlahTindakLanjutAlpa(today time.Time) (int, error) {
	s.jumlahAlpa.mu.Lock()
	if time.Now().Before(s.jumlahAlpa.sampai) {
		n := s.jumlahAlpa.jumlah
		s.jumlahAlpa.mu.Unlock()
		return n, nil
	}
	s.jumlahAlpa.mu.Unlock()
	perlu, _, err := s.ListTindakLanjutAlpa(today)
	if err != nil {
		return 0, err
	}
	return len(perlu), nil
}

// ListTemplateWATindakLanjut menaruh template menanyakan ketidakhadiran di depan
// template riayah biasa.
func (s *RiayahSantriService) ListTemplateWATindakLanjut() []models.RiayahWATemplate {
	alpa := models.RiayahWATemplate{
		Nama: "Menanyakan ketidakhadiran",
		Body: "Assalamu'alaikum {nama},\n\nSemoga senantiasa sehat dan dalam lindungan Allah. Kami dari admin Ruang Sanad memperhatikan {nama} belum hadir di kelas {nama_kelas} ({jadwal}) beberapa pertemuan terakhir.\n\nApakah ada kendala yang bisa kami bantu? Kami tunggu kabarnya.\n\nJazakumullahu khairan.",
	}
	return append([]models.RiayahWATemplate{alpa}, s.ListTemplateWA()...)
}
