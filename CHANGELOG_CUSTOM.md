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

### [CUST-004] Fix duplicated fields in data source forms (Confluence Space Key) — 2026-11-19

- **Type:** fix
- **Requirement:** In the Confluence data source form, the Space Key field repeated many times after switching the index mode.
- **Change:**
  - `confluence-constant.tsx` declares `config.space`, `config.page_id` and `config.index_recursively` twice: a visible field and a hidden copy.
  - `DynamicForm` keyed each field by `field.name`, so the two entries got the same React key. React warned "Encountered two children with the same key" and duplicated DOM nodes on re-render.
  - Fields are now keyed by name plus position, which also protects every other data source form with repeated names.
  - Present in upstream v1.0.0 and in upstream `main` as of 2026-11-19. A good candidate to send upstream.
- **Files (new):** `web/src/pages/user-setting/data-source/__tests__/confluence-form-fields.test.tsx`. It renders the real Confluence config, switches the mode six times, and asserts no duplicate-key warning and exactly one visible Space Key.
- **Files (upstream, edited):** `web/src/components/dynamic-form.tsx`: `key={`${field.name}-${index}`}`.
- **Database / config impact:** none
- **Tests:** `cd web && npx jest src/pages/user-setting/data-source/__tests__/confluence-form-fields.test.tsx`. Without the fix it fails on the duplicate-key warning; with the fix it passes. No new failures in the suites that render `DynamicForm`. The 6 type errors in `dynamic-form.tsx` and three dataset suites that fail are pre-existing on `main`.
- **Upstream merge notes:** if upstream fixes this itself (de-duplicating the Confluence config or changing the key), drop this change and keep whichever fix upstream chose.
- **Commits:** `git log --grep "CUST-004"`

### [CUST-003] Global model providers — 2026-11-19

- **Type:** feature, security
- **Jira:** sub-task of AI-145 "RAG Platform (RAGFlow) customization"
- **Requirement:** A super admin configures model providers, keys and models once, and every user can call them without their own setup. Users can still add their own providers. Global models are read-only for users: they can be used but never edited, disabled, deleted, re-keyed or read. Access profiles decide which global models a user may call.
- **Status:** A1 backend, A2 admin UI and A3 user side are done.
- **Change (A1):**
  - **Storage.** Global providers, instances and models live in the regular `tenant_model_provider`, `tenant_model_instance` and `tenant_model` tables, owned by the reserved tenant and user ID `global` (`entity.GlobalModelOwnerID`). Upstream's catalog and provider handling, and references by model ID, work unchanged. Every provider a user can configure can also be configured globally.
  - **One use rule.** `internal/globalmodel` provides `ProviderByName` (the tenant's own provider first, then global), `OwnedOrGlobalProviderIDs`, and `CheckUse`.
    - `CheckUse` refuses a global model unless its instance and model are enabled and the calling user's access profile allows it. The caller is the request's user, or the tenant owner for background work.
    - It replaces the tenant-ownership check at the 6 places that *use* a model: `service/model_factory.go`, `agent/component/llm_credentials.go` (by name, by model ID, by instance ID, and the credential fallback), `ingestion/component/dispatch_model.go` (by ID, by name, provider by name), `ingestion/component/pdf_parser_model.go`, and `agentic_rag/decision_config.go`.
    - For a global provider, the factory skips the upstream non-owner and per-resource permission checks, because `CheckUse` applies its rules instead.
  - **Read-only for users.** The *manage* paths (`/providers/...`, which go through `findProviderByIDOrNameInCurrentTenant` and `resolveInstance` in `service/model_service.go`) are not changed. Their tenant check never matches `global`, so users cannot list, edit, disable, delete or re-key a global provider, instance or model.
  - **Strict stop.** Disabling a global instance or model (status `inactive`) refuses every call at once, for every user, including existing chats, datasets and agents. A profile that doesn't allow a global model refuses every call by its users. A refused model in agentic RAG's decision config falls back to the environment judge, never to the refused model.
  - **Encrypted keys.**
    - Global API keys are AES-256-GCM encrypted (`enc:model:v1:` prefix) with `RAGFLOW_CONNECTOR_KEY` (`common/model_api_key.go`). Without the key, a global key cannot be saved.
    - A GORM `AfterFind` hook (`entity/global_model.go`) decrypts keys on load, so every place that reads `instance.APIKey` gets the plaintext without changes. If decryption fails, the key is cleared and the call fails.
    - A `BeforeSave` hook re-encrypts a global key if loaded code saves the instance back.
    - Keys are never returned: the admin listing shows `has_api_key` and `••••1234`.
  - **Global defaults.** The admin sets a default model per type (chat, embedding, speech2text, image2text, rerank, tts, ocr, decision) in `system_settings` (`global_models.defaults`). A user with no default of that type gets the global one, applied at lookup time with no copying. A user's own default still wins.
  - **Access profiles.** New `global_models` setting (`{"all": true}` by default, or a list of global model IDs). It's returned in `/users/me/features`, and an update that omits it keeps the current value.
  - **Admin API** (superuser only, admin console): `/api/v1/admin/global-models` — list, catalog, providers, instances (create, update key/endpoint/status, delete, test connection), models (add, enable/disable, delete), and defaults. Deletes are refused, with counts, while a dataset, chat, memory, agent, user default or global default uses the model. Every change is audit-logged without keys.
- **Change (A2 admin UI):**
  - New admin console page **Global models** (`/admin/global-models`, `web/src/pages/admin/global-models/`):
    - add a catalog provider;
    - add instances (base URL, optional API key and region, models with types and max tokens) with **Test connection**;
    - enable or disable an instance or a model (the hard stop);
    - add or delete models, delete instances and providers (refused with the reason while in use);
    - set global defaults per type.
  - Default keys are sent as `asr` / `vision` because the admin request client snake-cases top-level keys.
  - Each access-profile card gets a **Global models** section (all, or a checked allow list).
- **Change (A3 user side):**
  - `GET /api/v1/models` appends the global models the caller may use (`service/global_model_list.go`, called from `handler/providers.go`), marked `is_global`, with no key or endpoint.
  - `GET /api/v1/models/default` falls back to the global default (`service/user_tenant.go`).
  - In the frontend, every model dropdown shows global models in their own provider group with a **Global** badge (`components/model-tree-select.tsx`, `components/global-model-badge.tsx`).
  - The Model providers page's default view lists them read-only under "Provided by your organization" (`pages/user-setting/setting-model/organization-models.tsx`).
  - Users cannot edit them anywhere.
- **Files (new):** `internal/globalmodel/{globalmodel,defaults,endpoint}.go` + test, `internal/service/global_model_list.go`, `web/src/pages/admin/global-models/*` (page, hooks, constants, provider card, instance and add-provider dialogs, defaults card, tests), `web/src/pages/admin/access-profiles/global-model-access.tsx` + test, `web/src/components/global-model-badge.tsx`, `web/src/components/__tests__/model-tree-select-global.test.tsx`, `web/src/pages/user-setting/setting-model/organization-models.tsx` + test, `web/src/interfaces/{database,request}/global-model.ts`, `internal/common/model_api_key.go` + test, `internal/entity/global_model.go`, `internal/accessprofile/global_models.go`, `internal/service/global_model_admin.go`, `internal/service/global_model_test.go`, `internal/admin/global_models_handler.go`, `internal/ingestion/component/global_model_test.go`, `internal/agent/component/global_model_test.go`
- **Files (upstream, edited — check these first after an upstream merge):**
  - `internal/service/model_factory.go`: provider by name through `globalmodel.ProviderByName`; global branch before the tenant check; non-owner checks skipped for global; global default fallback in `defaultModelReference`.
  - `internal/agent/component/llm_credentials.go`: 4 lookups (see above).
  - `internal/ingestion/component/dispatch_model.go`: owned-or-global scope, `CheckUse` in both resolvers, provider by name, global default fallback.
  - `internal/ingestion/component/dispatch_model_test.go`: the scoped-SQL test expects the owned-or-global scope (still fully parameter-bound).
  - `internal/ingestion/component/pdf_parser_model.go`, `internal/agentic_rag/decision_config.go`: global exception + `CheckUse`.
  - `internal/admin/router.go`: `RegisterGlobalModelRoutes(protected)`.
  - `internal/handler/providers.go`: `AppendGlobalModels` after the user's own models in `ListModelsForUserInTenant`.
  - `internal/service/user_tenant.go`: global default fallback in `ListDefaultModels`.
  - `web/src/components/model-tree-select.tsx`: global provider group + Global badge.
  - `web/src/interfaces/database/llm.ts`: `is_global` on `IAddedModel`.
  - `web/src/pages/user-setting/setting-model/index.tsx`: `<OrganizationModels />` under the default models view.
  - `web/src/routes.tsx`, `web/src/pages/admin/layouts/navigation-layout.tsx`: Global models route and nav item.
  - `web/src/utils/api.ts`, `web/src/services/admin-service.ts`: global model endpoints and calls.
  - `web/src/locales/{en,zh}.ts`: `globalModel` section and access-profile strings.
  - Own files from CUST-001: `internal/entity/access_profile.go` (`global_models` column), `internal/dao/access_profile.go`, `internal/accessprofile/service.go`, `internal/admin/access_profile_routes_test.go` (now covers the global-model routes too).
- **Database / config impact:**
  - New column `access_profile.global_models` (AutoMigrate).
  - New `system_settings` row `global_models.defaults`, created on the first save.
  - Global rows in the three provider tables with tenant and user `global`.
  - `RAGFLOW_CONNECTOR_KEY` is required to save global keys. For local runs it comes from `docker/local-ca/secrets.env` (git-ignored); back it up.
- **Tests:**
  - `bash build.sh --test ./internal/globalmodel/... ./internal/common/... ./internal/service/... ./internal/agent/component/... ./internal/ingestion/component/... ./internal/accessprofile/... ./internal/admin/...`
  - In a CGO-free container, four PDF/DOCX parser tests in `ingestion/component` fail on `main` too (they need the native libraries).
- **Upstream merge notes:**
  - Grep for `globalmodel.` to find every hook.
  - If upstream adds a new place that loads a provider or instance to *call* a model, route it through `globalmodel.ProviderByName` / `CheckUse`. If it's a *manage* path, leave it tenant-only, which keeps global models read-only.
  - If upstream adds its own `AfterFind` / `BeforeSave` on `TenantModelInstance`, the build fails on the duplicate method; merge the logic into `entity/global_model.go`.
- **Commits:** `git log --grep "CUST-003"`

### [CUST-002] Local Docker build and run from the fork — 2026-11-18

- **Type:** config
- **Jira:** AI-146 (tooling used to test it)
- **Requirement:** Build the RAGFlow image from this checkout and run it locally, behind the Zscaler TLS-intercepting proxy, without editing upstream Docker files.
- **Change:**
  - `docker/docker-compose.local.yml` runs the stack with the image `ragflow:access-profiles` built from this checkout. It uses the same settings as `ragflow-cpu` (API 9380, Admin 9381, web 80). It also defines two helper images that add `docker/local-ca/bundle.pem` to the Dockerfile's base images, so downloads during the build trust the corporate proxy.
  - The RAGFlow image is built with `docker build ... --build-context`, not compose. Compose's bake builder dropped the `infiniflow/github_action_runner` override. `GOPROXY=https://proxy.golang.org,direct` skips `goproxy.cn`, which the proxy rejects. The full commands are in the compose file's header.
  - `docker/local.env` is passed after `.env` with `--env-file`. It moves host ports that clash on a developer machine: MinIO to 9020 (Zscaler holds 9000 and 9010) and MySQL to 3307 (local MySQL). Containers still use the internal ports.
  - `docker/local-ca/` holds the certificate bundle. Its `.gitignore` keeps everything except itself out of git.
  - Local secrets go in `docker/local-ca/secrets.env` (git-ignored, optional), read by `ragflow-local` after `.env`. `RAGFLOW_CONNECTOR_KEY` lives there for CUST-003.
- **Files (new):** `docker/docker-compose.local.yml`, `docker/local.env`, `docker/local-ca/.gitignore`
- **Files (upstream, edited):** none
- **Database / config impact:** none (local runs only)
- **Tests:** manual — the stack starts and the CUST-001 API checks pass against it.
- **Upstream merge notes:**
  - If upstream changes the `ragflow-cpu` service in `docker/docker-compose.yml`, mirror it in `ragflow-local`. That covers ports, volumes, command and environment.
  - If upstream changes the Dockerfile's base images (`ubuntu:24.04`, `infiniflow/github_action_runner:latest`), update the helper images and the `--build-context` names.
- **Commits:** `git log --grep "CUST-002"`

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
