# Remove proxy IP deduplication for proxy-pool routing

## Goal

Allow multiple saved proxy configurations to use the same proxy pool endpoint when the upstream pool routes traffic by username/password, and ensure runtime account-bound requests still use the selected saved proxy credentials.

## What I already know

* User currently cannot add multiple proxy entries when they share the same configured IP/host endpoint.
* User wants to configure all proxy entries to the same proxy pool address, with routing determined by username.
* The backend stores proxies in `config.Config.Proxies` and accounts reference them by `proxy_id`.
* `internal/config/StableProxyID` currently derives a default proxy ID from type, host, port, and username.
* `internal/config/ValidateProxyConfig` rejects duplicate normalized proxy IDs.
* Runtime proxy resolution in `internal/deepseek/client/proxy.go` resolves an account's `ProxyID` to a saved proxy and includes ID/type/host/port/username/password in the proxy client cache key.

## Assumptions (temporary)

* The requested duplicate allowance is for same host/port entries as long as they are distinct saved proxy records, typically with different usernames.
* Explicit duplicate `id` values must still be rejected because account bindings use `proxy_id` as the unique reference key.
* Runtime behavior should keep resolving by `proxy_id` and use the selected record's credentials.

## Requirements

* Adding/saving proxies must no longer reject records merely because they share the same proxy IP/host endpoint.
* Every saved proxy record is allowed as long as its normalized `id` is unique.
* Multiple proxy records may point to the same proxy pool address, including records that differ only by username/password or metadata.
* Account proxy binding must continue to reference a unique proxy record by `proxy_id`.
* Runtime requests must use the credentials from the account-bound proxy record.

## Acceptance Criteria

* [ ] Admin proxy creation allows multiple proxies with the same type/host/port when their IDs are unique.
* [ ] Config validation still rejects duplicate proxy IDs.
* [ ] Account proxy assignment and runtime lookup remain unambiguous by `proxy_id`.
* [ ] Runtime proxy client caching does not collapse different credentials for the same pool endpoint.
* [ ] A regression test covers same pool endpoint with different users.

## Definition of Done (team quality bar)

* Tests added/updated for the changed behavior.
* Relevant Go files formatted with `gofmt`.
* Targeted Go tests pass.
* Documentation updated if the public API behavior changes.

## Out of Scope (explicit)

* Changing proxy routing strategy away from account `proxy_id` bindings.
* Adding proxy pool discovery, health balancing, or automatic username rotation.
* Changing upstream proxy protocol support.

## Technical Approach

Keep `proxy_id` as the only uniqueness boundary for saved proxy records. Adjust default proxy ID generation if needed so auto-generated IDs no longer collapse records that share a pool endpoint but need separate saved identities. Preserve runtime lookup by `proxy_id`; the existing runtime cache key includes credentials, so different usernames/passwords should remain distinct.

## Decision (ADR-lite)

**Context**: Proxy pools can expose one shared host/port while routing by username/password, so endpoint-based deduplication blocks valid configurations.
**Decision**: Saved proxy records are unique only by normalized `id`; same type/host/port is allowed.
**Consequences**: Users can intentionally create multiple records for one pool endpoint. Duplicate explicit IDs remain invalid because accounts bind proxies by ID.

## Technical Notes

* `internal/config/config.go`: proxy model and stable default ID generation.
* `internal/config/validation.go`: proxy validation rejects duplicate IDs.
* `internal/httpapi/admin/proxies/handler_proxies.go`: add/update/delete/test and account proxy binding handlers.
* `internal/deepseek/client/proxy.go`: runtime account proxy resolution and per-proxy client cache key.
