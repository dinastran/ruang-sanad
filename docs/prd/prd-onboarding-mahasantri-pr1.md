# Onboarding Mahasantri — PR 1 Foundation

Scope: additive schema, atomic case opening on santri creation (CS manual and CRM lead conversion through shared SantriService.CreateWithQuerier), initial audit event, and bounded read-only endpoint.

Locked rules:
- One case per santri enforced by UNIQUE(santri_id).
- Day 0 is actual creation timestamp, not historical tanggal_daftar.
- Deadline is exclusive 00:00 WIB on Day 15; Day 14 is still on time.
- Existing santri never receive a backfilled case; historical CSV import through ImportService is explicitly exempted, including imports after this PR ships.
- No edits to existing santri.status, kelas_id, billing, class engine, or CRM stage.
- Linked-existing-santri conversion must not open another case.
- New cases start in onboarding_baru with empty PIC until PR 2.
- Class controls prepared, not populated until PR 3.
- Read-only GET /app/onboarding/cases restricted to admin_kelas/super_admin and newest 100 records.
- No WhatsApp sends, stage transitions, or automatic quota processing in PR 1.

Before merge: run npm run db:generate (sqlc v1.31.1), review generated output, gofmt, targeted Go tests and regression. Database migration and VPS deployment need separate approval. Do not backfill existing santri.
