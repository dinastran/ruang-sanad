# Class Planning P0: additive foundation

Branch: `feature/class-planning-p0-20261031`. Scope is **P0 only**; no new UI, endpoints, auto-assign, publication, migration of existing classes, or runtime changes in this patch.

## Existing flow impact audit

* `SantriService.Create`, update, and admin-class update call `KelasEngineService.ProcessSantri` / `ProcessSantriDenganGuru`. Those paths can create a class and immediately write `santri.kelas_id` before class-planning approvals exist.
* `KelasEngineService.PindahkanSantri` moves an existing roster member, may synchronize billing, and replaces `santri.level` with `kelas.level`. **Do not reuse this unchanged for a two-level class.**
* `OnboardingService.Move` currently requires `santri.kelas_id` to enter `waiting_kuota`. New plan candidates must remain separate from this legacy state until activation.
* `JadwalPenawaranService` computes seats using verified leads from the current offer period, while `kelas` occupancy counts active and cuti santri. Do not sum these measures or reset offer periods on planning actions.
* Existing class capacity defaults to 15. The new regular-class 8–10 rule applies to new plans; existing class capacities are not changed by this patch.

## Design guardrails

1. The additive schema stores **unpublished draft plans** and confirmed teacher availability separately from operational `kelas` and `kelas_jadwal_rutin`.
2. `ValidatePlanningActivation` is deterministic and side-effect-free. It enforces gender, 90-minute slots, two distinct days for 2x/pekan, confirmed availability, conflicts, approved teaching levels, minimum quota, capacity, eligibility, and agreed schedule. A result from an earlier preview is not a permission to activate.
3. `PlanningCandidate.PlacementEligible`, `ScheduleApproved`, and `PlanningTeacher.ApprovedLevels` must be derived from authoritative records, never assumed from `santri.status` or teacher certificates alone.
4. Workload limits are nullable pending a coordinator policy. `guru_availability.confirmed_at` is mandatory before an availability window is usable, even though the database permits unconfirmed imported rows.
5. Class-plan approval requires coordinator confirmation for the exact plan version and subsequent Admin Kelas activation. Quota exceptions below 8 require Super Admin approval, matching existing onboarding exception authority.
6. Activation must reread live roster, teacher bookings, approvals, and verified demand inside a single database transaction. `class_plan_events` must remain append-only in application logic.
7. For multi-level classes, preserve `santri.level`; use `kelas_allowed_levels` as an additional relation. Do not assign a fabricated merged level value to `kelas.level`.

## Next vertical slice (P1)

Implement read-only recommendation service / query (existing-class-first), persisted draft and role-controlled coordinator approval, then a separate activation transaction with regression tests. Add a guarded opt-in to the old placement entry points before enabling new-class activation. No automatic migration of current teachers' free-text availability; import requires normalization review and coordinator confirmation.

## Verification

P0 rule unit tests run independently with Go standard library. SQLite schema can be checked using in-memory DDL. Whole-app Go tests, production migration, and runtime behavior must be verified separately before deploy.
