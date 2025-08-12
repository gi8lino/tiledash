# Mock Jira API Server

This mock server emulates Jira’s `/rest/api/2/search` endpoint so you can test **tiledash** locally without connecting to a real Jira instance.

> ⚠️ For **development and testing only** – not for production.

## What it does

When tiledash issues a request like:

```bash
GET /rest/api/2/search?jql=filter=17201
```

the mock server extracts the filter ID (`17201`) and returns the contents of a static JSON file:

```text
data/17201.json
```

It also supports simple pagination so you can exercise tiledash’s query/body pagination paths.

## Project layout

```text
mock-server/
├── main.go
├── README.md
└── data/
├── 17201.json
├── 17203.json
└── 17206.json
```

Each file is named `<filterId>.json` and should roughly match Jira’s search response shape (e.g., `issues`, `startAt`, `maxResults`, `total`).

### Example `data/17203.json`

```json
{
  "total": 3,
  "startAt": 0,
  "maxResults": 50,
  "issues": [
    { "id": "1", "fields": { "summary": "A" } },
    { "id": "2", "fields": { "summary": "B" } },
    { "id": "3", "fields": { "summary": "C" } }
  ]
}
```

## Running

### With make

```bash
make run-mock
```

### Or directly

```bash
go run ./tests/main.go --config=./tests/config.yaml
```

### Flags

| Flag         | Description                                   |
| :----------- | :-------------------------------------------- |
| `--config`   | Path to mock-server config.yaml (required)    |
| `--log-body` | Log JSON request bodies (may contain secrets) |

## Using with tiledash

Point a provider at the mock server:

```yaml
providers:
  mock:
    baseURL: "http://localhost:8081"
```

Then create a tile that calls Jira search:

```yaml
tiles:
  - title: Mocked Issues
    template: issues.gohtml
    position: { row: 1, col: 1 }
    request:
      provider: mock
      method: GET
      path: /rest/api/2/search
      query:
        jql: filter=17203
      paginate: true
      page:
        location: query
        startField: startAt
        limitField: maxResults
        totalField: total
        reqStart: startAt
        reqLimit: maxResults
```

## Pagination behavior

When `paginate: true` is set, the mock server will:

- Read the **request** start/limit from either query (`startAt` / `maxResults`) or body (if you’re testing body-based pagination).
- Slice the `issues` array accordingly.
- Emit `startAt`, `maxResults`, and `total` in the **response** so tiledash can request subsequent pages.

This lets you verify:

- Query-based pagination
- Body-based pagination
- Merging behavior in templates via `.Acc.merged` and `.Acc.pages`
