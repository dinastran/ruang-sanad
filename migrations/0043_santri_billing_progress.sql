-- +goose Up
-- +goose StatementBegin

-- Billing progress is owned by the santri, not by the class. The counter stores
-- meetings accumulated toward the next SPP period; month 1 is prepaid before
-- class starts, so automated billing starts from month 2.
CREATE TABLE IF NOT EXISTS santri_billing_progress (
    santri_id INTEGER PRIMARY KEY REFERENCES santri(id) ON DELETE CASCADE,
    meeting_count INTEGER NOT NULL DEFAULT 0 CHECK (meeting_count >= 0),
    last_billed_month INTEGER NOT NULL DEFAULT 1 CHECK (last_billed_month >= 1),
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- One attendance-bearing meeting may advance a santri's billing counter only
-- once. This ledger makes normal completion, retries, and Finance Sync idempotent.
CREATE TABLE IF NOT EXISTS santri_billing_meeting (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    santri_id INTEGER NOT NULL REFERENCES santri(id) ON DELETE CASCADE,
    pertemuan_id INTEGER NOT NULL REFERENCES pertemuan(id) ON DELETE CASCADE,
    frekuensi TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (santri_id, pertemuan_id)
);

CREATE INDEX IF NOT EXISTS idx_santri_billing_meeting_santri
ON santri_billing_meeting(santri_id, pertemuan_id);

-- Existing attendance is intentionally not marked as processed here. The
-- service replays historical completed meetings in chronological order. Existing
-- tagihan bulan_ke rows act as anchors, so replay reconstructs the counter
-- without duplicating invoices and can create only genuinely missing periods.

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_santri_billing_meeting_santri;
DROP TABLE IF EXISTS santri_billing_meeting;
DROP TABLE IF EXISTS santri_billing_progress;
-- +goose StatementEnd
