-- +goose Up
-- +goose StatementBegin
-- P1c: Record alternative offers with explicit student consent.
-- Historical lead choice and original santri.jadwal stay unchanged.
CREATE TABLE santri_schedule_alternatives (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    santri_id INTEGER NOT NULL REFERENCES santri(id) ON DELETE CASCADE,
    jadwal_kode TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('proposed','approved','revoked')),
    consent_channel TEXT NOT NULL DEFAULT '' CHECK(consent_channel IN ('','whatsapp','phone','other')),
    consent_note TEXT NOT NULL DEFAULT '',
    consent_confirmed_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    consent_confirmed_at DATETIME,
    updated_by INTEGER NOT NULL REFERENCES users(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(santri_id,jadwal_kode),
    CHECK (status <> 'approved' OR (
        consent_channel <> '' AND length(trim(consent_note)) > 0
        AND consent_confirmed_by IS NOT NULL AND consent_confirmed_at IS NOT NULL
    ))
);
CREATE INDEX idx_santri_schedule_alternatives_status
ON santri_schedule_alternatives(status,jadwal_kode);

CREATE TABLE santri_schedule_alternative_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    alternative_id INTEGER NOT NULL REFERENCES santri_schedule_alternatives(id) ON DELETE CASCADE,
    santri_id INTEGER NOT NULL REFERENCES santri(id) ON DELETE CASCADE,
    actor_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    old_status TEXT NOT NULL,
    new_status TEXT NOT NULL,
    consent_channel TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_santri_schedule_alternative_events
ON santri_schedule_alternative_events(alternative_id,created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS santri_schedule_alternative_events;
DROP TABLE IF EXISTS santri_schedule_alternatives;
-- +goose StatementEnd
