package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

// v is the package-wide validator instance. Initialized once in init().
var v *validator.Validate

func init() {
	v = validator.New(validator.WithRequiredStructEnabled())
	must(v.RegisterValidation("abspath", isAbsPath))
	must(v.RegisterValidation("containsPath", containsPath))
	must(v.RegisterValidation("containsName", containsName))
	v.RegisterStructValidation(uniqueGroupNamesValidator, Config{})
}

// Validate reports whether c satisfies all schema constraints. On failure,
// returns a ValidationErrors aggregating every issue (one entry per failed
// field). Returns nil on success.
func (c *Config) Validate() error {
	err := v.Struct(c)
	if err == nil {
		return nil
	}
	if ferrs, ok := err.(validator.ValidationErrors); ok {
		return translate(ferrs)
	}
	return err
}

// isAbsPath checks the abspath validate tag. Tilde expansion happens in
// Load before Validate, so by the time Validate sees a Repo path it must
// already start with "/".
func isAbsPath(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	return strings.HasPrefix(s, "/")
}

// containsPath checks that a []string contains at least one element with
// the literal "{path}" placeholder. Used by AddDirArgs.
func containsPath(fl validator.FieldLevel) bool {
	xs := fl.Field().Interface().([]string)
	for _, s := range xs {
		if strings.Contains(s, "{path}") {
			return true
		}
	}
	return false
}

// containsName checks that a string contains the literal "{name}"
// placeholder. Used by WorktreePathPattern.
func containsName(fl validator.FieldLevel) bool {
	return strings.Contains(fl.Field().String(), "{name}")
}

// ValidationError is a single field-level validation failure with a
// JSON-style field path and the failed validator tag.
type ValidationError struct {
	Path string // e.g. "groups[2].repos[0]"
	Tag  string // e.g. "min", "abspath", "containsName", "unique"
	Msg  string // human-readable, suitable for end-user display
}

// Error implements the error interface.
func (e ValidationError) Error() string {
	if e.Path == "" {
		return e.Msg
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Msg)
}

// ValidationErrors is a list of ValidationError. The zero value is empty.
type ValidationErrors []ValidationError

// Error implements the error interface — one issue per line.
func (errs ValidationErrors) Error() string {
	parts := make([]string, len(errs))
	for i, e := range errs {
		parts[i] = e.Error()
	}
	return strings.Join(parts, "\n")
}

// Unwrap returns the individual entries so errors.Is/errors.As can match
// against any of them.
func (errs ValidationErrors) Unwrap() []error {
	out := make([]error, len(errs))
	for i, e := range errs {
		out[i] = e
	}
	return out
}

// translate converts validator.ValidationErrors into our typed ValidationErrors.
// Returns nil when ferrs is empty.
func translate(ferrs validator.ValidationErrors) ValidationErrors {
	if len(ferrs) == 0 {
		return nil
	}
	out := make(ValidationErrors, len(ferrs))
	for i, fe := range ferrs {
		out[i] = ValidationError{
			Path: jsonPath(fe.Namespace()),
			Tag:  fe.Tag(),
			Msg:  message(fe),
		}
	}
	return out
}

// jsonPath turns the validator namespace ("Config.Groups[2].Repos") into the
// JSON-style path ("groups[2].repos") by mapping each segment through the
// json struct tag. The first segment ("Config") is dropped since the
// top-level object has no JSON name.
func jsonPath(ns string) string {
	if ns == "" {
		return ""
	}
	segs := strings.Split(ns, ".")
	if len(segs) <= 1 {
		return ""
	}
	out := make([]string, 0, len(segs)-1)
	cur := reflect.TypeOf(Config{})
	for _, seg := range segs[1:] {
		name, idx := splitIndex(seg)
		f, ok := fieldByName(cur, name)
		if !ok {
			out = append(out, strings.ToLower(seg))
			continue
		}
		jsonName := jsonTagName(f)
		if idx >= 0 {
			out = append(out, fmt.Sprintf("%s[%d]", jsonName, idx))
		} else {
			out = append(out, jsonName)
		}
		// Descend into the field type for the next segment.
		ft := f.Type
		if ft.Kind() == reflect.Slice || ft.Kind() == reflect.Array {
			ft = ft.Elem()
		}
		cur = ft
	}
	return strings.Join(out, ".")
}

// splitIndex parses "Repos[0]" into ("Repos", 0). Returns idx == -1 when
// there's no bracket.
func splitIndex(s string) (string, int) {
	open := strings.IndexByte(s, '[')
	if open < 0 {
		return s, -1
	}
	close := strings.IndexByte(s, ']')
	if close < 0 || close < open {
		return s, -1
	}
	idx, err := strconv.Atoi(s[open+1 : close])
	if err != nil {
		return s[:open], -1
	}
	return s[:open], idx
}

func fieldByName(t reflect.Type, name string) (reflect.StructField, bool) {
	if t.Kind() != reflect.Struct {
		return reflect.StructField{}, false
	}
	return t.FieldByName(name)
}

// jsonTagName returns the JSON name for f, falling back to lower-cased
// field name.
func jsonTagName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" || tag == "-" {
		return strings.ToLower(f.Name)
	}
	if i := strings.IndexByte(tag, ','); i >= 0 {
		tag = tag[:i]
	}
	if tag == "" {
		return strings.ToLower(f.Name)
	}
	return tag
}

// message returns a human-readable explanation for a validator field error.
func message(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "min":
		return fmt.Sprintf("must contain at least %s entries", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fe.Param())
	case "abspath":
		return "must be an absolute path (tilde-expanded by loader before validation)"
	case "containsPath":
		return "must contain at least one entry with {path}"
	case "containsName":
		return "must contain {name}"
	case "unique":
		return fmt.Sprintf("duplicate value: %v", fe.Value())
	default:
		return fmt.Sprintf("failed %s validation", fe.Tag())
	}
}

// uniqueGroupNamesValidator reports a "unique" failure on every duplicate
// group name beyond the first occurrence. Mirrors the TS superRefine logic.
// The reported namespace uses an indexed field path (Groups[i].Name) so the
// downstream path-formatter (Task 7) produces a clean groups[i].name path.
func uniqueGroupNamesValidator(sl validator.StructLevel) {
	cfg := sl.Current().Interface().(Config)
	seen := make(map[string]struct{}, len(cfg.Groups))
	for i, g := range cfg.Groups {
		if _, dup := seen[g.Name]; dup {
			sl.ReportError(cfg.Groups[i].Name, fmt.Sprintf("Groups[%d].Name", i), "Name", "unique", "")
		}
		seen[g.Name] = struct{}{}
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
