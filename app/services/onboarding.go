package services

import (
 "context"
 "database/sql"
 "time"

 "github.com/maulanashalihin/laju-go/app/models"
 "github.com/maulanashalihin/laju-go/app/queries"
)

const OnboardingStageBaru = "onboarding_baru"
var onboardingWIB = time.FixedZone("WIB", 7*60*60)

// OnboardingDeadline is the exclusive midnight boundary after Day 14.
// Registration calendar date in WIB is Day 0.
func OnboardingDeadline(registeredAt time.Time) time.Time {
 local := registeredAt.In(onboardingWIB)
 midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, onboardingWIB)
 return midnight.AddDate(0, 0, 15).UTC()
}

// OpenInitialOnboarding runs within the santri creation transaction.
// Failed insert/activity recording aborts both santri and onboarding creation.
func OpenInitialOnboarding(ctx context.Context, q *queries.Querier, santriID, actorID int64, registeredAt time.Time) error {
 registeredAt = registeredAt.UTC()
 result, err := q.CreateOnboardingCase(ctx, queries.CreateOnboardingCaseParams{
  SantriID: santriID, RegisteredAt: registeredAt,
  DeadlineAt: OnboardingDeadline(registeredAt),
  CreatedAt: registeredAt, UpdatedAt: registeredAt,
 })
 if err != nil { return err }
 caseID, err := result.LastInsertId()
 if err != nil { return err }
 return q.CreateOnboardingActivity(ctx, queries.CreateOnboardingActivityParams{
  CaseID: caseID,
  ActorUserID: sql.NullInt64{Int64: actorID, Valid: actorID > 0},
  EventType: "onboarding.created", ToStage: OnboardingStageBaru,
  CreatedAt: registeredAt,
 })
}

type OnboardingService struct { querier *queries.Querier }
func NewOnboardingService(q *queries.Querier) *OnboardingService { return &OnboardingService{querier:q} }

// The endpoint is intentionally capped until PR 2 adds cursor pagination.
func (s *OnboardingService) List(ctx context.Context) ([]models.OnboardingCaseListItem, error) {
 rows, err := s.querier.ListOnboardingCases(ctx, 100)
 if err != nil { return nil, err }
 out := make([]models.OnboardingCaseListItem, 0, len(rows))
 for _, r := range rows {
  out = append(out, models.OnboardingCaseListItem{
   ID:r.ID, SantriID:r.SantriID, Nama:r.Nama, IDMahasantri:r.IDMahasantri, Stage:r.Stage,
   RegisteredAt:r.RegisteredAt.In(onboardingWIB).Format(time.RFC3339),
   DeadlineAt:r.DeadlineAt.In(onboardingWIB).Format(time.RFC3339),
  })
 }
 return out, nil
}
