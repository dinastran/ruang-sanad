# PROJECT STATE

REPO: dinastran/ruang-sanad
BRANCH: feature/management-overview-dashboard
BASE_COMMIT: 45992bb3a616a2f14582812204a8d3621ced1e16
ENVIRONMENT: isolated local checkout; no production access or deployment

## OBJECTIVE
Replace /app with an actionable management overview using existing module data.

## COMPLETED / VERIFIED
- Snapshot cards, six-month registration and finance charts, current status composition, program occupancy, riayah, teacher/class issues, stock, and finance.
- Server checks current database role before rendering. super_admin receives management data; admin and keuangan receive finance only. Other roles retain existing redirects.
- Read-only queries. No migrations, schedule generation, dependency version changes, or external AI provider.
- Current snapshots are separate from selected-month activity. Missing source data is unavailable, never a successful zero.
- Partial-error, empty, permission, month/WIB boundary, and read-only database tests pass.

## IMPORTANT FILES
- app/handlers/dashboard.go: role verification and overview rendering.
- app/services/overview.go: aggregation and action priorities.
- app/queries/overview.go: handwritten read-only aggregates; generated query files unchanged.
- app/models/overview.go and frontend/src/lib/overview.ts: contract.
- frontend/src/pages/app/Dashboard.svelte and components/OverviewChart.svelte: UI.
- app/services/overview_test.go and app/handlers/dashboard_test.go: behavior and permission tests.
- cmd/laju-go/main.go: dependency injection.

## LOCKED DECISIONS / LIMITS
- Financial access follows existing kuReadRole.
- No invented health score, retention figure, net profit, or complaint count. No structured complaint source found in inspected application models/queries.
- No merge or deployment without separate user authorization.

## VERIFICATION
- Frontend production build: PASS.
- Go application build: PASS (Go 1.26.1).
- Targeted overview and related finance/analytics/coordinator service tests: PASS.
- Dashboard handler, stale-role protection, and existing guru redirect tests: PASS.
- go vet on services, handlers, queries and git diff --check: PASS.
- Svelte server-render fixture checks for management, finance, partial errors, empty data: PASS with mocked navigation/browser globals.
- Browser visual checks / hydrated interaction / production smoke: NOT VERIFIED. Local Chrome exits before rendering; no dev server started per AGENTS.md.

## NEXT ACTION
Review the pull request, then obtain explicit approval for merge; deployment is separate.
