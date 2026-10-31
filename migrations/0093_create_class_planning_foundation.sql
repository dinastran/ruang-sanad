-- +goose Up
-- +goose StatementBegin
-- Additive foundation only. This migration does not change legacy class
-- assignment, existing class capacities, or published schedule quotas.
CREATE TABLE guru_availability (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guru_id INTEGER NOT NULL REFERENCES guru(id) ON DELETE CASCADE,
    hari INTEGER NOT NULL CHECK (hari BETWEEN 1 AND 7),
    jam_mulai TEXT NOT NULL,
    jam_selesai TEXT NOT NULL,
    berlaku_mulai DATE NOT NULL,
    berlaku_sampai DATE,
    confirmed_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    confirmed_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (length(jam_mulai) = 5 AND length(jam_selesai) = 5 AND jam_mulai < jam_selesai),
    CHECK (berlaku_sampai IS NULL OR berlaku_sampai >= berlaku_mulai),
    UNIQUE(guru_id, hari, jam_mulai, jam_selesai, berlaku_mulai)
);
CREATE INDEX idx_guru_availability_lookup ON guru_availability(guru_id, hari, berlaku_mulai);

CREATE TABLE guru_workload_policy (
    guru_id INTEGER PRIMARY KEY REFERENCES guru(id) ON DELETE CASCADE,
    target_weekly_minutes INTEGER CHECK (target_weekly_minutes IS NULL OR target_weekly_minutes >= 0),
    max_weekly_minutes INTEGER CHECK (max_weekly_minutes IS NULL OR max_weekly_minutes >= 0),
    updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (target_weekly_minutes IS NULL OR max_weekly_minutes IS NULL OR target_weekly_minutes <= max_weekly_minutes)
);

CREATE TABLE guru_teaching_levels (
    guru_id INTEGER NOT NULL REFERENCES guru(id) ON DELETE CASCADE,
    level_kode TEXT NOT NULL,
    approved_by INTEGER NOT NULL REFERENCES users(id),
    approved_at DATETIME NOT NULL,
    PRIMARY KEY(guru_id, level_kode)
);

CREATE TABLE class_plans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source TEXT NOT NULL CHECK (source IN ('auto', 'manual')),
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN (
        'draft','pending_coordinator','pending_admin','pending_exception',
        'blocked','activated','cancelled','expired'
    )),
    tipe TEXT NOT NULL,
    angkatan TEXT NOT NULL,
    gender TEXT NOT NULL CHECK (gender IN ('L','P')),
    guru_id INTEGER REFERENCES guru(id) ON DELETE SET NULL,
    frequency INTEGER NOT NULL CHECK (frequency IN (1,2)),
    min_quota INTEGER NOT NULL CHECK (min_quota > 0),
    capacity INTEGER NOT NULL CHECK (capacity > 0 AND capacity >= min_quota),
    target_kelas_id INTEGER REFERENCES kelas(id) ON DELETE SET NULL,
    activated_kelas_id INTEGER REFERENCES kelas(id) ON DELETE SET NULL,
    score REAL,
    ruleset_version TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    coordinator_confirmed_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    coordinator_confirmed_at DATETIME,
    coordinator_confirmed_version INTEGER,
    admin_approved_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    admin_approved_at DATETIME,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_class_plans_status ON class_plans(status, updated_at);
CREATE INDEX idx_class_plans_guru ON class_plans(guru_id, status);
CREATE UNIQUE INDEX uq_class_plans_new_class ON class_plans(activated_kelas_id)
WHERE activated_kelas_id IS NOT NULL AND target_kelas_id IS NULL;

CREATE TABLE class_plan_slots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_id INTEGER NOT NULL REFERENCES class_plans(id) ON DELETE CASCADE,
    hari INTEGER NOT NULL CHECK (hari BETWEEN 1 AND 7),
    jam_mulai TEXT NOT NULL,
    jam_selesai TEXT NOT NULL,
    CHECK (length(jam_mulai) = 5 AND length(jam_selesai) = 5 AND jam_mulai < jam_selesai),
    UNIQUE(plan_id, hari)
);

CREATE TABLE class_plan_levels (
    plan_id INTEGER NOT NULL REFERENCES class_plans(id) ON DELETE CASCADE,
    level_kode TEXT NOT NULL,
    PRIMARY KEY(plan_id, level_kode)
);

CREATE TABLE class_plan_students (
    plan_id INTEGER NOT NULL REFERENCES class_plans(id) ON DELETE CASCADE,
    santri_id INTEGER NOT NULL REFERENCES santri(id) ON DELETE CASCADE,
    level_kode TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'candidate' CHECK (status IN ('candidate','reserved','committed','released')),
    schedule_approved INTEGER NOT NULL DEFAULT 0 CHECK (schedule_approved IN (0,1)),
    added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(plan_id, santri_id)
);
-- A santri can appear in multiple previews, but only one live reservation.
CREATE UNIQUE INDEX uq_class_plan_student_reservation ON class_plan_students(santri_id)
WHERE status = 'reserved';

CREATE TABLE class_plan_exceptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_id INTEGER NOT NULL REFERENCES class_plans(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'requested' CHECK (status IN ('requested','approved','rejected','revoked')),
    reason TEXT NOT NULL,
    requested_by INTEGER NOT NULL REFERENCES users(id),
    requested_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    decided_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    decided_at DATETIME,
    decision_note TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_class_plan_exceptions_plan ON class_plan_exceptions(plan_id, requested_at DESC);

CREATE TABLE class_plan_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_id INTEGER NOT NULL REFERENCES class_plans(id),
    actor_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    from_status TEXT NOT NULL DEFAULT '',
    to_status TEXT NOT NULL DEFAULT '',
    reason TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_class_plan_events_plan ON class_plan_events(plan_id, created_at DESC);

CREATE TABLE kelas_allowed_levels (
    kelas_id INTEGER NOT NULL REFERENCES kelas(id) ON DELETE CASCADE,
    level_kode TEXT NOT NULL,
    PRIMARY KEY(kelas_id, level_kode)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS kelas_allowed_levels;
DROP TABLE IF EXISTS class_plan_events;
DROP TABLE IF EXISTS class_plan_exceptions;
DROP INDEX IF EXISTS uq_class_plan_student_reservation;
DROP TABLE IF EXISTS class_plan_students;
DROP TABLE IF EXISTS class_plan_levels;
DROP TABLE IF EXISTS class_plan_slots;
DROP INDEX IF EXISTS uq_class_plans_new_class;
DROP TABLE IF EXISTS class_plans;
DROP TABLE IF EXISTS guru_teaching_levels;
DROP TABLE IF EXISTS guru_workload_policy;
DROP TABLE IF EXISTS guru_availability;
-- +goose StatementEnd
