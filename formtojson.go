package axios

import (
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// FormToJSON converts flat, bracket-notation form values into a nested
// map[string]any, mirroring the axios helper of the same name. It is the
// inverse of the bracket-notation flattening used for query/form serialization
// (see FlattenParams).
//
// Keys are interpreted as follows:
//
//   - "a"            -> {"a": "1"}
//   - "a[b]"         -> {"a": {"b": "1"}}
//   - "a[b][c]"      -> {"a": {"b": {"c": "1"}}}
//   - "a[]"          -> {"a": ["1", "2", ...]}   (empty brackets append)
//   - "a[0]", "a[1]" -> {"a": ["x", "y"]}        (numeric brackets index)
//
// When a key repeats (url.Values may hold several values for one key) the
// values are collected into a slice in order. Leaf values are always strings.
// Keys are processed in sorted order so the result is deterministic.
func FormToJSON(values url.Values) map[string]any {
	root := map[string]any{}
	if len(values) == 0 {
		return root
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		tokens := ftjParseKey(k)
		if len(tokens) == 0 {
			continue
		}
		for _, v := range values[k] {
			root[tokens[0]] = ftjMerge(root[tokens[0]], tokens[1:], v)
		}
	}
	return root
}

// ftjParseKey splits a bracket-notation key into its path tokens. The first
// token is the portion before the first '[', followed by one token per
// bracketed segment (which may be empty for "[]").
func ftjParseKey(key string) []string {
	open := strings.IndexByte(key, '[')
	if open < 0 {
		return []string{key}
	}
	tokens := []string{key[:open]}
	rest := key[open:]
	for len(rest) > 0 {
		if rest[0] != '[' {
			// Malformed remainder; treat the rest as a single literal token.
			tokens = append(tokens, rest)
			break
		}
		close := strings.IndexByte(rest, ']')
		if close < 0 {
			tokens = append(tokens, rest[1:])
			break
		}
		tokens = append(tokens, rest[1:close])
		rest = rest[close+1:]
	}
	return tokens
}

// ftjMerge folds value into existing following the remaining path tokens,
// returning the updated node.
func ftjMerge(existing any, tokens []string, value string) any {
	if len(tokens) == 0 {
		switch cur := existing.(type) {
		case nil:
			return value
		case []any:
			return append(cur, value)
		default:
			return []any{cur, value}
		}
	}
	tok := tokens[0]
	rest := tokens[1:]

	if tok == "" { // append to an array
		arr, _ := existing.([]any)
		arr = append(arr, ftjMerge(nil, rest, value))
		return arr
	}

	if idx, err := strconv.Atoi(tok); err == nil && idx >= 0 {
		arr, _ := existing.([]any)
		for len(arr) <= idx {
			arr = append(arr, nil)
		}
		arr[idx] = ftjMerge(arr[idx], rest, value)
		return arr
	}

	m, _ := existing.(map[string]any)
	if m == nil {
		m = map[string]any{}
	}
	m[tok] = ftjMerge(m[tok], rest, value)
	return m
}
