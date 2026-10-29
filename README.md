# headless-broadcaster

Broadcast an HTTP request to every IP address resolved for a hostname.

Configure the server with `config.yaml` or `HB__WEB_HOST`, `HB__WEB_PORT`, and
`HB__WEB_API_KEY`. The default listener is `127.0.0.1:8080`. Send requests to
`/broadcast?url=https://example.com/path` with an `X-Api-Key` header when an API
key is configured. The destination must be an absolute `http://` or `https://`
URL with a hostname; other forms receive HTTP 400 before DNS lookup or forwarding.
`/api/ping` and `/api/health` are public health endpoints.

This tool can reach addresses accessible from its host and forwards request
headers and bodies. Run it on a trusted network with an API key. An empty key
allows unauthenticated requests. Avoid query-string authentication because
request URLs can appear in logs.

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

The broadcaster negotiates upstream response compression independently of the
caller's `Accept-Encoding` header. Go's transport negotiates and decodes gzip
before responses are parsed into aggregate JSON values or text. Invalid gzip is
reported as a destination read error rather than a successful string payload.

Each destination request owns its HTTP client and transport. Once its response
body has been read and closed (or the request fails), idle connections are
explicitly closed. Connections are not pooled across destinations or broadcasts;
the target-IP dial binding, request cancellation, and 30-second timeout remain
unchanged. Regression tests use local HTTP fixtures only.

### Credentials, TLS, and body limits

Copy `config.example.yaml` to `config.yaml` to configure these defaults.
Broadcaster authentication accepts `X-Api-Key`, the outer `apiKey` query
parameter, an `apiKey` cookie, or a Basic-auth password (in that priority order).
`X-Api-Key` and `apiKey` cookies are reserved for the broadcaster and are never
forwarded, including redundant copies. Basic Authorization values whose trimmed
password equals the configured broadcaster key are also removed. Other
Authorization values (Bearer or distinct Basic credentials), application
cookies, and repeated headers remain available for intentional downstream auth.
When broadcaster authentication is disabled, Basic auth passes through.
The outer query is not forwarded; the nested destination URL, including its own
query credentials or userinfo, is intentional downstream input and is retained.
Use separate credentials for the broadcaster and upstream applications.

Outbound HTTPS now verifies certificates and the destination URL's hostname
using system trust, while connecting to each selected IP. SNI uses the original
hostname. For a private CA, set `proxy.caFile` (`HB__PROXY_CA_FILE`) to a PEM bundle
that is added to system roots at startup. Missing or invalid bundles fail startup.
For compatibility only, `proxy.insecureSkipVerify: true`
(`HB__PROXY_INSECURE_SKIP_VERIFY=true`) disables certificate and hostname checks;
the default is `false`. Restart after changing configuration or CA files.

`proxy.maxRequestBodyBytes` (`HB__PROXY_MAX_REQUEST_BODY_BYTES`) defaults to
1048576 bytes (1 MiB). Oversized inbound bodies receive HTTP 413 before any
upstream request is made. `proxy.maxResponseBodyBytes`
(`HB__PROXY_MAX_RESPONSE_BODY_BYTES`) defaults to 4194304 bytes (4 MiB) per
upstream. Limits accept positive byte counts below MaxInt64; zero and negative
values fail startup. They apply to actual reads, including chunked/unknown-length
bodies; responses are measured after automatic decompression. Bodies exactly at
the limit are accepted. Oversized upstream bodies are closed and reported with
an `err` and their HTTP status, without returning a truncated payload as success.
Other destinations continue, and successful results remain in the aggregate
response (HTTP 500 when any destination fails). Cancellation and the 30-second
per-destination timeout still interrupt outbound body reads. These are per-body
bounds; total aggregate size still scales with the number of resolved addresses.
