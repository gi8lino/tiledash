# Mock Jira API Server

This mock server emulates Jira's `/rest/api/2/search` endpoint for **testing `jirapanel` locally** without connecting to a real Jira instance.

> ⚠️ This is for **development and testing only**. It should **not** be used in production.

## How It Works

When `jirapanel` runs, it queries the Jira API with a JQL like:

```bash
GET /rest/api/2/search?jql=filter=17201
```

The mock server intercepts this request and returns the contents of a static JSON file named:

```text
data/17201.json
```

It looks up files using the filter ID from the query (`filter=17201` → `data/17201.json`), which should match the `filterId` in your `config.yaml`.

## Example File Layout

```text
mock-server/
├── main.go
├── data/
│   ├── 17201.json
│   ├── 17203.json
│   └── 17206.json
└── Makefile
```

Each file must be named `<filterId>.json`, where `<filterId>` is a number from your dashboard config:

```yaml
layout:
  - filterId: 17201
    title: Issues
    template: issues
  - filterId: 17203
    title: Bugs
    template: issues
```

## Usage

### Start the server:

```bash
make run
```

Or run manually:

```bash
go run ./main.go --port=8081 --data-dir=./data --random-delay
```

### Available Flags

| Flag             | Description                                   | Default  |
| :--------------- | :-------------------------------------------- | :------- |
| `--port`         | Port to run mock server on                    | `8081`   |
| `--data-dir`     | Directory to serve JSON data from             | `./data` |
| `--random-delay` | Add random 200-1000 ms delay to each response | `false`  |

## Integration with jirapanel

To use this server with `jirapanel`, simply configure your Jira base URL to point to it:

```yaml
jira:
  baseUrl: http://localhost:8081
```

This makes `jirapanel` fetch data from your mock server instead of the real Jira API.
