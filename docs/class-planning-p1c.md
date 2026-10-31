# P1c: Teaching eligibility and alternative schedule consent

## Branch boundary

Base: `feature/class-planning-readiness-20261031` (PR #76, which also includes merged PR #77). Do **not** merge this branch to main independently of its base. No production mutation or deployment.

## Teaching level approvals

- The existing eight-field `guru_kompetensi` proficiency matrix is **not** equivalent to permission to teach a class level.
- `guru_teaching_levels` from migration 0093 is the source of explicitly approved level codes.
- Existing `level` master is used as the allowed set. A coordinator can replace the approved set in the existing guru detail page; empty set revokes all permissions.
- Only `koordinator_guru` and `super_admin` can submit a change (existing role middleware).
- Change uses transaction and avoids rewriting unchanged approvals. Every newly approved row has an approver and timestamp.
- Existing class teacher assignments and student levels remain unchanged.

## Alternative schedule preferences

- New additive migration 0094 stores a proposal, an approval evidenced by a recorded contact channel and admin note, or revocation. Every change produces an append-only event.
- Admin Kelas or Super Admin updates alternatives from the existing Santri detail page. CS does not obtain this write route.
- Selecting an alternative is limited to offered schedules compatible with santri gender and level, within an active onboarding case. Existing CRM primary choice cannot be duplicated as an alternative.
- This is a **recorded human attestation**, not independent evidence from the student's phone or WhatsApp account. Before any automated reassignment, the service must recheck current approval, active offer and source notes.
- Revoking an existing alternative remains possible even when the offer is no longer open.
- Historical `leads.jadwal_kode`, `santri.jadwal`, class membership, teacher occupancy, payment status and onboarding stage are not mutated.

## Next integration gate

The read-only matching engine currently only consumes confirmed **primary** CRM schedules. Do not auto-consume alternatives or alter assignment yet. First implement per-candidate preference evaluation, payment eligibility for non-CRM registrants, guard legacy `ProcessSantri` auto-placement, and final administrator/coordinator approval transaction.

## Verification

Targeted tests cover valid and invalid level sets, proposal/approval/revocation validation, actual query read/write and audit records, and source eligibility. Run CI build/tests plus a staging role-based smoke test before merge and production migrations.
