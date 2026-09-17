package services

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

type RiayahService struct {
	querier *queries.Querier
}

func NewRiayahService(querier *queries.Querier) *RiayahService {
	return &RiayahService{querier: querier}
}

func (s *RiayahService) ListRiayahByGuru(guruID int64) ([]models.RiayahResponse, error) {
	list, err := s.querier.ListRiayahByGuru(context.Background(), sql.NullInt64{Int64: guruID, Valid: true})
	if err != nil {
		return nil, err
	}
	out := make([]models.RiayahResponse, 0, len(list))
	for _, r := range list {
		out = append(out, models.RiayahResponse{
			ID:          r.ID,
			TargetType:  r.TargetType,
			TargetID:    r.TargetID,
			TargetNama:  r.TargetNama,
			Catatan:     r.Catatan,
			PenulisNama: r.PenulisNama,
			CreatedAt:   r.CreatedAt.Format("2006-01-02 15:04"),
		})
	}
	return out, nil
}

func (s *RiayahService) ListRiayahByTarget(targetType string, targetID int64) ([]models.RiayahResponse, error) {
	list, err := s.querier.ListRiayahByTarget(context.Background(), queries.ListRiayahByTargetParams{
		TargetType: targetType,
		TargetID:   targetID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]models.RiayahResponse, 0, len(list))
	for _, r := range list {
		out = append(out, models.RiayahResponse{
			ID:          r.ID,
			TargetType:  r.TargetType,
			TargetID:    r.TargetID,
			Catatan:     r.Catatan,
			PenulisNama: r.PenulisNama,
			CreatedAt:   r.CreatedAt.Format("2006-01-02 15:04"),
		})
	}
	return out, nil
}

func (s *RiayahService) CreateRiayah(guruID, userID int64, req models.RiayahInput) (*models.RiayahResponse, error) {
	result, err := s.querier.CreateRiayah(context.Background(), queries.CreateRiayahParams{
		GuruID:       sql.NullInt64{Int64: guruID, Valid: true},
		AuthorUserID: sql.NullInt64{Int64: userID, Valid: true},
		TargetType:   req.TargetType,
		TargetID:     req.TargetID,
		Catatan:      req.Catatan,
	})
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &models.RiayahResponse{
		ID:         id,
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		Catatan:    req.Catatan,
		CreatedAt:  time.Now().Format("2006-01-02 15:04"),
	}, nil
}

// CreateUserRiayah records a note from an authorized user without a linked guru profile.
func (s *RiayahService) CreateUserRiayah(userID int64, req models.RiayahInput) (*models.RiayahResponse, error) {
	result, err := s.querier.CreateUserRiayah(context.Background(), queries.CreateUserRiayahParams{
		AuthorUserID: sql.NullInt64{Int64: userID, Valid: true},
		TargetType:   req.TargetType,
		TargetID:     req.TargetID,
		Catatan:      req.Catatan,
	})
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &models.RiayahResponse{
		ID:         id,
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		Catatan:    req.Catatan,
		CreatedAt:  time.Now().Format("2006-01-02 15:04"),
	}, nil
}

func (s *RiayahService) UpdateRiayah(riayahID, guruID int64, catatan string) error {
	return s.querier.UpdateRiayah(context.Background(), queries.UpdateRiayahParams{
		Catatan: catatan,
		ID:      riayahID,
		GuruID:  sql.NullInt64{Int64: guruID, Valid: true},
	})
}

func (s *RiayahService) EnsureRiayahTarget(riayahID, guruID, santriID int64) error {
	riayah, err := s.querier.GetRiayahByID(context.Background(), riayahID)
	if err != nil {
		return err
	}
	if !riayah.GuruID.Valid || riayah.GuruID.Int64 != guruID || !isSantriRiayah(riayah, santriID) {
		return fmt.Errorf("catatan riayah tidak ditemukan")
	}
	return nil
}

func (s *RiayahService) EnsureRiayahSantriTarget(riayahID, santriID int64) error {
	riayah, err := s.querier.GetRiayahByID(context.Background(), riayahID)
	if err != nil || !isSantriRiayah(riayah, santriID) {
		return fmt.Errorf("catatan riayah tidak ditemukan")
	}
	return nil
}

func isSantriRiayah(riayah queries.CatatanRiayah, santriID int64) bool {
	return riayah.TargetType == "santri" && riayah.TargetID == santriID
}

func (s *RiayahService) DeleteRiayah(riayahID, guruID int64) error {
	return s.querier.DeleteRiayah(context.Background(), queries.DeleteRiayahParams{
		ID:     riayahID,
		GuruID: sql.NullInt64{Int64: guruID, Valid: true},
	})
}

func (s *RiayahService) UpdateRiayahByID(riayahID int64, catatan string) error {
	return s.querier.UpdateRiayahByID(context.Background(), queries.UpdateRiayahByIDParams{Catatan: catatan, ID: riayahID})
}

func (s *RiayahService) DeleteRiayahByID(riayahID int64) error {
	return s.querier.DeleteRiayahByID(context.Background(), riayahID)
}

func (s *RiayahService) GetLinkWA(noWa, templateName string, vars map[string]string) string {
	return getWALink(noWa, templateName, vars)
}

func (s *RiayahService) GetBroadcastLinks(kelasID int64, templateName string) ([]map[string]string, error) {
	santri, err := s.querier.GetSantriByKelasID(context.Background(), sql.NullInt64{Int64: kelasID, Valid: true})
	if err != nil {
		return nil, err
	}
	out := make([]map[string]string, 0, len(santri))
	for _, sant := range santri {
		if sant.NoWa == "" {
			continue
		}
		link := getWALink(sant.NoWa, templateName, map[string]string{
			"nama": sant.Nama,
		})
		out = append(out, map[string]string{
			"nama": sant.Nama,
			"wa":   link,
		})
	}
	return out, nil
}

func getWALink(noWa, templateName string, vars map[string]string) string {
	phone := noWa
	if phone == "" {
		return ""
	}
	if phone[0] == '0' {
		phone = "62" + phone[1:]
	} else if phone[0] != '+' {
		phone = "+62" + phone
	}

	text := getTemplateText(templateName, vars)
	return fmt.Sprintf("https://wa.me/%s?text=%s", url.PathEscape(phone), url.QueryEscape(text))
}

func getTemplateText(templateName string, vars map[string]string) string {
	templates := map[string]string{
		"default": "Assalamu'alaikum *{{nama}}*,\n\nIni adalah pesan dari Ruang Sanad.",
		"riayah":  "Assalamu'alaikum *{{nama}}*,\n\nKami ingin memberikan catatan riayah untuk ananda.\n\nJazakumullah khairan katsiran.",
	}
	text, ok := templates[templateName]
	if !ok {
		text = templates["default"]
	}
	for k, v := range vars {
		placeholder := "{{" + k + "}}"
		text = replaceAllString(text, placeholder, v)
	}
	return text
}

func replaceAllString(s, old, new string) string {
	result := make([]byte, 0, len(s))
	i := 0
	for i <= len(s)-len(old) {
		if s[i:i+len(old)] == old {
			result = append(result, []byte(new)...)
			i += len(old)
		} else {
			result = append(result, s[i])
			i++
		}
	}
	if i < len(s) {
		result = append(result, s[i:]...)
	}
	return string(result)
}
