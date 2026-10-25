package services

import (
	"context"
	"encoding/json"
	"time"
)

type leadPauseState struct {
	paused bool
	until  string
}

// Resolve independently of the capped timeline, using event occurrence order.
// Expiration does not depend on the bridge delivering a separate resume event.
func (s *LeadService) leadPauseStates(ctx context.Context, ids []int64) (map[int64]leadPauseState, error) {
	states := make(map[int64]leadPauseState)
	if len(ids) == 0 {
		return states, nil
	}
	events, err := s.querier.ListLeadPauseEvents(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		if event.Type != "ai.paused" {
			continue
		}
		var payload struct {
			Until string `json:"until"`
		}
		if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
			return nil, err
		}
		if payload.Until != "" {
			until, err := time.Parse(time.RFC3339, payload.Until)
			// Malformed legacy deadlines cannot safely establish that a pause expired.
			if err == nil && !until.After(s.nowUTC()) {
				continue
			}
		}
		states[event.LeadID] = leadPauseState{paused: true, until: payload.Until}
	}
	return states, nil
}
