# P2c stage 2: Atomic class activation

**Base:** `feature/class-planning-p1c-20261031` after reservation PR #84. The integration PR #85 to `main` was still open/draft at development start. No production deployment in scope.

## Activation contract

Existing `POST /app/onboarding/planning/plans/:id/actions` accepts `{ "action":"activate", "version":1 }` only for Admin Kelas / Super Admin. Current role is rechecked in service. After approval, candidates must have an **explicit successful reservation**.

Preconditions checked again inside the same SQL transaction:

- Plan remains `pending_admin`, new-class `target_kelas_id IS NULL`, regular, exact immutable version.
- Coordinator confirmed the same version and a separate Admin Kelas approved. Actor activating must not be the confirmer.
- All candidates must remain `reserved`, in the current unassigned active onboarding roster, complete, with correct gender, cohort, academic level, and billing frequency.
- Verified CRM payment, the currently open primary offered schedule or a separately approved and evidenced alternative, approved teacher level, confirmed effective availability, teacher workload, existing active routines and competing pending plans still pass.
- Minimum quota 8 (unless valid latest Super Admin exception); capacity 8..10. 90-minute sessions with distinct weekdays, times may differ only where santri has exact evidenced consent.

All following writes are atomic:

1. Create one new regular `kelas` record using a plan-specific immutable key.
2. Create `kelas_allowed_levels` and dated `kelas_jadwal_rutin` rows.
3. Mark reserved plan students as `committed` (allows database trigger 0099 to permit this plan's class assignment).
4. Assign each santri with a conditional update; record the operational class schedule without overwriting the student's individual academic level.
5. Update class roster count, plan status to `activated` and `activated_kelas_id`, append audit event.
6. Commit only if every row update succeeds. Any failure rolls back the whole class, roster, routines, plan and events.

A repeated `activate` on the same committed version returns success for the existing class, without a duplicate class or audit record. UI shows the created class ID and does **not** mark onboarding `masuk_kelas`; that stage still requires actual start evidence.

## Legacy compatibility

Ordinary legacy `ProcessSantri` no longer silently reassigns a student already committed to an activated planning class. Unrelated profile edits remain allowed. A change of class placement/level/gender/schedule/frequency needs an explicit manual class-transfer workflow. Other santri and manually initiated `PindahkanSantri` are not globally disabled.

## Out of scope / deployment risks

- No automatic CS schedule publication, onboarding transition to `masuk_kelas`, automatic teacher notification, existing-class fill activation, or multi-level shared class in this PR.
- Historical/non-CRM payment eligibility is not inferred.
- Staging must test the actual role/CSRF controls, SQL migration 0098 + 0099 sequencing and real class UI before production deployment.
- Merging code does not apply production migrations or prove production health.

## Verification

Full-Goose-migration service integration tests cover successful activation, immediate retry idempotence, fresh eligibility checks, cross-role denial, legacy edit protection, and rollback when a routine INSERT is interrupted. Frontend/backend build and Go source formatting are CI gates.