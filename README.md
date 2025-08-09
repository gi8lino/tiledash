# JiraPanel

[![Go Report Card](https://goreportcard.com/badge/github.com/gi8lino/jira-panel?style=flat-square)](https://goreportcard.com/report/github.com/gi8lino/jira-panel)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/gi8lino/jira-panel)
[![Release](https://img.shields.io/github/release/gi8lino/jira-panel.svg?style=flat-square)](https://github.com/gi8lino/jira-panel/releases/latest)
[![GitHub tag](https://img.shields.io/github/tag/gi8lino/jira-panel.svg?style=flat-square)](https://github.com/gi8lino/jira-panel/releases/latest)
![Tests](https://github.com/gi8lino/jira-panel/actions/workflows/tests.yml/badge.svg)
[![Build](https://github.com/gi8lino/jira-panel/actions/workflows/release.yml/badge.svg)](https://github.com/gi8lino/jira-panel/actions/workflows/release.yml)
[![license](https://img.shields.io/github/license/gi8lino/jira-panel.svg?style=flat-square)](LICENSE)

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
| :---------------- | :------- | :------------------------------------------- |
| `title`           | string   | Dashboard title (HTML page title and header) |
| `grid.columns`    | int      | Number of columns in the layout              |
| `grid.rows`       | int      | Number of rows in the layout                 |
| `refreshInterval` | duration | Auto-refresh interval (e.g., `60s`, `2m`)    |
| `cells`           | []cell   | List of grid cells (data cards)              |
| `customization`   | object   | Optional visual styles and layout settings   |

#### 🧱 `cells[]` Fields

| Field              | Type   | Description                                           |
| :----------------- | :----- | :---------------------------------------------------- |
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
| :--------------------- | :-------------------------------- | :--------------------- |
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

Place your `.gohtml` file in the directory passed via `--template-dir`.

Each template has access to the following:

- `.Title` - the cell title from your config
- `.Data` - the parsed Jira API response
- `.ID` - the 0-based index of the cell in your dashboard layout (e.g. `cell-0`, `cell-1`, `cell-2`, ...)

> ⚠️ **You can now use `.ID` to target the card container** via:
>
> - `id="cell-{{ .ID }}"` (already applied in `base.gohtml`)
> - JS: `document.getElementById("cell-{{ .ID }}")`
> - CSS: `#cell-{{ .ID }} { display: none; }`
>   see [examples/templates/env_alert.gohtml](examples/templates/env_alert.gohtml)

> **Template Visibility**
>
> A template can hide its containing card by rendering a marker element:
>
> ```html
> <div data-jp-hidden></div>
> ```
>
> JiraPanel's JS will hide the card when this marker is present. Use this for "empty/quiet" states (e.g., no alerts). This avoids layout flicker and doesn't require special client-side rules.

### 🛠 Example Template

This template shows a list of issues with their summary.

```gohtml
<h2>{{ .Title }}</h2>

{{ $issues := .Data.issues }}
<ul>
{{- range $i := $issues }}
  <li>{{ dig $i.fields "summary" }}</li>
{{- end }}
</ul>
```

### 🛠 Why Use `dig`?

Jira's API returns deeply nested and dynamic fields, especially under `.fields`. These values are usually `map[string]interface{}` in Go - meaning you can't access them directly like `.fields.summary`.

The `dig` function safely extracts values from maps as strings:

```gohtml
{{ dig $issue.fields "summary" }}
```

If the field doesn't exist or isn't a string, it returns an empty string instead of crashing.

Use `dig` for anything under `.fields`, `.fields.customfield_*`, or other unpredictable Jira fields.

### 🧰 Built-in Helpers

| Helper           | Signature                     | Description                                                            | Example Usage                                       |
| :--------------- | :---------------------------- | :--------------------------------------------------------------------- | :-------------------------------------------------- |
| `setany`         | `setany map key value`        | Sets `map[key] = value` and returns the map                            | `{{ setany $m "key" "val" }}`                       |
| `dig`            | `dig map key`                 | Extracts a string from a `map[string]any`; safe for nested Jira fields | `{{ dig .fields "summary" }}`                       |
| `formatJiraDate` | `formatJiraDate input layout` | Formats Jira timestamps using Go layouts                               | `{{ formatJiraDate .fields.created "2006-01-02" }}` |
| `appendSlice`    | `appendSlice slice item`      | Appends `item` to a slice                                              | `{{ $list := appendSlice $list $item }}`            |
| `sortBy`         | `sortBy field desc slice`     | Sorts a slice of maps by field name                                    | `{{ sortBy "count" true $entries }}`                |
| `uniq`           | `uniq list`                   | Removes duplicate strings                                              | `{{ uniq (list "a" "b" "a") }}` → `["a", "b"]`      |
| `defaultStr`     | `defaultStr value fallback`   | Fallback if `value` is empty or whitespace                             | `{{ defaultStr .name "Unknown" }}`                  |
| `typeOf`         | `typeOf value`                | Returns the Go type of the input value                                 | `{{ typeOf .fields }}`                              |
| `sumBy`          | `sumBy field slice`           | Sums numeric fields across a slice of maps                             | `{{ sumBy "count" $entries }}`                      |

You can also reuse logic from other templates using `{{ template "name" . }}` - great for status badges, labels, or error handling partials.

### 3. 📁 Browse Examples

See the [`examples/templates/`](examples/templates/) folder for more real-world templates, including:

- `assignees.gohtml` - count issues per assignee
- `env_issues.gohtml` - issue table with columns
- `epics.gohtml` - group by epic
- `functions.gohtml` - reusable helpers
- `issues.gohtml` - issue table with columns
- `podium.gohtml` - podium chart

With just YAML and `.gohtml` templates, you can build flexible, data-rich Jira dashboards tailored to your needs.

## 🐞 Debug Mode

Press `D` on the dashboard to:

- Show a red overlay with `row`, `col`, `colSpan`, and `template`
- Blur actual content for layout focus

Useful for spotting overlaps and grid misalignment.

## 📦 CLI Flags

| Flag                     | Description                               |
| :----------------------- | :---------------------------------------- |
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
| :------------------- | :----- | :--------------------- |
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
