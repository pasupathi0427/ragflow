# Custom Changelog

This file lists every change made in this fork that is not in upstream RAGFlow (`infiniflow/ragflow`).
Before merging or rebasing onto upstream, read this file to see which files carry custom changes and must be checked after the merge.

## How to add an entry

- Add new entries at the top of the **Changes** section.
- Write one entry per feature or fix, not one per commit.
- List every file the change touches, so a merge conflict can be traced back to its entry.
- Fill in **Upstream merge notes** whenever the change edits a file upstream also changes often (handlers, routers, entities, migrations, `web/src` pages, Docker files).

### Entry template

```markdown
### [CUST-NNN] Short title — YYYY-MM-DD

- **Type:** feature | fix | config | refactor | security
- **Requirement:** what the use case needs, in one or two sentences
- **Change:** what was changed and why
- **Files:**
  - `path/to/file.go` — what changed
- **Database / config impact:** migrations, new config keys, new env vars (or "none")
- **Tests:** tests added or updated, and the command that runs them
- **Upstream merge notes:** what to check or re-apply after merging upstream
- **Commits:** short hashes
```

## Upstream sync log

| Date | Upstream commit merged | Conflicts in custom files | Notes |
|------|------------------------|---------------------------|-------|
| 2026-11-18 | `d3f50ff9a` (fork baseline) | — | Fork baseline before any custom change |

## Changes

### [CUST-001] Access profiles: super admin controls which settings features users see — 2026-11-18

- **Type:** feature, security
- **Jira:** AI-146 (sub-task of AI-145 "RAG Platform (RAGFlow) customization")
- **Requirement:** A super admin decides which user-settings features (Model providers, Data sources, Chat channels, MCP, Team, API) normal users can see and use. Users are grouped into named access profiles; each user has one profile, and users without one get the built-in Default profile.
- **Change:**
  - New Go package `internal/accessprofile` holds the profiles, user assignments, the feature check (cached 30 s per API process), gin middleware `RequireFeature`, and the admin handlers.
  - When a feature is off, its API routes return 403 and the UI hides it: the sidebar entry is gone, and a direct URL shows a "disabled for your account" notice.
  - Items the user already created keep running: connector syncs, channel bots, and MCP steps in existing agents.
  - Exception: when API is off, the user's existing API keys are rejected everywhere API keys are accepted (`AuthMiddleware`, `BetaAuthMiddleware`, `ResolveMCPUser`). Embed (beta) tokens are not affected.
  - Admin console has a new **Access profiles** page (`/admin/access-profiles`), and the Users table has a new access-profile column.
  - On that page, a search box filters profiles by name or description. Profile cards start collapsed and show a one-line summary of switched-off features. Expand all / Collapse all buttons are next to the search, and a newly created profile opens expanded.
  - Named "access profiles", not "roles", to stay clear of the EE `/admin/roles`, `/admin/model-providers`, `/admin/settings` paths and the `internal/permission` policy.
- **Gated routes (when the feature is off):**

  | Feature | Blocked | Left open |
  |---|---|---|
  | Model providers | `/api/v1/providers/*`, `PATCH /api/v1/models`, `PATCH /api/v1/models/default`, `PATCH /api/v1/users/me/models` | `GET /models`, `GET /models/default`, model groups (EE-only) |
  | Data sources | `/api/v1/connectors/*` | OAuth callbacks (need a blocked start call) |
  | Chat channels | `/api/v1/chat-channels/*` | |
  | MCP | `/api/v1/mcp/servers/*` | `/api/v1/mcp` (RAGFlow's own MCP endpoint; API-key callers are still checked against API) |
  | Team | `/api/v1/tenants/:id/users` (list/add/remove members) | `GET /tenants`, accepting an invite |
  | API | `/api/v1/system/tokens/*`, `/api/v1/system/keys/*`, and existing API keys | embed (beta) tokens |

- **Files (new):**
  - `internal/accessprofile/{feature,service,handler,api_key}.go` and `accessprofile_test.go`: feature definitions, service + cache, middleware/handlers, API-key check, unit tests.
  - `internal/entity/access_profile.go`: `access_profile`, `user_access_profile` tables.
  - `internal/dao/access_profile.go`: DAO.
  - `internal/admin/access_profile_routes_test.go`: admin routes require the superuser token.
  - `web/src/constants/access-profile.ts`, `web/src/interfaces/{database,request}/access-profile.ts`
  - `web/src/services/access-profile-service.ts`, `web/src/hooks/use-access-profile-request.ts` (user-side switches)
  - `web/src/components/access-feature-route-guard.tsx`
  - `web/src/layouts/components/app-sidebar/access-features.ts`
  - `web/src/pages/admin/access-profiles/*` (page, card, dialog, switch list, users-table select, admin hooks, tests)
  - Tests: `web/src/hooks/__tests__/use-access-profile-request.test.tsx`, `web/src/components/__tests__/access-feature-route-guard.test.tsx`
- **Files (upstream, edited — check these first after an upstream merge):**
  - `internal/dao/database.go`: two entities added to `autoMigrateModels()`.
  - `internal/admin/router.go`: `accessprofile.RegisterAdminRoutes(protected)`.
  - `internal/router/router.go`: `accessprofile.RequireFeature(...)` on the route groups above, plus `GET /api/v1/users/me/features`.
  - `internal/handler/auth.go`: `accessprofile.CheckAPIKeyAccess` after each API-token billing check (3 places).
  - `web/src/utils/api.ts`: access-profile endpoints. `web/src/services/admin-service.ts`: admin calls appended.
  - `web/src/routes.tsx`: `Routes.AdminAccessProfiles` + route. `web/src/pages/admin/layouts/navigation-layout.tsx`: nav item.
  - `web/src/pages/admin/users.tsx`: `access_profile` column.
  - `web/src/layouts/root-layout.tsx`, `web/src/pages/user-setting/index.tsx`: wrap `<Outlet />` in `AccessFeatureRouteGuard`.
  - `web/src/layouts/components/app-sidebar/{index,nav-item,user-footer}.tsx`, `web/src/pages/user-setting/sidebar/index.tsx`: filter entries by feature.
  - `web/src/hooks/use-mcp-request.ts`: MCP list/detail queries only run when MCP is allowed.
  - `web/src/pages/agent/form/agent-form/tool-popover/index.tsx`: MCP tab hidden when MCP is off.
  - `web/src/pages/dataset/setting/index.tsx`: data-source section hidden when Data sources is off.
  - `web/src/locales/{en,zh}.ts`: `accessProfile` section appended.
  - Tests mocking `@/hooks/use-access-profile-request`: `layouts/root-layout.test.tsx`, `layouts/components/app-sidebar/{index,user-footer}.test.tsx`, `pages/user-setting/{index,sidebar/index}.test.tsx`.
- **Database / config impact:** Two new tables created by AutoMigrate on startup: `access_profile` (`id`, `name` unique, `description`, `features` JSON) and `user_access_profile` (`user_id` PK, `profile_id`). The Default profile (`id = default`, all features on) is created on first admin use. With no profile rows nothing is restricted. No new config keys or env vars. No version-gated migration, so `docs/administrator/upgrade_and_migration/database_change_log.md` is unchanged.
- **Tests:**
  - Go: `bash build.sh --test ./internal/accessprofile/... ./internal/admin/... ./internal/router/...`. On Windows without native libs, `CGO_ENABLED=0 go test` in a `golang:1.27` container also covers these packages.
  - Web: `cd web && npx jest src/hooks/__tests__/use-access-profile-request.test.tsx src/components/__tests__/access-feature-route-guard.test.tsx src/pages/admin/access-profiles`
- **Upstream merge notes:**
  - If upstream renames or regroups the routes in `internal/router/router.go`, re-attach `RequireFeature` to the new groups. Grep for `accessprofile.` to find every hook.
  - If upstream adds a new place that accepts API keys (`ResolveAPIToken`), add `accessprofile.CheckAPIKeyAccess` there too.
  - If upstream adds settings pages, extend `AccessFeatureRoutes` / `IntegrationAccessFeatures` in `web/src/constants/access-profile.ts`.
  - If upstream (or EE) adds its own feature toggles to `ToggleModel()` etc., both checks run; no conflict.
  - Changes reach the API server within 30 s (per-process cache), because the admin console runs in a separate process.
- **Commits:** `git log --grep "CUST-001"` (branch `custom/access-profiles`)
