# P2b: Quota exception approval and legacy placement boundary

Base: `feature/class-planning-p1c-20261031`. PR #81 is merged into this feature branch, not into `main`. No activation or deployment in this slice.

## Exception workflow

- Existing `class_plan_exceptions` and `class_plan_events` tables are reused, no new migration.
- Under-8 regular group: Admin Kelas submits `submit` with a specific reason of at least 10 characters; if and only if all other rules pass, the plan enters `pending_exception` and an exception request/audit event is inserted in the same transaction.
- Only Super Admin (server-side persisted role) may `exception_approve` or `exception_reject`; the requester cannot decide their own exception.
- Approving requires fresh preflight to report **only** `MIN_QUOTA_NOT_MET`. All teacher, gender, student, payment, schedule, and conflict checks still apply.
- Approval writes decision actor/timestamp/note and moves the plan to `pending_coordinator` in one transaction. Coordinator must still confirm the unchanged plan version, and an Admin Kelas different from the coordinator still must approve.
- Rejection records the decision and moves the plan to `blocked`. The plan cannot proceed to coordinator review. A new draft is required if circumstances change.
- Duplicate and stale-version decisions are rejected by SQL compare-and-swap conditions; an exception only counts when the latest record is approved, decided by someone other than requester, and has a non-empty note.
- Super Admin controls are displayed in the existing Onboarding > Kelas & Kuota approval inbox. The request reason and decision status are visible to reviewers; no new sidebar item.

## Legacy placement audit (no code mutation)

Targeted repository inspection found that operational placement continues to use:

- `app/services/santri.go`: `ProcessSantri` in creation and update flows; `ProcessSantriDenganGuru` in Admin Kelas data updates; `PindahkanSantri` in explicit class moves.
- `app/services/kelas_engine.go`: `ProcessSantriDenganGuru` may allocate/create a class; `PindahkanSantri` mutates class membership.
- `app/services/kelas_perubahan.go`: another `PindahkanSantri` path and level-change class creation/assignment.
- `queries/kelas.sql`: `CreateKelas` and class mutations are still used by operational services.

Do not unconditionally block these flows: registrations, edits, historical imports, and migrations may rely on them. Before enabling planning activation, introduce a planning-scoped reservation/guard that detects `pending_admin` or approved plans, verify conflicts inside a SQLite write transaction, and test each legacy entry point for regressions.

## Operational activation remains blocked

This P2b change does not write to `kelas`, `santri.kelas_id`, `kelas_jadwal_rutin`, or auto-assignment logic. The future activation step requires a single transaction that checks fresh student eligibility, booked/other-plan conflicts, current teacher availability and teaching levels, class capacity, effective schedule consent, coordinator confirmation, Admin Kelas approval, and a valid exception if under eight.

## Verification gates

Targeted tests for request/decision compare-and-swap, reviewer distinctness, rejected-plan blocking, approved-plan continuation, and hard-constraint bypass; CI Go tests, build and formatting. Runtime role/CSRF smoke test and production verification are separate gates.
