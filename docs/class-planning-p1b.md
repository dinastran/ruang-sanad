# P1b: Read-only class matching preview

Base: PR #76 (`feature/class-planning-readiness-20261031`). No merge/deploy requested.

## Behavior and invariants

1. `GET /app/onboarding/planning/recommendations` is guarded by existing Admin Kelas/Super Admin middleware and emits a read-only JSON snapshot.
2. Demand includes only active, complete, unassigned students in `matching_kelas` or `waiting_kuota`, joined to CRM leads with verified payment, valid status (`open` or `won`), no review flag, and an existing offered schedule code. Reserved/committed plan candidates are excluded. This **does not imply unverified manual payments are unpaid**.
3. A schedule is treated as unambiguous only if offer days (1 or 2) and student frequency match, and the offer meeting duration is exactly 90 minutes. It does not infer consent to alternative days/times from free-text `santri.jadwal`.
4. Fill compatible existing regular classes before proposing new classes. Existing candidate class must be active with an assigned active guru, exact gender, cohort, level, frequency, current routine slots and **no recorded meetings**. For capacity calculations, cap any old 15-seat class at 10 for a new recommendation without modifying old data.
5. New class preview requires 8 confirmed candidates to be marked `ready_preview`, at most 10 per class. 16 to 18 candidates can split 8+8, 9+8 or 10+8.
6. New-class teacher choices require active guru, approved teaching level, current effective **confirmed** availability containing the full desired 90-minute intervals, compatible gender, no active-class routine or coordinator-confirmed-plan overlap, and no unknown schedule for any active class belonging to the teacher. Enforce max weekly minutes when a policy exists.
7. For akhwat, prefer approved akhwat teachers; permanent status is a secondary preference. Existing bookings are counted in weekly workload, and previewed new classes consume virtual slots so the same teacher is not proposed twice at the same time within one snapshot.
8. 2x/pekan offers from existing Jadwal Penawaran currently assume equal times on both days. **Different start times cannot be recommended to a confirmed demand group without recorded individual schedule consent**. This remains future scope.
9. Preview returns reasons and warnings. It **does not** reserve students, mutate roster, edit `kelas`, change `kelas_jadwal_rutin`, submit coordinator approval, or publish to CS. It must be revalidated against fresh source data before any future activation.

## Known limitations

- No historical demand prediction, retention score, teacher target-hour optimization, or multi-level class combination.
- No support for non-CRM/manual payment verification or schedule alternatives until data source and consent flow are implemented.
- Existing classes with any recorded meetings are excluded from automatic fill to avoid progression mismatches.
- Existing routine class duration is assumed 90 minutes based on current business rule; unknown/invalid busy slot blocks a teacher conservatively.
- Effective date is the current WIB date, not a future-week date search.
- Results are capped at 500 source candidates and 40 options. An incomplete snapshot carries `truncated=true`, so it must not be read as full allocation coverage.
- Readiness counts and detailed matching coverage are different; zero recommendations is valid if input data is not verified.

## Integration tests

- Existing-class-first with no duplicate candidate IDs.
- Under-quota never marked ready.
- Akhwat teacher preference, ikhwan teacher constraint.
- Busy interval and unknown active-class schedule block teacher.
- Split 16 confirmed candidates into two viable groups and don't double-book teacher.
- Active workload max policy.
- DB query excludes missing payment, reserved students, assigned students, and gender-mismatched offers.
- DB busy query includes active routines and pending-admin plans, but excludes drafts.

## Next gate

Before write paths/approval, add normalized alternative schedule consent, non-CRM payment eligibility, teacher level approval management and guarded legacy auto-placement. Full transactional activation is intentionally out of scope.
