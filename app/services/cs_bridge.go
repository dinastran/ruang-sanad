package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// WhatsApp sends from the CS number go through the CS AI bridge (PRD D6):
// POST {CS_BRIDGE_URL}/api/outbound. The bridge sends via GOWA, keeps the
// message in the chat context for the CS AI, and emits message.out back to
// the Dashboard. Contract: docs/panduan/bridge-outbound-api.md.

// OutboundSender sends one message through the bridge.
type OutboundSender interface {
	SendOutbound(ctx context.Context, req OutboundRequest) (string, error)
}

type OutboundRequest struct {
	ChatID      string `json:"chat_id"`
	Text        string `json:"text"`
	ReferenceID int64  `json:"reference_id"`
	Purpose     string `json:"purpose"`
}

// OutboundError is a failed send. Permanent errors are not retried (bad
// request, auth); temporary ones (network, 5xx, chat paused) are.
type OutboundError struct {
	Msg       string
	Permanent bool
}

func (e *OutboundError) Error() string { return e.Msg }

var ErrBridgeNotConfigured = &OutboundError{Msg: "CS_BRIDGE_URL / CS_BRIDGE_OUTBOUND_TOKEN belum diisi di .env", Permanent: true}

type CSBridgeClient struct {
	url    string
	token  string
	client *http.Client
}

func NewCSBridgeClient(url, token string) *CSBridgeClient {
	return &CSBridgeClient{url: strings.TrimRight(url, "/"), token: token, client: &http.Client{Timeout: 20 * time.Second}}
}

func (c *CSBridgeClient) SendOutbound(ctx context.Context, req OutboundRequest) (string, error) {
	if c == nil || c.url == "" || c.token == "" {
		return "", ErrBridgeNotConfigured
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/api/outbound", bytes.NewReader(body))
	if err != nil {
		return "", &OutboundError{Msg: "URL bridge tidak valid", Permanent: true}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", &OutboundError{Msg: "bridge tidak bisa dihubungi"}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	var out struct {
		WaMessageID string `json:"wa_message_id"`
		Error       string `json:"error"`
	}
	_ = json.Unmarshal(raw, &out)
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		if out.WaMessageID == "" {
			return "", &OutboundError{Msg: "bridge tidak mengembalikan wa_message_id"}
		}
		return out.WaMessageID, nil
	case resp.StatusCode == http.StatusConflict:
		return "", &OutboundError{Msg: "chat sedang dijeda (admin menangani langsung)"}
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		return "", &OutboundError{Msg: fmt.Sprintf("bridge membalas HTTP %d %s", resp.StatusCode, out.Error)}
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return "", &OutboundError{Msg: "bridge menolak token (cek CS_BRIDGE_OUTBOUND_TOKEN)", Permanent: true}
	default:
		return "", &OutboundError{Msg: strings.TrimSpace(fmt.Sprintf("bridge menolak pesan: HTTP %d %s", resp.StatusCode, out.Error)), Permanent: true}
	}
}

func outboundErrorInfo(err error) (string, bool) {
	var oe *OutboundError
	if errors.As(err, &oe) {
		return oe.Msg, oe.Permanent
	}
	return "gagal mengirim: " + err.Error(), false
}
