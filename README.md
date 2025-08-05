# JiraPanel

**JiraPanel** is a flexible, self-hosted dashboard for visualizing data from your Jira Cloud or Server instance using templates and JQL queries.

## Features

- Render multiple dashboard grid cells with custom templates
- Query Jira issues using JQL or filters
- Fully configurable grid layout
- Auto-refresh support (configurable)
- Clean bootstrap-based layout
- Simple CLI configuration via flags or environment variables

## 🧠 How It Works

JiraPanel renders a dynamic dashboard by combining a **base layout template** with **per-cell content**, which is loaded asynchronously in the browser.

### Step-by-Step Flow:

1. **Base Template Load**
   The main dashboard view (`/`) renders `base.gohtml`, which defines the grid layout and includes placeholders for each cell.

2. **Asynchronous Cell Rendering**
   After the base page is loaded:

   - JavaScript fetches each individual cell via `/cells/{id}`
   - The server renders the cell using its configured template and Jira response data
   - The HTML is injected into the correct grid cell

3. **Server-Side Cell Rendering**
   For each `/cells/{id}` request:

   - A JQL query is executed against Jira
   - The JSON response is passed to a Go HTML template (e.g. `issues.gohtml`)
   - The rendered HTML is returned to the browser

4. **Error Handling**
   If any cell fails to render (e.g. bad query, timeout), the server renders a fallback using `cell_error.gohtml`. This ensures the rest of the dashboard remains usable.

### Benefits

- Only the base page is loaded initially
- Each cell loads independently, improving performance and fault tolerance
- Full HTML is rendered server-side — no client-side templating required

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
| :----------------------- | :--------------------------------------------------------- |
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

| **Key**           | **Type**   | **Description**                                                            |
| :---------------- | :--------- | :------------------------------------------------------------------------- |
| `title`           | `string`   | The title of the dashboard (displayed as page heading and `<title>`).      |
| `grid.columns`    | `int`      | Number of columns in the dashboard grid layout. Must be > 0.               |
| `grid.rows`       | `int`      | Number of rows in the dashboard grid layout. Must be > 0.                  |
| `refreshInterval` | `duration` | Interval for automatic dashboard refresh (e.g., `60s`, `2m`).              |
| `cells`           | `[]cell`   | List of dashboard cells/cards to display. Each one defines its own layout. |

### 🧱 **Cells Fields (`Cell[]`)**

| **Field**          | **Type**         | **Description**                                                               |
| :----------------- | :--------------- | :---------------------------------------------------------------------------- |
| `title`            | `string`         | Title for this cell/card (used in template rendering).                        |
| `query`            | `string`         | JQL query (e.g., `filter=12345`) to fetch Jira issues for this cell.          |
| `template`         | `string`         | Name of the Go HTML template used to render the cell (e.g., `issues.gohtml`). |
| `position.row`     | `int`            | Zero-based row index of the cells starting position.                          |
| `position.col`     | `int`            | Zero-based column index of the cells starting position.                       |
| `position.colSpan` | `int` (optional) | Number of columns this cell spans. Defaults to 1 if omitted or 0.             |

### 💡 Notes

- **`position.colSpan`** must not exceed `grid.columns` when added to `col`. For example:
  If `col: 1` and `colSpan: 2`, this overflows a 2-column grid.
- **Templates** must exist in the specified `templateDir` and be named exactly as listed.
- **Section order** in `cell[]` affects rendering order — no automatic sorting is done.

### 📁 Example Config

```yaml
---
title: My Jira Dashboard
grid:
  columns: 2
  rows: 4
cells:
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

### 🎨 Customization (`customization` block)

You can optionally fine-tune the look and feel of the dashboard using the `customization` section in your `config.yaml`. If omitted, sensible defaults are used.

| Key                                  | Type     | Description                       | Default                           |
| :----------------------------------- | :------- | :-------------------------------- | :-------------------------------- |
| `customization.grid.gap`             | `string` | Spacing between grid cells        | `"2rem"`                          |
| `customization.grid.padding`         | `string` | Padding inside the grid container | `"0rem"`                          |
| `customization.grid.marginTop`       | `string` | Space above the grid              | `"0rem"`                          |
| `customization.grid.marginBottom`    | `string` | Space below the grid              | `"0rem"`                          |
| `customization.card.borderColor`     | `string` | Card border color                 | `"#ccc"`                          |
| `customization.card.padding`         | `string` | Padding inside cards              | `"0rem"`                          |
| `customization.card.backgroundColor` | `string` | Card background color             | `"#fff"`                          |
| `customization.card.borderRadius`    | `string` | Card border radius                | `"0.5rem"`                        |
| `customization.card.boxShadow`       | `string` | Card shadow                       | `"0 2px 4px rgba(0, 0, 0, 0.05)"` |
| `customization.header.align`         | `string` | Alignment of the `<h1>` title     | `"left"`                          |
| `customization.header.marginBottom`  | `string` | Space below the header            | `"0rem"`                          |
| `customization.footer.marginTop`     | `string` | Space above the footer            | `"1rem"`                          |
| `customization.font.family`          | `string` | Font family for the dashboard     | `"Segoe UI", sans-serif`          |
| `customization.font.size`            | `string` | Font size (any valid CSS size)    | `"16px"`                          |

All values are directly injected into the rendered HTML/CSS. You can use any valid CSS value (e.g. `px`, `rem`, `%`, color codes).

#### Example

```yaml
customization:
  grid:
    gap: 1rem
    padding: 1.5rem
    marginTop: 2rem
    marginBottom: 2rem
  card:
    borderColor: "#ddd"
    padding: 1rem
    backgroundColor: "#fff"
    borderRadius: 0.5rem
    boxShadow: 0 2px 4px rgba(0, 0, 0, 0.05)
  header:
    align: center
    marginBottom: 1rem
  footer:
    marginTop: 1rem
  font:
    family: "Segoe UI, sans-serif"
    size: 16px
```

> ⚙️ If `customization` is omitted or incomplete, defaults are automatically filled in.

## 🧩 Creating a New Section Template

To visualize custom data from Jira, you can create **your own `.gohtml` cell templates** using standard Go templates with helpers.

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

Each dashboard **cell** corresponds to a `.gohtml` file in your `--template-dir`.

Here’s a simple example that **groups issues by assignee and counts them**:

```gohtml
<h2>{{ .Title }}</h2> <!-- Render the cell title -->

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
      <td>{{ $total }}</td> <!-- Total issues in the cell -->
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

They must exist inside the directory specified via `--template-dir`. If a cell template listed in your `config.yaml` is missing or malformed, the dashboard will fail to render and display an error.

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
| :----- | :---------- | :-------------- |
| GET    | `/`         | Dashboard view  |
| GET    | `/healthz`  | Health check    |
| POST   | `/healthz`  | Health check    |
| GET    | `/static/*` | JS, CSS, assets |

## Auto-Refresh

- Interval defined via `refreshInterval` in `config.yaml`
- Exposed as `<meta name="refresh-interval" content="60">`
- JS reads and updates the reload interval dynamically
- Displayed in footer via `{{ .RefreshInterval }}`

### 🧩 Kubernetes Deployment via Kustomize

This directory provides a ready-to-deploy Kubernetes setup using [`kustomize`](https://kubectl.docs.kubernetes.io/).

It generates a `Deployment` that bundles:

- Your `config.yaml` application configuration
- All `.gohtml` templates
- The core app manifests (`Deployment`, `Service`, `Ingress`, etc.)

#### 📁 Structure

```bash
examples/
├── config.yaml                    # Application configuration
├── templates/                     # Go HTML templates for the dashboard
├── kubernetes/
│   ├── deployment.yaml            # App deployment spec
│   ├── service.yaml               # Exposes the app
│   ├── ingress.yaml               # Optional ingress (if enabled)
│   ├── secret.yaml                # Add your secrets here
│   ├── namespace.yaml             # Defines the app namespace
│   ├── kustomization.yaml         # Kustomize entry point
```

#### ⚙️ How it works

- `kustomization.yaml` uses `configMapGenerator` to package:

  - `../config.yaml` → as `ConfigMap/jirapanel-config`
  - all templates → as `ConfigMap/jirapanel-templates`

- These ConfigMaps are mounted into the container at runtime.

Kustomize automatically appends a **hash suffix** to the ConfigMap names (e.g. `jirapanel-config-fd8d7f97b9`) when their content changes. The `Deployment` references them by logical name (`jirapanel-config`, `jirapanel-templates`), and Kustomize resolves the hashed names at build time.
This has the advantage that **when you update the config or templates, the hash changes, triggering a rollout restart of the container** — ensuring your app always runs with the latest configuration.

#### 🚀 Deploy locally

You can preview the full manifest with:

```bash
kustomize build examples/kubernetes
```

Or apply directly:

```bash
kubectl apply -k examples/kubernetes
```

#### 🔧 Disabling Hashing (for local testing)

To disable content-based hashes on ConfigMaps (e.g., for stable volume mounts during development), set:

```yaml
generatorOptions:
  disableNameSuffixHash: true
```

in `examples/kubernetes/kustomization.yaml`.

> ⚠️ It's recommended to **keep the hash enabled** in production for safe config rollouts.

## License

This project is licensed under the Apache 2.0 License. See the [LICENSE](LICENSE) file for details.
