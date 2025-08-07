# JiraPanel

**JiraPanel** is a flexible, self-hosted dashboard for visualizing data from your Jira Cloud or Server instance using templates and JQL queries.

## Features

- 📊 Multiple dashboard grid cells rendered from Go HTML templates
- 🧾 Query Jira issues using JQL or saved filters
- 🎯 1-based grid layout configuration
- 🔁 Auto-refresh support (configurable interval)
- 🧰 Simple CLI or environment variable setup
- 🧠 Debug mode for visualizing layout structure
- 🎨 Full visual customization via YAML

## 🚀 How It Works

JiraPanel renders a dynamic HTML dashboard by combining a **base layout template** (`base.gohtml`) with **per-cell content** fetched and rendered individually.

### Rendering Flow

1. **Dashboard page (`/`)** renders the grid and placeholders.
2. **JavaScript fetches each cell** from `/api/v1/cells/{id}`.
3. Server executes the cell's **JQL query** and renders the **associated template**.
4. If errors occur, a fallback template (`cell_error.gohtml`) is used.

Each cell is rendered independently, allowing fast and fault-tolerant dashboards.

## 📁 Configuration Overview

You configure the dashboard using:

- A `config.yaml` file
- `.gohtml` templates for each cell
- A CLI or environment flags for server setup

### 🧾 `config.yaml`

This is your main dashboard layout and data source file.

#### 🧱 Top-Level Keys

| Key               | Type     | Description                                  |
| ----------------- | -------- | -------------------------------------------- |
| `title`           | string   | Dashboard title (HTML page title and header) |
| `grid.columns`    | int      | Number of columns in the layout              |
| `grid.rows`       | int      | Number of rows in the layout                 |
| `refreshInterval` | duration | Auto-refresh interval (e.g., `60s`, `2m`)    |
| `cells`           | []cell   | List of grid cells (data cards)              |
| `customization`   | object   | Optional visual styles and layout settings   |

#### 🧱 `cells[]` Fields

| Field              | Type   | Description                                           |
| ------------------ | ------ | ----------------------------------------------------- |
| `title`            | string | Cell title (used in templates)                        |
| `query`            | string | JQL query or filter (e.g., `filter=12345`)            |
| `template`         | string | Template file name (must end with `.gohtml`)          |
| `position.row`     | int    | **1-based** row index of the cell (top to bottom)     |
| `position.col`     | int    | **1-based** column index of the cell (left to right)  |
| `position.colSpan` | int    | Number of columns to span (optional, defaults to `1`) |

#### 💡 Notes

- Grid positions (`row`, `col`) are **1-based** in YAML, but internally converted to 0-based.
- Templates must exist in `--template-dir` and be valid `.gohtml` files.
- Cells are rendered **in the order listed**.

#### 📄 Example

```yaml
title: My Jira Dashboard
grid:
  columns: 2
  rows: 3
refreshInterval: 60s
cells:
  - title: Epics
    query: filter=12345
    template: epics.gohtml
    position: { row: 1, col: 1 }

  - title: Open Issues
    query: filter=54321
    template: issues.gohtml
    position:
      row: 1
      col: 2

  - title: Grouped View
    query: filter=54321
    template: assignees.gohtml
    position:
      row: 2
      col: 1
      colSpan: 2
```

## 🎨 Customization

The `customization` block lets you tweak styling via CSS-like settings. If omitted, defaults are used.

| Key                    | Default                           | Description            |
| ---------------------- | --------------------------------- | ---------------------- |
| `grid.gap`             | `"2rem"`                          | Gap between cells      |
| `grid.padding`         | `"0rem"`                          | Padding inside grid    |
| `grid.marginTop`       | `"0rem"`                          | Space above grid       |
| `grid.marginBottom`    | `"0rem"`                          | Space below grid       |
| `card.borderColor`     | `"#ccc"`                          | Cell/card border color |
| `card.padding`         | `"0rem"`                          | Padding inside cells   |
| `card.backgroundColor` | `"#fff"`                          | Background color       |
| `card.borderRadius`    | `"0.5rem"`                        | Border radius          |
| `card.boxShadow`       | `"0 2px 4px rgba(0, 0, 0, 0.05)"` | Box shadow             |
| `header.align`         | `"left"`                          | `<h1>` alignment       |
| `header.marginBottom`  | `"0rem"`                          | Margin below header    |
| `footer.marginTop`     | `"1rem"`                          | Margin above footer    |
| `font.family`          | `"Segoe UI, sans-serif"`          | Font family            |
| `font.size`            | `"16px"`                          | Font size              |

## 🧩 Creating Custom Templates

Each cell renders a `.gohtml` file using the data returned by your Jira query.

### 1. Explore the Data

To see what fields you have access to:

```sh
curl -H "Authorization: Bearer YOUR_TOKEN" \
     -H "Accept: application/json" \
     "https://jira.example.com/rest/api/2/search?jql=filter=12345"
```

JiraPanel will expose the `.issues` array under `.Data`.

Example structure:

```json
{
  "issues": [
    {
      "fields": {
        "summary": "Some issue",
        "assignee": { "displayName": "Alice" },
        "status": { "name": "In Progress" }
      }
    }
  ]
}
```

### 2. Write a Template

Place it in your `--template-dir` with the `.gohtml` extension.

```gohtml
<h2>{{ .Title }}</h2>

{{ $issues := .Data.issues }}
<ul>
{{- range $i := $issues }}
  <li>{{ dig $i.fields "summary" }}</li>
{{- end }}
</ul>
```

> ⚠️ Why use `dig`?
>
> Jira's API responses often include deeply nested and dynamic fields, especially under `fields`. These values are usually returned as `map[string]interface{}` in Go — meaning you can’t access them directly like `.fields.summary` because Go templates don’t support type assertions.
>
> The `dig` function helps safely extract values from these maps as strings, like:
>
> ```gohtml
> {{ dig $i.fields "summary" }}
> ```
>
> This avoids crashes or empty output when accessing untyped or missing fields. If the field doesn’t exist or isn’t a string, `dig` returns an empty string instead of panicking.
>
> Use `dig` for anything under `.fields`, `.fields.customfield_*`, or other unpredictable structures returned by Jira.

All templates have access to:

- Standard Go `html/template`
- [Sprig functions](https://masterminds.github.io/sprig/)
- JiraPanel-specific helpers:

| Helper           | Signature                     | Description                                                                          | Example Usage                                        |
| :--------------- | :---------------------------- | :----------------------------------------------------------------------------------- | :--------------------------------------------------- |
| `setany`         | `setany map key value`        | Sets `map[key] = value` and returns the same map                                     | `{{ setany $m "key" "val" }}`                        |
| `dig`            | `dig map key`                 | Retrieves a string from a `map[string]any` by key; returns input if already a string | `{{ dig .fields "summary" }}`                        |
| `formatJiraDate` | `formatJiraDate input layout` | Parses Jira timestamp and formats it using Go layout                                 | `{{ formatJiraDate .fields.created "2006-01-02" }}`  |
| `appendSlice`    | `appendSlice slice item`      | Appends `item` to a `[]any` slice                                                    | `{{ $list := appendSlice $list $item }}`             |
| `sortBy`         | `sortBy field desc slice`     | Sorts a slice of `map[string]any` by field, descending if `desc` is true             | `{{ sortBy "count" true $entries }}`                 |
| `uniq`           | `uniq list`                   | Removes duplicate string values                                                      | `{{ uniq (list "a" "b" "a") }}` → `["a" "b"]`        |
| `defaultStr`     | `defaultStr value fallback`   | Returns `fallback` if `value` is empty or only whitespace                            | `{{ defaultStr .name "Unknown" }}`                   |
| `typeOf`         | `typeOf value`                | Returns the Go type of the given value                                               | `{{ typeOf .fields }}` → `"map[string]interface {}"` |
| `sumBy`          | `sumBy field slice`           | Sums the numeric values of the given field in a slice of `map[string]any`            | `{{ sumBy "count" $entries }}`                       |

You can also reuse logic from other templates using `{{ template "name" . }}` — great for status badges, labels, or error handling partials.

### 3. 📁 Browse Examples

See the [`examples/templates/`](examples/templates/) folder for more real-world templates, including:

- `assignees.gohtml` — count issues per assignee
- `env_issues.gohtml` — issue table with columns
- `epics.gohtml` — group by epic
- `functions.gohtml` — reusable helpers
- `issues.gohtml` — issue table with columns
- `podium.gohtml` — podium chart

With just YAML and `.gohtml` templates, you can build flexible, data-rich Jira dashboards tailored to your needs.

## 🐞 Debug Mode

Press `D` on the dashboard to:

- Show a red overlay with `row`, `col`, `colSpan`, and `template`
- Blur actual content for layout focus

Useful for spotting overlaps and grid misalignment.

## 📦 CLI Flags

| Flag                     | Description                               |
| ------------------------ | ----------------------------------------- |
| `--config`               | Path to `config.yaml` (**required**)      |
| `--template-dir`         | Path to template files (**required**)     |
| `--jira-api-url`         | Jira REST API base URL (**required**)     |
| `--jira-email`           | Email for basic/cloud auth                |
| `--jira-auth`            | API token or password (paired with email) |
| `--jira-bearer-token`    | Bearer token (alternative to email/token) |
| `--jira-skip-tls-verify` | Skip TLS verification (not recommended)   |
| `--listen-address`       | HTTP listen address (default `:8080`)     |
| `--debug`                | Enable debug logging                      |
| `--log-format`           | `text` or `json` (default: `text`)        |

### 🔐 Auth Methods

Use one of:

- `--jira-email` + `--jira-auth`
- `--jira-bearer-token`

## 🌐 Endpoints

| Path                 | Method | Description            |
| -------------------- | ------ | ---------------------- |
| `/`                  | GET    | Dashboard view         |
| `/api/v1/cells/{id}` | GET    | Render cell by ID      |
| `/healthz`           | GET    | Health check           |
| `/static/*`          | GET    | Static assets (JS/CSS) |

## 🧪 Local Dev + Deployment

Kubernetes manifests are available in `examples/kubernetes/`. Use `kustomize` to build ConfigMaps and deploy.

To render final YAML:

```sh
kustomize build examples/kubernetes
```

## 🪪 License

Apache 2.0. See `LICENSE`.
