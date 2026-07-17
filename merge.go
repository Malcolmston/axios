package axios

import (
	"net/http"
	"net/url"
)

// mergeConfig deep-merges override onto base and returns the result, mirroring
// how axios.create merges instance config over defaults. Map-valued fields
// (headers, header groups, params) are merged key-by-key with override winning;
// scalar, pointer, slice and function fields are taken from override when set,
// otherwise from base.
func mergeConfig(base, override Config) Config {
	out := base

	out.Headers = mergeHeader(base.Headers, override.Headers)
	out.HeaderGroups = mergeHeaderGroups(base.HeaderGroups, override.HeaderGroups)
	out.Params = mergeValues(base.Params, override.Params)
	out.ParamsMap = mergeAnyMap(base.ParamsMap, override.ParamsMap)

	if override.BaseURL != "" {
		out.BaseURL = override.BaseURL
	}
	if override.ParamsSerializer != nil {
		out.ParamsSerializer = override.ParamsSerializer
	}
	if override.ArrayFormat != 0 {
		out.ArrayFormat = override.ArrayFormat
	}
	if override.Timeout != 0 {
		out.Timeout = override.Timeout
	}
	if override.BasicAuth != nil {
		out.BasicAuth = override.BasicAuth
	}
	if override.BearerToken != "" {
		out.BearerToken = override.BearerToken
	}
	if override.Proxy != nil {
		out.Proxy = override.Proxy
	}
	if override.HTTPClient != nil {
		out.HTTPClient = override.HTTPClient
	}
	if override.Transport != nil {
		out.Transport = override.Transport
	}
	if override.Context != nil {
		out.Context = override.Context
	}
	if override.Signal != nil {
		out.Signal = override.Signal
	}
	if override.ValidateStatus != nil {
		out.ValidateStatus = override.ValidateStatus
	}
	if override.MaxRedirects != 0 {
		out.MaxRedirects = override.MaxRedirects
	}
	if override.RedirectPolicy != nil {
		out.RedirectPolicy = override.RedirectPolicy
	}
	if override.MaxContentLength != 0 {
		out.MaxContentLength = override.MaxContentLength
	}
	if override.MaxBodyLength != 0 {
		out.MaxBodyLength = override.MaxBodyLength
	}
	if override.Decompress != nil {
		out.Decompress = override.Decompress
	}
	if override.ResponseType != 0 {
		out.ResponseType = override.ResponseType
	}
	if override.XSRFCookieName != "" {
		out.XSRFCookieName = override.XSRFCookieName
	}
	if override.XSRFHeaderName != "" {
		out.XSRFHeaderName = override.XSRFHeaderName
	}
	if override.OnUploadProgress != nil {
		out.OnUploadProgress = override.OnUploadProgress
	}
	if override.OnDownloadProgress != nil {
		out.OnDownloadProgress = override.OnDownloadProgress
	}
	if override.TransformRequest != nil {
		out.TransformRequest = override.TransformRequest
	}
	if override.TransformResponse != nil {
		out.TransformResponse = override.TransformResponse
	}
	if override.Retry != nil {
		out.Retry = override.Retry
	}
	if len(override.RequestInterceptors) > 0 {
		out.RequestInterceptors = append(append([]RequestInterceptor(nil), base.RequestInterceptors...), override.RequestInterceptors...)
	}
	if len(override.ResponseInterceptors) > 0 {
		out.ResponseInterceptors = append(append([]ResponseInterceptor(nil), base.ResponseInterceptors...), override.ResponseInterceptors...)
	}
	return out
}

func mergeHeader(base, override http.Header) http.Header {
	if base == nil && override == nil {
		return nil
	}
	out := http.Header{}
	for k, vs := range base {
		out[k] = append([]string(nil), vs...)
	}
	for k, vs := range override {
		out[http.CanonicalHeaderKey(k)] = append([]string(nil), vs...)
	}
	return out
}

func mergeValues(base, override url.Values) url.Values {
	if base == nil && override == nil {
		return nil
	}
	out := url.Values{}
	for k, vs := range base {
		out[k] = append([]string(nil), vs...)
	}
	for k, vs := range override {
		out[k] = append([]string(nil), vs...)
	}
	return out
}

func mergeAnyMap(base, override map[string]any) map[string]any {
	if base == nil && override == nil {
		return nil
	}
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}

func mergeHeaderGroups(base, override HeaderDefaults) HeaderDefaults {
	return HeaderDefaults{
		Common:  mergeHeader(base.Common, override.Common),
		Get:     mergeHeader(base.Get, override.Get),
		Post:    mergeHeader(base.Post, override.Post),
		Put:     mergeHeader(base.Put, override.Put),
		Patch:   mergeHeader(base.Patch, override.Patch),
		Delete:  mergeHeader(base.Delete, override.Delete),
		Head:    mergeHeader(base.Head, override.Head),
		Options: mergeHeader(base.Options, override.Options),
	}
}
