package toolx

import (
	"strconv"

	"github.com/oapi-codegen/nullable"
)

// Ptr returns a pointer to a copy of v (for value -> *T).
func Ptr[T any](v T) *T { return &v }

// PtrIfNonZero returns a pointer to v if v != 0, otherwise nil.
func PtrIfNonZero[T ~int | ~int32 | ~int64](v T) *T {
	if v == 0 {
		return nil
	}
	return &v
}

// PtrIf returns a pointer to v if v != zero, otherwise nil.
// T must be comparable (e.g. not slices, maps, funcs).
// Examples:
// PtrIf("", "")         -> nil
// PtrIf("x", "")        -> *"x"
// PtrIf(int64(0), int64(0)) -> nil
func PtrIf[T comparable](v, zero T) *T {
	if v == zero {
		return nil
	}
	return &v
}

// MapSlice maps []S -> []D using f.
func MapSlice[S any, D any](in []S, f func(S) D) []D {
	if in == nil {
		return nil
	}
	out := make([]D, len(in))
	for i := range in {
		out[i] = f(in[i])
	}
	return out
}

// MapNullable maps a nullable.Nullable[T] to *U using f.
// Returns nil when the field is unspecified OR explicitly null.
// Propagates Get() errors from the nullable package.
func MapNullable[T, U any](n nullable.Nullable[T], f func(T) U) (*U, error) {
	if !n.IsSpecified() || n.IsNull() {
		return nil, nil
	}
	v, err := n.Get()
	if err != nil {
		return nil, err
	}
	u := f(v)
	return &u, nil
}

// MapNullableV maps a nullable field to a value using f; returns def for unspecified or null.
func MapNullableV[T, U any](n nullable.Nullable[T], f func(T) U, def U) (U, error) {
	if !n.IsSpecified() || n.IsNull() {
		return def, nil
	}
	v, err := n.Get()
	if err != nil {
		return def, err
	}
	return f(v), nil
}

// MapNullableSliceV maps a nullable slice to a []D; returns nil (or empty) for unspecified/null.
func MapNullableSliceV[S, D any](n nullable.Nullable[[]S], f func(S) D) ([]D, error) {
	if !n.IsSpecified() || n.IsNull() {
		return []D{}, nil
	}
	in, err := n.Get()
	if err != nil {
		return nil, err
	}
	out := make([]D, len(in))
	for i := range in {
		out[i] = f(in[i])
	}
	return out, nil
}

// MapNullableSlice maps a nullable.Nullable[[]S] to *[]D using f for elements.
// Returns nil when the field is unspecified OR explicitly null.
func MapNullableSlice[S, D any](n nullable.Nullable[[]S], f func(S) D) (*[]D, error) {
	if !n.IsSpecified() || n.IsNull() {
		return nil, nil
	}
	in, err := n.Get()
	if err != nil {
		return nil, err
	}
	out := make([]D, len(in))
	for i := range in {
		out[i] = f(in[i])
	}
	return &out, nil
}

// CopyPtr returns a new pointer with the same value, or nil if input is nil.
func CopyPtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

// MapPtr applies f to *S and returns *D (nil-safe).
func MapPtr[S, D any](p *S, f func(S) D) *D {
	if p == nil {
		return nil
	}
	v := f(*p)
	return &v
}

// Deref returns the value pointed to by p, or def if p is nil.
func Deref[T any](p *T, def T) T {
	if p == nil {
		return def
	}
	return *p
}

// DerefSlice returns a copy of the slice pointed to by p, or nil if p is nil.
func DerefSlice[T any](p *[]T) []T {
	if p == nil {
		return nil
	}
	out := make([]T, len(*p))
	copy(out, *p)
	return out
}

// DerefSlicePtr maps *[]S -> []D
func DerefSlicePtr[S any, D any](p *[]S, f func(S) D) []D {
	if p == nil {
		return nil
	}
	out := make([]D, len(*p))
	for i, v := range *p {
		out[i] = f(v)
	}
	return out
}

// DerefMap pointer to value using mapper and default
func DerefMap[S any, D any](p *S, f func(S) D, def D) D {
	if p == nil {
		return def
	}
	return f(*p)
}

// DerefSlicePtrE maps *[]S -> []D using f(S) (D, error).
// Returns nil, nil if p is nil.
func DerefSlicePtrE[S any, D any](p *[]S, f func(S) (D, error)) ([]D, error) {
	if p == nil {
		return nil, nil
	}
	in := *p
	out := make([]D, len(in))
	for i := range in {
		d, err := f(in[i])
		if err != nil {
			return nil, err
		}
		out[i] = d
	}
	return out, nil
}

// Int64PtrToStringPtr maps *int64 -> *string using strconv.FormatInt. Returns nil if input is nil.
func Int64PtrToStringPtr(p *int64) *string {
	return MapPtr(p, func(v int64) string {
		return strconv.FormatInt(v, 10)
	})
}

// Int64PtrToString converts *int64 → string, returns "" if nil.
func Int64PtrToString(p *int64) string {
	return DerefMap(p, func(v int64) string {
		return strconv.FormatInt(v, 10)
	}, "")
}

// StringPtrToInt64 converts *string → int64 (0 if nil), error if parsing fails.
func StringPtrToInt64(p *string) (int64, error) {
	if p == nil {
		return 0, nil
	}
	return strconv.ParseInt(*p, 10, 64)
}

// StringToInt64 converts string → int64, returns error if not parsable.
func StringToInt64(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseInt(s, 10, 64)
}

// StringToInt64Ptr converts string → *int64, returns nil if empty, error if invalid.
func StringToInt64Ptr(s string) (*int64, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func StringToInt32Ptr(s string) (*int32, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return nil, err
	}
	x := int32(v)
	return &x, nil
}

// StringPtrToInt64Ptr converts *string → *int64. Returns nil if input is nil, error if parsing fails.
func StringPtrToInt64Ptr(p *string) (*int64, error) {
	if p == nil {
		return nil, nil
	}
	v, err := strconv.ParseInt(*p, 10, 64)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// Int64ToString converts int64 → string.
func Int64ToString(v int64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatInt(v, 10)
}

// MapMap maps a map’s values to a new map with the same keys.
func MapMap[K comparable, S any, D any](in map[K]S, f func(S) D) map[K]D {
	if in == nil {
		return nil
	}
	out := make(map[K]D, len(in))
	for k, v := range in {
		out[k] = f(v)
	}
	return out
}

// PtrIfNonEmptyMap returns *map when m has at least one entry, otherwise nil.
// The pointer is to a copy of the map header, not a deep copy.
func PtrIfNonEmptyMap[K comparable, V any](m map[K]V) *map[K]V {
	if len(m) == 0 {
		return nil
	}
	cp := m
	return &cp
}

// PtrIfNonEmptySlice returns *[]T when s has at least one element, otherwise nil.
func PtrIfNonEmptySlice[T any](s []T) *[]T {
	if len(s) == 0 {
		return nil
	}
	cp := s
	return &cp
}

// CopyMap returns a shallow copy of in (nil-safe).
func CopyMap[K comparable, V any](in map[K]V) map[K]V {
	if in == nil {
		return nil
	}
	out := make(map[K]V, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// PtrCopyMap returns a pointer to a new shallow copy of in.
// Nil-safe. Uses CopyMap.
func PtrCopyMap[K comparable, V any](in map[K]V) *map[K]V {
	if in == nil {
		return nil
	}
	out := CopyMap(in)
	return &out
}
