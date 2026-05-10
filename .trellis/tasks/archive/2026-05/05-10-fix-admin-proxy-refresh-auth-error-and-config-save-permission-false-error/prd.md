# Fix admin proxy refresh auth error and config save permission false error

## Goal

Fix two Admin UI proxy-management bugs: refreshing the `/admin/proxies` browser route should keep serving the SPA instead of returning a protected API JSON error, and failed config file persistence must not leave the UI showing changes that only exist in process memory.

## What I already know

* User reports refreshing `/admin/proxies` consistently returns `{"detail": "authentication required"}`.
* User reports saving proxy configuration returns `{"detail":"open /data/config.json: permission denied"}`, while the refreshed page still shows the new config.
* Backend registers `/admin/proxies` as a protected JSON API route in `internal/httpapi/admin/proxies/routes.go`.
* The SPA also uses `/admin/proxies` as a browser tab route in `webui/src/layout/DashboardShell.jsx`.
* Server fallback for `/admin/*` only runs on NotFound in `internal/server/router.go`, so exact protected API routes such as `GET /admin/proxies` intercept browser refreshes before WebUI fallback can serve `index.html`.
* Proxy mutations call `Store.Update` in `internal/httpapi/admin/proxies/handler_proxies.go`.
* `Store.Update` currently assigns `s.cfg = cfg` before `saveLocked()` in `internal/config/store.go`, so a persistence error leaves in-memory config mutated even though `/data/config.json` was not written.

## Assumptions (temporary)

* Browser document navigations to known Admin SPA tab paths should prefer the WebUI shell, while programmatic API calls to the same paths should keep returning JSON and requiring Authorization.
* If file-backed persistence fails during a config mutation, the mutation should fail atomically: no in-memory commit, no pool reset side effects based on unsaved config, and the UI should continue showing the last persisted config after refresh.

## Requirements

* Refreshing or directly opening `/admin/proxies` in a browser must serve the Admin SPA shell instead of the protected API error.
* Authenticated API calls to `GET /admin/proxies` must keep working as JSON API calls.
* Unauthenticated/non-browser API calls to `GET /admin/proxies` must keep returning an auth error.
* Proxy create/update/delete must not mutate in-memory config if writing the configured file path fails.
* The fix must include targeted investigation of why `/data/config.json` is not writable and surface that as verification evidence or an operator-facing note.
* Existing env-backed behavior must remain unchanged: when writeback is intentionally disabled, saves should still skip file writes without reporting permission errors.

## Acceptance Criteria

* [ ] A browser-style `GET /admin/proxies` request with `Accept: text/html` returns the WebUI shell.
* [ ] A JSON/API-style unauthenticated `GET /admin/proxies` request still returns `401` with an auth error.
* [ ] An authenticated API `GET /admin/proxies` request still returns the proxy JSON payload.
* [ ] When config persistence returns `permission denied`, proxy mutation returns an error and the store snapshot remains unchanged.
* [ ] Relevant Go tests cover the route conflict and config update atomicity.

## Definition of Done

* Tests added/updated for the changed backend behavior.
* Relevant Go tests pass.
* `gofmt` applied to changed Go files.
* No unrelated refactors or behavior changes.

## Technical Approach

Recommended minimal approach:

1. Add content-negotiated Admin SPA handling for browser document requests to known Admin tab paths that overlap API routes, especially `/admin/proxies`, while preserving JSON API behavior for fetch/curl clients.
2. Make `config.Store.Update` commit in-memory state only after persistence succeeds for file-backed stores, so save errors do not create process-memory-only config changes.

## Decision (ADR-lite)

**Context**: `/admin/proxies` is both a SPA route and a protected JSON API route; `Store.Update` mutates memory before persistence.

**Decision**: Pending user confirmation.

**Consequences**: Browser refreshes work without renaming API endpoints, and permission failures become honest hard failures instead of false-success memory updates.

## Out of Scope

* Changing proxy API paths.
* Automatically fixing host/container filesystem permissions for `/data/config.json`.
* Redesigning the whole Admin routing/auth system.
* Broad frontend UI changes beyond any needed refresh/error handling.

## Technical Notes

* `internal/server/router.go` registers Admin API routes before WebUI fallback; exact API route matches bypass NotFound fallback.
* `internal/webui/handler.go` already serves SPA fallback for `/admin/*` NotFound GET requests.
* `internal/httpapi/admin/auth/handler_auth.go` currently returns JSON auth errors from protected middleware before handlers run.
* `internal/config/store.go` needs atomicity around `Update`; `Replace`, `Save`, and env-backed skip behavior should be checked for consistency.
