package axios

import (
	"net/url"
	"regexp"
	"sort"
	"strconv"
)

// FormToJSON converts flat, bracket-notation form values into a nested
// map[string]any, mirroring the axios helper of the same name. It is the
// inverse of the bracket-notation flattening used for query/form serialization
// (see FlattenParams).
//
// Keys are interpreted the way axios's parsePropPath does, splitting on
// brackets as well as any other non-word separator (".", "-", " "):
//
//   - "a"            -> {"a": "1"}
//   - "a[b]"         -> {"a": {"b": "1"}}
//   - "a[b][c]"      -> {"a": {"b": {"c": "1"}}}
//   - "a.b.c"        -> {"a": {"b": {"c": "1"}}}
//   - "a[]"          -> {"a": ["1", "2", ...]}   (empty brackets append)
//   - "a[0]", "a[1]" -> {"a": ["x", "y"]}        (numeric brackets index)
//
// When a key repeats (url.Values may hold several values for one key) the
// values are collected into a slice in order. Leaf values are always strings.
// Keys are processed in sorted order so the result is deterministic.
//
// As a guard against prototype-pollution (matching the axios CVE fix), any key
// whose path contains a "__proto__" segment is skipped entirely.
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
		if len(tokens) == 0 || ftjHasProto(tokens) {
			continue
		}
		for _, v := range values[k] {
			root[tokens[0]] = ftjMerge(root[tokens[0]], tokens[1:], v)
		}
	}
	return root
}

// ftjHasProto reports whether any token is the dangerous "__proto__" key.
func ftjHasProto(tokens []string) bool {
	for _, t := range tokens {
		if t == "__proto__" {
			return true
		}
	}
	return false
}

// ftjKeyRe mirrors axios parsePropPath's /\w+|\[(\w*)]/g: each match is either a
// run of word characters or a bracketed segment (whose inner content, possibly
// empty, is captured in group 1).
var ftjKeyRe = regexp.MustCompile(`\w+|\[(\w*)\]`)

// ftjParseKey splits a bracket- or dot-notation key into its path tokens the
// way axios parsePropPath does. A bare "[]" yields an empty token (array
// append); a bracketed "[name]" yields "name"; any other run of word
// characters yields itself.
func ftjParseKey(key string) []string {
	matches := ftjKeyRe.FindAllStringSubmatch(key, -1)
	tokens := make([]string, 0, len(matches))
	for _, m := range matches {
		switch {
		case m[0] == "[]":
			tokens = append(tokens, "")
		case m[1] != "":
			tokens = append(tokens, m[1])
		default:
			tokens = append(tokens, m[0])
		}
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
