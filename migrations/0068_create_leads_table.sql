-- +goose Up
-- +goose StatementBegin
-- Calon santri yang chat ke nomor CS (CRM lead → closing). Satu baris per chat WA.
CREATE TABLE IF NOT EXISTS leads (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    wa_chat_id TEXT NOT NULL UNIQUE,
    phone TEXT NOT NULL DEFAULT '',
    wa_name TEXT NOT NULL DEFAULT '',
    nama TEXT NOT NULL DEFAULT '',
    usia INTEGER,
    domisili TEXT NOT NULL DEFAULT '',
    jenis_kelamin TEXT NOT NULL DEFAULT '',
    paket_kode TEXT NOT NULL DEFAULT '',
    kelas_kode TEXT NOT NULL DEFAULT '',
    waktu_belajar TEXT NOT NULL DEFAULT '',
    riwayat_belajar TEXT NOT NULL DEFAULT '',
    belajar_dimana TEXT NOT NULL DEFAULT '',
    untuk_siapa TEXT NOT NULL DEFAULT '',
    stage TEXT NOT NULL DEFAULT 'baru',
    status TEXT NOT NULL DEFAULT 'open',
    lost_reason TEXT NOT NULL DEFAULT '',
    followup_permission TEXT NOT NULL DEFAULT 'unknown',
    source TEXT NOT NULL DEFAULT 'wa_cs',
    assigned_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    payment_amount INTEGER NOT NULL DEFAULT 0,
    payment_info_sent_at DATETIME,
    proof_received_at DATETIME,
    verified_at DATETIME,
    verified_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    converted_at DATETIME,
    converted_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    santri_id INTEGER UNIQUE REFERENCES santri(id) ON DELETE SET NULL,
    needs_review INTEGER NOT NULL DEFAULT 0,
    last_inbound_at DATETIME,
    last_outbound_at DATETIME,
    last_analyzed_message_id INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_leads_status_stage ON leads(status, stage, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_leads_assigned ON leads(assigned_user_id);
CREATE INDEX IF NOT EXISTS idx_leads_last_inbound ON leads(last_inbound_at);
CREATE INDEX IF NOT EXISTS idx_leads_phone ON leads(phone);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_leads_phone;
DROP INDEX IF EXISTS idx_leads_last_inbound;
DROP INDEX IF EXISTS idx_leads_assigned;
DROP INDEX IF EXISTS idx_leads_status_stage;
DROP TABLE IF EXISTS leads;
-- +goose StatementEnd
