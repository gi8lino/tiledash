package templates

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"reflect"

	"github.com/gi8lino/tiledash/internal/config"
)

// RenderError is a generic error returned by RenderCell to surface UI-friendly failures.
type RenderError struct {
	Title   string
	Message string
	Detail  string
}

// NewRenderError creates a new RenderError.
func NewRenderError(typ, msg string, detail any) *RenderError {
	return &RenderError{
		Title:   typ,
		Message: msg,
		Detail:  fmt.Sprint(detail),
	}
}

// Error implements the error interface.
func (e *RenderError) Error() string {
	return fmt.Sprintf("%s: %s (%s)", e.Title, e.Message, e.Detail)
}

// RenderCell renders a single dashboard tile by index using pre-fetched data.
func RenderCell(
	ctx context.Context,
	id int,
	cfg config.DashboardConfig,
	tileTmpl *template.Template,
	data any, // []byte JSON, map[string]any payload, or accumulator {"merged":..., "pages":[...]}
) (template.HTML, *RenderError) {
	tile, err := cfg.GetCellByIndex(id)
	if err != nil {
		return "", NewRenderError("render", "Failed to get tile", err.Error())
	}

	primary, acc, raw, nerr := normalizeData(data)
	if nerr != nil {
		return "", NewRenderError("json", "Response could not be parsed", nerr.Error())
	}

	// Build template input (keep "Data" compatible with your existing templates).
	in := map[string]any{
		"ID":    id,
		"Title": tile.Title,
		"Data":  primary, // prefer merged or first page payload
		"Acc":   acc,     // optional full accumulator
		"Raw":   raw,     // original input for debugging
	}

	var buf bytes.Buffer
	if err := tileTmpl.ExecuteTemplate(&buf, tile.Template, in); err != nil {
		return "", NewRenderError("template", "Template rendering failed", err.Error())
	}

	return template.HTML(buf.String()), nil
}

// normalizeData converts arbitrary runner output into (primary, accumulator, raw, error).
// It unwraps named map types (e.g., providers.Accumulator) by reflecting to map[string]any,
// then prefers "merged" (if present) or the first page, falling back to the whole object.
func normalizeData(in any) (primary any, accumulator map[string]any, raw any, err error) {
	raw = in // preserve original input for optional debugging

	// If bytes: unmarshal to generic value, then recurse.
	if b, ok := in.([]byte); ok {
		var v any
		if uerr := json.Unmarshal(b, &v); uerr != nil {
			return nil, nil, raw, uerr
		}
		return normalizeData(v)
	}

	// Fast path: already a plain map[string]any.
	if m, ok := in.(map[string]any); ok {
		// Accumulator shape? Prefer "merged" payload if available.
		if hasKey(m, "merged") || hasKey(m, "pages") {
			if mv, _ := m["merged"].(map[string]any); len(mv) > 0 {
				return mv, m, raw, nil
			}
			// Try pages as []map[string]any
			if ps, _ := m["pages"].([]map[string]any); len(ps) > 0 {
				return ps[0], m, raw, nil
			}
			// Or pages as []any (mixed/unknown element types)
			if pa, _ := m["pages"].([]any); len(pa) > 0 {
				if pm, _ := pa[0].(map[string]any); pm != nil {
					return pm, m, raw, nil
				}
			}
			// Fallback to entire accumulator if we can't pick a primary.
			return m, m, raw, nil
		}
		// Single-page JSON object.
		return m, nil, raw, nil
	}

	// Robust path: reflect over any map with string keys (covers named map types).
	// Example: type Accumulator map[string]any won't satisfy in.(map[string]any).
	rv := reflect.ValueOf(in)
	if rv.IsValid() && rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
		// Copy to a plain map[string]any, then recurse to reuse logic above.
		mm := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			mm[iter.Key().String()] = iter.Value().Interface()
		}
		return normalizeData(mm)
	}

	// For other JSON-like shapes, pass through; else, normalize via a JSON round-trip.
	if rt := reflect.TypeOf(in); rt != nil {
		switch rt.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map, reflect.Struct:
			return in, nil, raw, nil
		}
	}

	// Last resort: JSON round-trip to coerce primitives/aliases.
	b, merr := json.Marshal(in)
	if merr != nil {
		return nil, nil, raw, merr
	}
	var v any
	if uerr := json.Unmarshal(b, &v); uerr != nil {
		return nil, nil, raw, uerr
	}
	return v, nil, raw, nil
}

// hasKey reports whether a map has a key, ignoring the value.
func hasKey(m map[string]any, k string) bool {
	_, ok := m[k]
	return ok
}
