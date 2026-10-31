# P1d: Approved alternative schedules in read-only class matching

Base: P1c `feature/class-planning-p1c-20261031` (PR #78). Main is not merged by this change.

## Read path

- The original P1b recommendation still uses verified CRM payment and the student's primary offered schedule.
- P1d reads **only approved** entries in `santri_schedule_alternatives` with confirmation timestamp, channel, and non-empty evidence note.
- Alternative offers must still be open, compatible with current santri gender/level, and must not be the current primary CRM choice.
- A student must be active, complete, in a matching/waiting onboarding stage, unassigned to a class, and not reserved or committed to a planning draft.
- The new query is intersected with the original verified primary-demand snapshot in memory. Missing or ambiguous original demand is not silently recovered by an alternative. A schedule must still parse as exactly one or two 90-minute lessons per week.

## Matching policy

1. Primary schedule recommendations have precedence: a santri in a `ready_preview` primary class is skipped from alternate-class matching.
2. Alternative schedules are currently compared to **existing regular classes without recorded meetings**, with exact cohort, gender, level, frequency, days and times. Active teacher gender rule still applies.
3. A santri can appear in at most one alternative match. If moved from a primary proposal that could not start, the original speculative option removes that candidate and recalculates its minimum-quota state.
4. Class capacity is shared across all primary offer-code groups and approved alternatives. A legacy class capacity above 10 is not mutated; at most 10 total seats are recommended for new intake.
5. Alternative matches are displayed in a **separate read-only section**, not added to confirmed demand or converted into enrollment. This is not an automatic schedule change.
6. Results are bounded; truncation is signaled. Confirmed approvals and schedules must be re-read at final placement time.

## Legacy activation gate

Existing `KelasEngineService.ProcessSantri` and `ProcessSantriDenganGuru` can directly create classes and set `kelas_id`. P1d **does not change this operational route** because blindly blocking it would alter live CS/import/onboarding behavior. Before enabling planning activation:

- Identify all entrypoints: create/import/update santri, update admin-class data, manual pindah.
- Introduce an approved feature-gated *planning-only* write path. Both coordinator confirmation for the exact version and Admin Kelas activation must be checked transactionally.
- Protect a candidate against concurrent legacy placement and recheck consent and class/teacher conflict inside one transaction.
- Confirm compatibility of any new legacy-path guard under staging tests; obtain explicit approval before changing operational placement semantics.
- Do not treat this PR's read-only matches as authorization to place santri.

## Verification

Tests cover primary precedence, 90-minute shape, alternative consent/payment/reservation filters, deterministic alternative selection, no duplicate candidate identities, and per-class seat capacity across multiple offer codes. CI/production status must be evaluated separately.
