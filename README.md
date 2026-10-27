# headless-broadcaster

Broadcast an HTTP request to every IP address resolved for a hostname.

Configure the server with `config.yaml` or `HB__WEB_HOST`, `HB__WEB_PORT`, and
`HB__WEB_API_KEY`. The default listener is `127.0.0.1:8080`. Send requests to
`/broadcast?url=https://example.com/path` with an `X-Api-Key` header when an API
key is configured. `/api/ping` and `/api/health` are public health endpoints.

This tool can reach addresses accessible from its host and forwards request
headers and bodies. Run it on a trusted network with an API key. An empty key
allows unauthenticated requests. Outbound TLS certificate verification is
currently disabled. Avoid query-string authentication because request URLs
can appear in logs.

Run `make lint test build` and `goreleaser check` to validate changes.
`VERSION` and the release tag must agree. Releases publish binaries and
containers for Docker Hub and GHCR; only tag pushes publish artifacts.
The release script creates a local commit and signed tag; publishing requires
an explicit push.

Licensed under the GNU Affero General Public License v3.0.

### Broadcast results and connection lifecycle

A destination succeeds only when its final HTTP status is in the full 200–299
range and no transport or response-body read error occurred. The response
includes per-destination status/error details and total, success, and failed
counts. The aggregate HTTP status is 200 when all destinations succeed and 500
when any destination fails, including a truncated body received with status 200.
An aggregate failure does not prove the operation was not applied upstream;
inspect individual results before deciding whether a retry is safe.

Each destination request owns its HTTP client and transport. Once its response
body has been read and closed (or the request fails), idle connections are
explicitly closed. Connections are not pooled across destinations or broadcasts;
the target-IP dial binding, request cancellation, and 30-second timeout remain
unchanged. Regression tests use local HTTP fixtures only.
