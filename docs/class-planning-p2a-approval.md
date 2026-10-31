# P2a: Manual draft and two-role approval (no activation)

Base: `feature/class-planning-p1c-20261031`, which now contains merged P1d PR #80. PR #76 and PR #78 have not yet merged to `main` as of branch creation.

## Scope

Add API workflow over existing `class_plans` tables from migration 0093. **This does not modify operational `kelas`, `santri.kelas_id`, existing `KelasEngine`, live rosters or payment. There is no activation endpoint.**

### API (session-authenticated and role-protected)

- `GET /app/onboarding/planning/plans`: Admin Kelas, Koordinator Guru and Super Admin. Returns open workflow items (max 100).
- `POST /app/onboarding/planning/plans`: Admin Kelas and Super Admin. Creates a `manual` draft, including slots, level and selected candidates. The client must send `angkatan`, `gender`, `level`, `guru_id`, `frequency`, `capacity`, `slots:[{Day,Start,End}]`, `candidate_ids` and optionally `target_kelas_id`. Regular capacity 8..10, every meeting 90 minutes, 1 or 2 distinct days.
- `POST /app/onboarding/planning/plans/:id/actions`: Admin Kelas and Super Admin can send `submit` or `admin_approve`; Koordinator Guru and Super Admin can send `coordinator_confirm`. Body: `{action,version,reason}`. All actions are checked server-side against the authenticated user's current persisted role, not just route middleware.

### Transitions

1. `draft -> pending_coordinator`: Admin submits. Transaction rechecks current candidate eligibility, verified CRM payment, exact approved primary or alternative offered schedule, teacher approved level, current confirmed availability, teacher conflicts and room/capacity.
2. `pending_coordinator -> pending_admin`: Coordinator confirms the same immutable version. The specific confirmer and confirmation version are persisted.
3. `pending_admin -> pending_admin` with `admin_approved_by` and timestamp: Admin authorizes a *future* activation, not a class change. Approver must differ from coordinator confirmer. The final placement phase will re-read all evidence.
4. Under-quota with otherwise valid evidence: `draft -> pending_exception`. The exception approval/rejection path is deliberately **not available** in P2a. No bypass to coordinator confirmation.
5. Any conflict, stale version, duplicate action or missing evidence rejects the transition without persisting a partial event. Each committed transition appends an audit event.

## Safety and limitations

- Existing `ProcessSantri`, manual pindah and onboarding operations remain unchanged. The parallel planning write workflow is isolated to its own tables.
- Drafts can be incomplete or become stale: creation does not reserve students, classes, or teachers. Submission/approval *rechecks* source evidence. Draft candidates in another active planning workflow are rejected when submitting.
- `version` is an immutable draft version in this slice; no edit route exists. Any future edit path must bump version and clear all approvals.
- No exception decision, manual payment eligibility, different times per day in original offer, direct automatic recommendation promotion, optimizer, or actual class activation in this PR.
- Approval by itself must **never** be interpreted as a booked class. Atomic activation needs separate explicit approval and a transaction with fresh conflict/consent/roster verification.

## Next gate

Implement a review interface for Admin Kelas / Koordinator Guru, exception handling, a compatibility-safe legacy placement guard, and a separately approved transactional activation. Staging smoke tests and migration/deploy gate remain required.
