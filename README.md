# tiledash (formerly **JiraPanel**)

[![Go Report Card](https://goreportcard.com/badge/github.com/gi8lino/tiledash?style=flat-square)](https://goreportcard.com/report/github.com/gi8lino/tiledash)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/gi8lino/tiledash)
[![Release](https://img.shields.io/github/release/gi8lino/tiledash.svg?style=flat-square)](https://github.com/gi8lino/tiledash/releases/latest)
[![GitHub tag](https://img.shields.io/github/tag/gi8lino/tiledash.svg?style=flat-square)](https://github.com/gi8lino/tiledash/releases/latest)
![Tests](https://github.com/gi8lino/tiledash/actions/workflows/tests.yml/badge.svg)
[![Build](https://github.com/gi8lino/tiledash/actions/workflows/release.yml/badge.svg)](https://github.com/gi8lino/tiledash/actions/workflows/release.yml)
[![license](https://img.shields.io/github/license/gi8lino/tiledash.svg?style=flat-square)](LICENSE)

## What changed?

This project was previously called **JiraPanel**. It’s now **tiledash** with a more general model:

- ✅ Not just Jira — **any HTTP API** via “providers”
- ✅ First-class **pagination** (query or body)
- ✅ Request-level **TTL caching**
- ✅ Cleaner config & validation, generic **auth** (basic or bearer)
- ✅ Go templates as before, but a clearer **data shape** for paginated responses

If you used JiraPanel, see **Migration notes** at the end.

## Features

- 🧱 Grid of tiles rendered from Go HTML templates
- 🌍 Pluggable **providers** (base URL + auth) you can reuse across tiles
- 🔎 Flexible per-tile request: method, path, query/headers, raw or JSON body
- 📑 **Pagination** (query/body) with automatic page merging + de-dup
- ⚡ Per-request **cache TTL**
- 🔁 Auto-refresh interval for the whole dashboard
- 🎨 CSS-like **customization** via YAML (fonts, spacing, cards)
- 🧰 Minimal CLI (config path, template dir, bind address, logs)
- 🧪 Simple **mock server** for local development

## How it works

1. The dashboard (`/`) renders the grid shell.
2. The browser fetches each tile from `/api/v1/tile/{id}`.
3. The server executes the configured HTTP request (and pagination if enabled).
4. JSON is decoded and passed to the tile’s `.gohtml` template.
5. Failures render an error partial instead of breaking the page.

Each tile is independent.

## Configuration

tiledash is configured with a single **YAML** file plus a folder of `.gohtml` templates.

### Top-level

```yaml
title: My Dashboard
refreshInterval: 60s
grid:
  columns: 2
  rows: 5

customization:
  grid:
    gap: 1rem
    padding: 0rem
    marginTop: 0rem
    marginBottom: 0rem
  card:
    borderColor: "#ddd"
    padding: 1rem
    backgroundColor: "#fff"
    borderRadius: 0.5rem
    boxShadow: 0 2px 4px rgba(0, 0, 0, 0.05)
  header:
    align: center
    marginBottom: 0.5rem
  footer:
    marginTop: 1rem
  font:
    family: "Segoe UI, sans-serif"
    size: 16px
```

### Providers (base URL + auth)

```yaml
providers:
  jira-v2:
    baseURL: "https://jira.example.com"
    skipTLSVerify: false
    auth:
      basic:
        username: "me@example.com"
        password: "JIRA_API_TOKEN"
    # or:
    # auth:
    #   bearer:
    #     token: "YOUR_BEARER_TOKEN"
```

Auth values (`providers.*.auth.basic.username`, `providers.*.auth.basic.password`, `providers.*.auth.bearer.token`) are resolved using [containeroo/resolver](https://github.com/containeroo/resolver) before use.
That means you can reference environment variables or other resolver-supported sources instead of hardcoding secrets. See the resolver docs for syntax and supported backends.

### Tiles

```yaml
tiles:
  - title: issues
    template: issues.gohtml
    position: { row: 1, col: 1, colSpan: 2 } # 1-based indexing
    request:
      provider: jira-v2
      method: GET
      path: /rest/api/2/search
      ttl: 20s
      query:
        jql: filter=17203
        maxResults: 50
      # headers:
      #   X-Whatever: abc

      # One of:
      # body: '{"raw":"payload"}'
      # bodyJSON:
      #   project: "ABC"

      paginate: true # boolean (not string)
      page:
        location: query # "query" or "body"
        startField: startAt # fields in the RESPONSE
        limitField: maxResults
        totalField: total
        reqStart: startAt # fields in the REQUEST (query/body)
        reqLimit: maxResults
        # limitPages: 3         # optional cap
```

#### Request fields at a glance

- `provider`: which configured provider to use
- `method`: default `GET`
- `path`: relative to provider’s `baseURL`
- `ttl`: cache duration (Go duration string, e.g. `30s`)
- `query`, `headers`: string maps
- `body`: raw body string
- `bodyJSON`: an object to be JSON-encoded (auto sets `Content-Type: application/json` unless you override)
- `paginate`: enable pagination
- `page`: pagination wiring (names in response vs. request)

> Pagination merges top-level array fields across pages into a single array in the **accumulator’s** `merged` map with de-duplication by `id`/`key` (if present), otherwise by structure.

## Templates

Templates are Go HTML templates (`.gohtml`). Every tile template receives:

- `.Title` — tile title
- `.ID` — 0-based tile index
- `.Data` — **primary payload** (if pagination: usually the merged page or the first page; otherwise the object itself)
- `.Acc` — full **accumulator** when pagination/merging is used:

  - `.Acc.pages` — raw page payloads in order
  - `.Acc.merged` — concatenated arrays by key, de-duplicated

- `.Raw` — original input (for debugging)

### Example (Jira issues table)

```gohtml
{{/* If pagination is enabled and "merged.issues" exists, .Data will be the merged map itself. */}}
<h2>{{ .Title }}</h2>

{{ $merged := .Acc.merged }}
{{ $issues := index $merged "issues" }}
<ul>
  {{- range $i := $issues }}
    <li>{{ index (index $i "fields") "summary" }}</li>
  {{- end }}
</ul>
```

### Template helpers

tiledash ships with [sprig](https://masterminds.github.io/sprig/) (HTML-safe map) **plus** a few custom helpers:

- `formatJiraDate input layout` — parse Jira timestamp, format with `layout`
- `setany map key value` — set a key on `map[string]any` and return the map
- `dig map key` — safe string lookup
- `sortBy field desc slice` — sort `[]any` of `map[string]any` by field
- `appendSlice slice item` — append to a `[]any`
- `uniq []string` — unique strings
- `defaultStr val fallback` — fallback if empty/whitespace
- `typeOf v` — Go type string
- `sumBy field []map[string]any` — sum numeric field

## Running

```bash
tiledash \
  --config ./config.yaml \
  --template-dir ./templates \
  --listen-address :8080 \
  --log-format text \
  --debug
```

Flags:

- `--config` (path to YAML; default `config.yaml`)
- `--template-dir` (default `templates`)
- `--listen-address` (default `:8080`)
- `--route-prefix` (path prefix to mount the app (e.g., /tiledash).
- `--log-format` (`text` or `json`)
- `--debug` (bool)

## Endpoints

| Path                | Method | Description         |
| :------------------ | :----- | :------------------ |
| `/`                 | GET    | Dashboard           |
| `/api/v1/tile/{id}` | GET    | Render tile by ID   |
| `/api/v1/hash/{id}` | GET    | Hash of a tile spec |
| `/healthz`          | GET    | Health check        |
| `/static/*`         | GET    | Static assets       |

> Notes: IDs are 0-based. Hash endpoints are useful for cache-busting on the client.

> Note: If you set `--route-prefix=/tiledash`, the above paths will be `/tiledash/{PATH}` instead of `/`.

## Local development: mock server

If you want to develop templates and requests without hitting real APIs, use the included mock server. It can emulate Jira’s `/rest/api/2/search` with optional pagination.

➡️ See **tests/README.md** for full usage, flags, and data layout.

## Migration notes (from JiraPanel)

- **Config**: Jira-specific fields were replaced by a generic HTTP request (`request.method`, `request.path`, `request.query`, `request.bodyJSON`, etc.).
- **Templates**: Pagination data now lives in the **accumulator**. Use `.Acc.merged.<key>` and `.Acc.pages` when you need all pages; `.Data` is the “primary” view (usually merged or first page).
- **Endpoints**: tile endpoint is now `/api/v1/tile/{id}` (singular).
- **Auth**: `providers.*.auth` supports **basic** and **bearer**.

## License

Apache 2.0 — see [LICENSE](LICENSE).
