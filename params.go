package axios

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// ArrayFormat controls how repeated query parameters (values with more than one
// entry) are serialized, mirroring the qs/axios paramsSerializer options.
type ArrayFormat int

const (
	// ArrayFormatRepeat repeats the key for each value: a=1&a=2. This is the
	// default and matches url.Values.Encode.
	ArrayFormatRepeat ArrayFormat = iota
	// ArrayFormatBrackets appends empty brackets to the key: a[]=1&a[]=2.
	ArrayFormatBrackets
	// ArrayFormatIndices appends the element index in brackets: a[0]=1&a[1]=2.
	ArrayFormatIndices
	// ArrayFormatComma joins the values with commas into a single pair: a=1,2.
	ArrayFormatComma
)

// ParamsSerializer converts query parameters into an encoded query string
// (without a leading "?"). It mirrors the axios paramsSerializer hook. When set
// on a Config or RequestConfig it fully replaces the built-in serialization.
type ParamsSerializer func(params url.Values) string

// SerializeParams encodes params into a query string using the given
// ArrayFormat. Keys and values are URL-escaped and keys are emitted in sorted
// order for deterministic output.
func SerializeParams(params url.Values, format ArrayFormat) string {
	if len(params) == 0 {
		return ""
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	write := func(key, val string) {
		if b.Len() > 0 {
			b.WriteByte('&')
		}
		b.WriteString(url.QueryEscape(key))
		b.WriteByte('=')
		b.WriteString(url.QueryEscape(val))
	}

	for _, k := range keys {
		vs := params[k]
		switch {
		case len(vs) <= 1:
			for _, v := range vs {
				write(k, v)
			}
		case format == ArrayFormatComma:
			write(k, strings.Join(vs, ","))
		case format == ArrayFormatBrackets:
			for _, v := range vs {
				write(k+"[]", v)
			}
		case format == ArrayFormatIndices:
			for i, v := range vs {
				write(k+"["+strconv.Itoa(i)+"]", v)
			}
		default: // ArrayFormatRepeat
			for _, v := range vs {
				write(k, v)
			}
		}
	}
	return b.String()
}

// FlattenParams turns a nested map[string]any into flat url.Values using
// bracket notation for nested objects and arrays, matching how axios serializes
// object params. For example {"filter": {"name": "ada"}, "ids": []any{1, 2}}
// becomes filter[name]=ada, ids[]=1, ids[]=2 (with ArrayFormatBrackets). Scalar
// values are stringified; nested maps recurse with bracketed keys.
func FlattenParams(m map[string]any) url.Values {
	out := url.Values{}
	for k, v := range m {
		flattenInto(out, k, v)
	}
	return out
}

func flattenInto(out url.Values, key string, v any) {
	switch val := v.(type) {
	case map[string]any:
		for k, sub := range val {
			flattenInto(out, key+"["+k+"]", sub)
		}
	case []any:
		for _, sub := range val {
			flattenInto(out, key, sub)
		}
	case nil:
		// skip nils, matching axios which omits null/undefined params.
	default:
		out.Add(key, stringifyParam(val))
	}
}

func stringifyParam(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case bool:
		return strconv.FormatBool(val)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'g', -1, 64)
	default:
		return fmt.Sprintf("%v", val)
	}
}
