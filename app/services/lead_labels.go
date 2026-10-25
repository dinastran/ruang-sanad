package services

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/maulanashalihin/laju-go/app/queries"
)

// A catalog is shared by CS, finance and super admins. Existing link IDs are
// retained on rename so every lead immediately uses the edited name/color.
func (s *LeadService) SaveLabel(ctx context.Context, id int64, name, color string) error {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" || utf8.RuneCountInString(name) > 60 {
		return errors.New("nama label wajib diisi, maksimal 60 karakter")
	}
	if color != "brand" && color != "secondary" && color != "neutral" {
		return errors.New("warna label tidak valid")
	}
	return s.withTx(ctx, func(q *queries.Querier) error {
		labels, err := q.ListLeadLabels(ctx)
		if err != nil {
			return err
		}
		found := id == 0
		for _, label := range labels {
			if label.ID == id {
				found = true
				if label.Nama != name {
					if err := ensureLabelEditable(ctx, q, label.Nama); err != nil {
						return err
					}
				}
			} else if strings.EqualFold(label.Nama, name) {
				return errors.New("nama label sudah digunakan")
			}
		}
		if !found {
			return errors.New("label tidak ditemukan")
		}
		if id == 0 {
			_, err = q.CreateCustomLeadLabel(ctx, queries.CreateCustomLeadLabelParams{Nama: name, Warna: color})
			return err
		}
		return q.UpdateCustomLeadLabel(ctx, queries.UpdateCustomLeadLabelParams{Nama: name, Warna: color, ID: id})
	})
}

func ensureLabelEditable(ctx context.Context, q *queries.Querier, name string) error {
	settings, err := q.GetCRMSettings(ctx)
	if err != nil {
		return err
	}
	if strings.EqualFold(name, pilotFollowupLabel) || (settings.FollowupOnlyLabel != "" && strings.EqualFold(name, settings.FollowupOnlyLabel)) {
		return errors.New("label dipakai aturan follow-up; ubah aturan terlebih dahulu sebelum mengganti nama atau menghapus label")
	}
	return nil
}

func (s *LeadService) DeleteLabel(ctx context.Context, id int64) error {
	return s.withTx(ctx, func(q *queries.Querier) error {
		label, err := q.GetLeadLabelByID(ctx, id)
		if err != nil {
			return errors.New("label tidak ditemukan")
		}
		if err := ensureLabelEditable(ctx, q, label.Nama); err != nil {
			return err
		}
		return q.DeleteCustomLeadLabel(ctx, id)
	})
}
