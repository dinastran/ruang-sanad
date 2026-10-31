# P2c stage 1: Approved-candidate reservation and legacy guard

Base: `feature/class-planning-p1c-20261031` (contains merged P2b #82). The feature branch is **not equivalent to main**.

## Scope

- **New admin action** `reserve` on an approved `pending_admin` new-class plan. Only the authenticated Admin Kelas or Super Admin (not the coordinator confirmer) can reserve. Admin approval and coordinator confirmation of the exact immutable version are prerequisites.
- Re-evaluate all hard rules, candidate eligibility, CRM payment, consent, teacher availability/conflict, workload, and approved under-quota exception inside the reservation transaction.
- Update every candidate to `reserved` and its snapshot `schedule_approved=1`. The row count must equal the plan candidate count; otherwise the whole SQL transaction is rolled back. The existing partial unique index guarantees a student has at most one live reservation.
- SQLite migration 0099 adds guards **only for explicitly reserved students in pending-admin plans**. Their legacy `santri.kelas_id` changes and deletions are refused. Ordinary operations for all other students continue unchanged.
- **New admin action** `cancel`: an approved pending-admin plan can be cancelled with a recorded reason of at least 10 characters, atomically marking any candidate/reserved rows `released` and writing the audit event. The guard then permits legacy placement again.
- Existing approvals, audit, version and permission middleware are reused. Existing routes are unchanged.

## Critical boundary

**No class is created, no `santri.kelas_id` is changed, and no CS schedule is published.** A reservation is not an active class or an irrevocable hold. Approval and reservation are not deployment.

There is no activation action in this slice. At activation, do not blindly toggle the reservation status and then assign: use one transaction to re-check current teacher validity, schedule consent, same-plan reservation ownership, competing plans, cohort/gender, capacity and any approved exception. Only after all checks pass may the plan transition to `activated` and student rows become `committed`.

The migration guard is intentionally conditional on `pending_admin` and `reserved`. Cancelling a plan releases the guard, and future activation can transition reservations to `committed` *inside the same transaction* immediately before updating operational membership, with rollback on failure.

## Verification targets

- Actual Goose Up SQL guards legacy class change and deletion of reserved santri.
- An unreserved santri's class change remains possible.
- Plan cannot reserve without coordinator/admin approval or with stale version.
- Repeated reservation does not claim new seats; duplicate student across approved plans is blocked by the partial unique index.
- Transaction rollback prevents partial reservation if one student is already claimed.
- Cancelling the plan releases all reservations and permits normal operation.
- Backend unit/query tests, frontend build, and code generation/formatting in CI.

## Known risks / next gate

- The old placement engine still auto-assigns **unreserved** santri; this is intentional backward compatibility.
- Changes to source schedule consent and teacher availability after reservation must still block final activation. Reservation is not final authorization.
- Runtime smoke test with Admin Kelas, Koordinator Guru and Super Admin has not been performed.
- A permission-aware release/expiry policy and full activation need separate review and staging testing.
- Integration of parent feature branches to `main` and deployment are separate approvals.
