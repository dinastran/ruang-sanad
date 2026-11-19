package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PaymentGateway is a hosted-checkout payment provider (Xendit today).
type PaymentGateway interface {
	CreateSession(ctx context.Context, req GatewaySessionRequest) (GatewaySession, error)
	GetSession(ctx context.Context, id string) (GatewaySession, error)
}

type GatewaySessionRequest struct {
	ReferenceID   string
	Amount        int64
	ExpiresAt     time.Time
	Description   string
	CustomerRef   string
	CustomerName  string
	CustomerPhone string // 62…
	ReturnURL     string
}

type GatewaySession struct {
	ID          string
	ReferenceID string
	Status      string // ACTIVE | COMPLETED | EXPIRED | CANCELED
	Amount      int64
	URL         string
	ExpiresAt   time.Time
	PaymentID   string
}

// XenditGateway talks to the Xendit Payment Session API (POST /sessions,
// mode PAYMENT_LINK). Auth: HTTP basic with the secret key as username.
type XenditGateway struct {
	baseURL   string
	secretKey string
	client    *http.Client
}

func NewXenditGateway(baseURL, secretKey string) *XenditGateway {
	if baseURL == "" {
		baseURL = "https://api.xendit.co"
	}
	return &XenditGateway{baseURL: strings.TrimRight(baseURL, "/"), secretKey: secretKey, client: &http.Client{Timeout: 20 * time.Second}}
}

type xenditSession struct {
	PaymentSessionID string    `json:"payment_session_id"`
	ReferenceID      string    `json:"reference_id"`
	Status           string    `json:"status"`
	Amount           float64   `json:"amount"`
	PaymentLinkURL   string    `json:"payment_link_url"`
	ExpiresAt        time.Time `json:"expires_at"`
	PaymentID        string    `json:"payment_id"`
}

func (x xenditSession) toSession() GatewaySession {
	return GatewaySession{
		ID: x.PaymentSessionID, ReferenceID: x.ReferenceID, Status: x.Status, Amount: int64(x.Amount),
		URL: x.PaymentLinkURL, ExpiresAt: x.ExpiresAt, PaymentID: x.PaymentID,
	}
}

func (g *XenditGateway) do(ctx context.Context, method, path string, body any) (xenditSession, error) {
	var out xenditSession
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return out, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, g.baseURL+path, rdr)
	if err != nil {
		return out, err
	}
	req.SetBasicAuth(g.secretKey, "")
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return out, fmt.Errorf("xendit: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var e struct {
			ErrorCode string `json:"error_code"`
			Message   string `json:"message"`
		}
		_ = json.Unmarshal(raw, &e)
		return out, fmt.Errorf("xendit %d %s: %s", resp.StatusCode, e.ErrorCode, e.Message)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("xendit: respons tidak terbaca: %w", err)
	}
	return out, nil
}

func (g *XenditGateway) CreateSession(ctx context.Context, r GatewaySessionRequest) (GatewaySession, error) {
	body := map[string]any{
		"reference_id": r.ReferenceID,
		"session_type": "PAY",
		"mode":         "PAYMENT_LINK",
		"amount":       r.Amount,
		"currency":     "IDR",
		"country":      "ID",
		"locale":       "id",
		"description":  r.Description,
		"expires_at":   r.ExpiresAt.UTC().Format(time.RFC3339),
		"customer": map[string]any{
			"reference_id":      r.CustomerRef,
			"type":              "INDIVIDUAL",
			"mobile_number":     "+" + r.CustomerPhone,
			"individual_detail": map[string]any{"given_names": r.CustomerName},
		},
		"success_return_url": r.ReturnURL,
		"cancel_return_url":  r.ReturnURL,
	}
	x, err := g.do(ctx, http.MethodPost, "/sessions", body)
	return x.toSession(), err
}

func (g *XenditGateway) GetSession(ctx context.Context, id string) (GatewaySession, error) {
	x, err := g.do(ctx, http.MethodGet, "/sessions/"+url.PathEscape(id), nil)
	return x.toSession(), err
}
