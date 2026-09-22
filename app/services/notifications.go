package services

import (
	"context"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

type NotificationService struct {
	querier *queries.Querier
}

func NewNotificationService(querier *queries.Querier) *NotificationService {
	return &NotificationService{querier: querier}
}

func (s *NotificationService) ListForUser(userID int64) ([]models.NotificationResponse, error) {
	rows, err := s.querier.ListNotificationsForUser(context.Background(), userID)
	if err != nil {
		return nil, err
	}
	out := make([]models.NotificationResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.NotificationResponse{
			ID:        row.ID,
			Type:      row.Type,
			Title:     row.Title,
			Message:   row.Message,
			ActionURL: row.ActionUrl,
			Read:      row.ReadAt.Valid,
			CreatedAt: row.CreatedAt.In(wib).Format("2006-01-02 15:04"),
		})
	}
	return out, nil
}

func (s *NotificationService) MarkRead(userID, notificationID int64) error {
	_, err := s.querier.MarkNotificationRead(context.Background(), queries.MarkNotificationReadParams{
		ID:     notificationID,
		UserID: userID,
	})
	return err
}
