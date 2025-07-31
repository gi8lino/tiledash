# JiraPanel

**JiraPanel** is a flexible, self-hosted dashboard for visualizing data from your Jira Cloud or Server instance using templates and JQL queries.

## Features

- Render multiple dashboard sections with custom templates
- Query Jira issues using JQL or filters
- Fully configurable grid layout
- Auto-refresh support (configurable)
- Clean bootstrap-based layout
- Simple CLI configuration via flags or environment variables

## Example Usage

```sh
jirapanel \
  --config dashboard.yaml \
  --template-dir templates \
  --jira-api-url https://yourcompany.atlassian.net/rest/api/3 \
  --jira-email alice@yourcompany.com \
  --jira-token xxxx
```

Or using environment variables:

```sh
JIRAPANEL_CONFIG=dashboard.yaml \
JIRAPANEL_TEMPLATE_DIR=templates \
JIRAPANEL_JIRA_API_URL=https://yourcompany.atlassian.net/rest/api/3 \
JIRAPANEL_JIRA_EMAIL=alice@yourcompany.com \
JIRAPANEL_JIRA_TOKEN=xxxx \
jirapanel
```

## CLI Flags

| Flag                     | Description                                                |
| ------------------------ | ---------------------------------------------------------- |
| `--config`               | Path to dashboard config file (YAML). **Required**         |
| `--template-dir`         | Directory with template files. **Required**                |
| `--listen-address`       | HTTP listen address (default `:8080`)                      |
| `--api-token`            | Optional token for external APIs                           |
| `--jira-api-url`         | Jira REST base URL (`/rest/api/2` or `/3`). **Required**   |
| `--jira-email`           | Email for basic/cloud auth                                 |
| `--jira-token`           | API token or password (used with `--jira-email`)           |
| `--jira-bearer-token`    | Bearer token (self-hosted Jira alternative to email/token) |
| `--jira-skip-tls-verify` | Disable TLS verification (not recommended)                 |
| `--debug`                | Enable debug logging                                       |
| `--log-format`           | `text` or `json` (default: `text`)                         |

**Mutually exclusive:**

- `--jira-email` + `--jira-token`
- `--jira-bearer-token`

## Dashboard Config (`dashboard.yaml`)

```yaml
title: My Jira Dashboard
grid:
  columns: 2
  rows: 2
refreshInterval: 60
layout:
  - title: Open Issues
    query: filter=12345
    params:
      maxResults: "100"
    template: issues.gohtml
    position: { row: 0, col: 0 }

  - title: Assignee Stats
    query: filter=12345
    template: assignees.gohtml
    position: { row: 0, col: 1 }
```

## Templates

JiraPanel uses Go’s `html/template` engine with custom helpers and supports [Bootstrap](https://getbootstrap.com/) (v5.3.0) styling and [Tablesort.js](https://github.com/tristen/tablesort) (v5.6.0) for client-side sorting.

### 🔧 Built-in Template Functions

These helpers are registered for use inside your `.gohtml` templates:

- `default val def` — return `def` if `val` is zero value
- `add`, `list`, `listany`, `append`, `slice`
- `dict`, `set`, `keys`, `dig`
- `formatDate input layout` — format Jira timestamps, e.g.:

  ```gohtml
  {{ formatDate .fields.created "02.01.2006" }}
  ```

### 🎨 Styling and Behavior

- Use **Bootstrap 5** classes directly (via `bootstrap.min.css`)
- Tables with class `tablesort` are automatically sortable via **Tablesort.js**

Example:

```gohtml
<table class="table table-bordered table-hover tablesort">
  <thead>
    <tr><th>Name</th><th>Count</th></tr>
  </thead>
  <tbody>
    {{ range .Data }}
      <tr>
        <td>{{ .name }}</td>
        <td>{{ .count }}</td>
      </tr>
    {{ end }}
  </tbody>
</table>
```

### 📁 Example File

See [examples/templates/assignees.gohtml](examples/templates/assignees.gohtml) for a complete usage example:

```gohtml
<h2>{{ .Title }}</h2>
<table class="table table-bordered tablesort">
  <thead>
    <tr><th>Assignee</th><th>Issues</th></tr>
  </thead>
  <tbody>
    {{ range .Data.issues }}
      ...
    {{ end }}
  </tbody>
</table>
```

## Endpoints

| Method | Path        | Description     |
| ------ | ----------- | --------------- |
| GET    | `/`         | Dashboard view  |
| GET    | `/healthz`  | Health check    |
| POST   | `/healthz`  | Health check    |
| GET    | `/static/*` | JS, CSS, assets |

## Auto-Refresh

- Interval defined via `refreshInterval` in `dashboard.yaml`
- Exposed as `<meta name="refresh-interval" content="60">`
- JS reads and updates the reload interval dynamically
- Displayed in footer via `{{ .RefreshInterval }}`

## License

MIT License
