package axios

import (
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// absoluteURLRe matches absolute URLs the way axios helpers/isAbsoluteURL does:
// an optional RFC 3986 scheme ("<letter>[<letter|digit|+|-|.>]*:") followed by
// "//", or a protocol-relative "//host". Matching is case-insensitive.
var absoluteURLRe = regexp.MustCompile(`(?i)^([a-z][a-z\d+\-.]*:)?//`)

// IsAbsoluteURL reports whether rawURL is an absolute URL, mirroring axios's
// helpers/isAbsoluteURL. A URL is absolute when it begins with "<scheme>://" or
// is protocol-relative ("//host"). Per RFC 3986 a scheme name begins with a
// letter and is followed by any combination of letters, digits, plus, period or
// hyphen; a leading run that is not a valid scheme (for example "123://" or
// "!valid://") is therefore not absolute.
func IsAbsoluteURL(rawURL string) bool {
	return absoluteURLRe.MatchString(rawURL)
}

// combineTrailingSlashRe strips a trailing slash (or double slash) from a base
// URL, matching axios combineURLs' /\/?\/$/ replacement.
var combineTrailingSlashRe = regexp.MustCompile(`/?/$`)

// combineLeadingSlashRe strips leading slashes from a relative URL, matching
// axios combineURLs' /^\/+/ replacement.
var combineLeadingSlashRe = regexp.MustCompile(`^/+`)

// CombineURLs joins baseURL and relativeURL the way axios's helpers/combineURLs
// does. Trailing slashes are stripped from baseURL and leading slashes from
// relativeURL, and the two are joined with a single "/". When relativeURL is
// empty, baseURL is returned unchanged.
func CombineURLs(baseURL, relativeURL string) string {
	if relativeURL == "" {
		return baseURL
	}
	return combineTrailingSlashRe.ReplaceAllString(baseURL, "") + "/" +
		combineLeadingSlashRe.ReplaceAllString(relativeURL, "")
}

// BuildFullPath combines baseURL with requestedURL exactly like axios's
// core/buildFullPath. When baseURL is non-empty and either requestedURL is
// relative or allowAbsoluteURLs is false, the two are combined via CombineURLs;
// otherwise requestedURL is returned untouched. This lets an absolute
// requestedURL override the configured base (the axios default) while
// allowAbsoluteURLs=false forces every request under the base.
func BuildFullPath(baseURL, requestedURL string, allowAbsoluteURLs bool) string {
	isRelative := !IsAbsoluteURL(requestedURL)
	if baseURL != "" && (isRelative || !allowAbsoluteURLs) {
		return CombineURLs(baseURL, requestedURL)
	}
	return requestedURL
}

// isUnreservedURIComponent reports whether c is left unescaped by JavaScript's
// encodeURIComponent: ASCII letters, digits, and the marks -_.!~*'().
func isUnreservedURIComponent(c byte) bool {
	switch {
	case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		return true
	}
	switch c {
	case '-', '_', '.', '!', '~', '*', '\'', '(', ')':
		return true
	}
	return false
}

const uriHexUpper = "0123456789ABCDEF"

// encodeURIComponentJS percent-encodes s byte-for-byte the way JavaScript's
// encodeURIComponent does, using uppercase hex and operating on the UTF-8
// bytes of s.
func encodeURIComponentJS(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUnreservedURIComponent(c) {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(uriHexUpper[c>>4])
		b.WriteByte(uriHexUpper[c&0xf])
	}
	return b.String()
}

// EncodeURIComponent encodes a query key or value the way axios's helpers/buildURL
// encode function does: like JavaScript's encodeURIComponent, but with ':', '$'
// and ',' left unescaped and a space encoded as '+' rather than "%20". Square
// brackets and other reserved characters remain percent-encoded.
func EncodeURIComponent(s string) string {
	e := encodeURIComponentJS(s)
	e = strings.ReplaceAll(e, "%3A", ":")
	e = strings.ReplaceAll(e, "%24", "$")
	e = strings.ReplaceAll(e, "%2C", ",")
	e = strings.ReplaceAll(e, "%20", "+")
	return e
}

// serializeParamsAxios renders params into a query string using axios's encode
// rules (see EncodeURIComponent) and the given ArrayFormat for multi-valued
// keys. Keys are emitted in sorted order for deterministic output.
func serializeParamsAxios(params url.Values, format ArrayFormat) string {
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
		b.WriteString(EncodeURIComponent(key))
		b.WriteByte('=')
		b.WriteString(EncodeURIComponent(val))
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
		case format == ArrayFormatIndices:
			for i, v := range vs {
				write(k+"["+strconv.Itoa(i)+"]", v)
			}
		case format == ArrayFormatRepeat:
			for _, v := range vs {
				write(k, v)
			}
		default: // ArrayFormatBrackets is the axios default for arrays.
			for _, v := range vs {
				write(k+"[]", v)
			}
		}
	}
	return b.String()
}

// BuildURL appends serialized query params to rawURL, mirroring axios's
// helpers/buildURL. Params are encoded with EncodeURIComponent, and
// multi-valued keys are expanded according to format (axios defaults arrays to
// bracket notation, ArrayFormatBrackets). Any fragment ("#...") in rawURL is
// discarded, and the params are appended after "?" when rawURL has no query or
// "&" when it already does. When params is empty, rawURL is returned unchanged
// (fragment included). Keys are emitted in sorted order for deterministic
// output.
func BuildURL(rawURL string, params url.Values, format ArrayFormat) string {
	serialized := serializeParamsAxios(params, format)
	if serialized == "" {
		return rawURL
	}
	if i := strings.IndexByte(rawURL, '#'); i != -1 {
		rawURL = rawURL[:i]
	}
	sep := "?"
	if strings.IndexByte(rawURL, '?') != -1 {
		sep = "&"
	}
	return rawURL + sep + serialized
}
