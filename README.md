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
  --config config.yaml \
  --template-dir templates \
  --jira-api-url https://yourcompany.atlassian.net/rest/api/3 \
  --jira-email alice@yourcompany.com \
  --jira-auth xxxx
```

Or using environment variables:

```sh
JIRAPANEL_CONFIG=config.yaml \
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
| `--jira-auth`            | API token or password (used with `--jira-email`)           |
| `--jira-bearer-token`    | Bearer token (self-hosted Jira alternative to email/token) |
| `--jira-skip-tls-verify` | Disable TLS verification (not recommended)                 |
| `--debug`                | Enable debug logging                                       |
| `--log-format`           | `text` or `json` (default: `text`)                         |

### Proxy Environment Variables

- `HTTP_PROXY`: URL of the proxy server to use for HTTP requests
- `HTTPS_PROXY`: URL of the proxy server to use for HTTPS requests

**Groups:**

- `--jira-email` + `--jira-auth`
  or
- `--jira-bearer-token`

## Dashboard Config (`config.yaml`)

### 🧾 **Dashboard Config Reference**

| **Key**           | **Type**    | **Description**                                                               |
| ----------------- | ----------- | ----------------------------------------------------------------------------- |
| `title`           | `string`    | The title of the dashboard (displayed as page heading and `<title>`).         |
| `grid.columns`    | `int`       | Number of columns in the dashboard grid layout. Must be > 0.                  |
| `grid.rows`       | `int`       | Number of rows in the dashboard grid layout. Must be > 0.                     |
| `refreshInterval` | `duration`  | Interval for automatic dashboard refresh (e.g., `60s`, `2m`).                 |
| `layout`          | `[]section` | List of dashboard sections/cards to display. Each one defines its own layout. |

### 🧱 **Section Fields (`layout[]`)**

| **Field**          | **Type**         | **Description**                                                                  |
| ------------------ | ---------------- | -------------------------------------------------------------------------------- |
| `title`            | `string`         | Title for this section/card (used in template rendering).                        |
| `query`            | `string`         | JQL query (e.g., `filter=12345`) to fetch Jira issues for this section.          |
| `template`         | `string`         | Name of the Go HTML template used to render the section (e.g., `issues.gohtml`). |
| `position.row`     | `int`            | Zero-based row index of the section's starting position.                         |
| `position.col`     | `int`            | Zero-based column index of the section's starting position.                      |
| `position.colSpan` | `int` (optional) | Number of columns this section spans. Defaults to 1 if omitted or 0.             |

### 💡 Notes

- **`position.colSpan`** must not exceed `grid.columns` when added to `col`. For example:
  If `col: 1` and `colSpan: 2`, this overflows a 2-column grid.
- **Templates** must exist in the specified `templateDir` and be named exactly as listed.
- **Section order** in `layout[]` affects rendering order — no automatic sorting is done.

### 📁 Example Config

```yaml
---
title: My Jira Dashboard
grid:
  columns: 2
  rows: 4
layout:
  - title: Env Epics
    query: filter=17201
    template: epics.gohtml
    position:
      row: 0 # row 0 is the first row
      col: 0 # col 0 is the first column
  - title: Open Environment Issues
    query: filter=17203
    template: issues.gohtml
    position: { row: 0, col: 1 }
  - title: Two Dimensional Open Environment Issues
    query: filter=17203
    template: env_issues.gohtml
    position:
      row: 1
      col: 1
  - title: "Issue Statistics: Open Environment Issues (Assignee)"
    query: filter=17203
    template: assignees.gohtml
    position:
      row: 3
      col: 0
      colSpan: 2
refreshInterval: 60s
```

## 🧩 Creating a New Section Template

To visualize custom data from Jira, you can create **your own `.gohtml` section templates** using standard Go templates with helpers.

### 1. 🔍 Fetch Real Data to Explore

Use `curl` to preview the **raw Jira issue data** your template will receive:

```sh
curl -s \
  -H "Authorization: Bearer YOUR_API_TOKEN" \
  -H "Accept: application/json" \
  "https://jira.example.com/rest/api/2/search?jql=filter=17203" \
  > issues.json
```

The response will be a JSON object with this structure:

```json
{
  "issues": [
    {
      "fields": {
        "summary": "Issue summary here",
        "assignee": {
          "displayName": "Alice Example"
        },
        "status": {
          "name": "In Progress"
        },
        "components": [
          { "name": "API" }
        ]
      }
    },
    ...
  ]
}
```

Your template will access this via `.Data.issues`, so for each issue, you can use:

- `.fields.summary`
- `.fields.assignee.displayName`
- `.fields.status.name`
- `.fields.components`, etc.

### 2. 🧱 Write a template based on that Structure

Each dashboard **section** corresponds to a `.gohtml` file in your `--template-dir`.

Here’s a simple example that **groups issues by assignee and counts them**:

```gohtml
<h2>{{ .Title }}</h2> <!-- Render the section title -->

{{- $data := .Data.issues }} <!-- Assign issues list to a local variable -->
{{- $assignees := dict }}    <!-- Create an empty map to count issues per assignee -->
{{- $total := 0 }}           <!-- Counter for total number of issues -->

{{/* Group and count issues per assignee */}}
{{- range $issue := $data }}
  {{- $assignee := "Unassigned" }} <!-- Default label if no assignee -->
  {{- with $issue.fields.assignee.displayName }}
    {{- $assignee = . }} <!-- Use display name if present -->
  {{- end }}
  {{- $count := index $assignees $assignee }} <!-- Get current count for this assignee -->
  {{- if not $count }}
    {{- $_ := set $assignees $assignee 1 }} <!-- First occurrence -->
  {{- else }}
    {{- $_ := set $assignees $assignee (add $count 1) }} <!-- Increment count -->
  {{- end }}
  {{- $total = add $total 1 }} <!-- Increment total count -->
{{- end }}

<table class="table table-bordered table-hover align-middle text-left">
  <thead class="table-dark">
    <tr>
      <th>Assignee</th>
      <th>Count</th>
    </tr>
  </thead>
  <tbody>
    {{- range $name, $count := $assignees }}
      <tr>
        <td>{{ $name }}</td> <!-- Assignee name -->
        <td>{{ $count }}</td> <!-- Number of issues assigned -->
      </tr>
    {{- end }}
    <tr class="table-secondary fw-bold">
      <td>Total Issues</td>
      <td>{{ $total }}</td> <!-- Total issues in the section -->
    </tr>
  </tbody>
</table>
```

This template is minimal but still demonstrates:

- Data extraction (`.Data.issues`)
- Safe access to fields
- Aggregation by key (`assignee`)
- Total summing
- Clean HTML table output

### 3. 🧠 Template Helpers

All templates have access to:

- **[Sprig functions](https://masterminds.github.io/sprig/)** like `dict`, `list`, `add`, `len`, `slice`, `date`, etc.
- **Custom helpers** like:

  - `set`, `setany`, `dig`
  - `formatJiraDate`

You can also define reusable logic in a separate `.gohtml` and use `{{ template "name" . }}` to include it.

### 4. 📁 Browse Examples

See the [`examples/templates/`](examples/templates/) folder for more real-world templates, including:

- `assignees.gohtml` — count issues per assignee
- `epics.gohtml` — group by epic
- `env_issues.gohtml` — issue table with columns
- `functions.gohtml` — reusable helpers

With just YAML and `.gohtml` templates, you can build flexible, data-rich Jira dashboards tailored to your needs.

## Templates

JiraPanel uses Go’s `html/template` engine with custom helpers and supports [Bootstrap](https://getbootstrap.com/) (v5.3.0) styling and [Tablesort.js](https://github.com/tristen/tablesort) (v5.6.0) for client-side sorting.

### 📁 Section Templates

> **Important:** Section templates **must end with `.gohtml`**. For example: `epics.gohtml`, `issues.gohtml`.

They must exist inside the directory specified via `--template-dir`. If a section template listed in your `config.yaml` is missing or malformed, the dashboard will fail to render and display an error.

### 🔧 Built-in Template Functions

These helpers are available inside your `.gohtml` templates:

#### 🧮 Data Manipulation

- `add`, `list`, `append`, `slice`, `dict`, `keys`
  Standard utilities from [Sprig](https://masterminds.github.io/sprig/).

#### 🗺 Dictionary Helpers

- `setany m key val` — set a key-value pair in a `map[string]any`, modifying it in place.

  ```gohtml
  {{ $_ := setany $myMap "key" "value" }}
  ```

- `dig m key` — safely extract a string value from a `map[string]any` or return a string directly.

  ```gohtml
  {{ dig .fields "summary" }}
  ```

#### 🕒 Jira-Specific

- `formatJiraDate input layout` — parse and format Jira timestamps.

  ```gohtml
  {{ formatJiraDate .fields.created "02.01.2006" }}
  ```

> Note: `formatJiraDate` handles Jira's timezone format (`Z` → `+0000`), falling back to raw input if parsing fails.

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

## Endpoints

| Method | Path        | Description     |
| ------ | ----------- | --------------- |
| GET    | `/`         | Dashboard view  |
| GET    | `/healthz`  | Health check    |
| POST   | `/healthz`  | Health check    |
| GET    | `/static/*` | JS, CSS, assets |

## Auto-Refresh

- Interval defined via `refreshInterval` in `config.yaml`
- Exposed as `<meta name="refresh-interval" content="60">`
- JS reads and updates the reload interval dynamically
- Displayed in footer via `{{ .RefreshInterval }}`

## License

This project is licensed under the Apache 2.0 License. See the [LICENSE](LICENSE) file for details.
