# Changelog

All notable changes to this project are documented here. This project adheres
to semantic versioning.

## 0.2.0

Large parity push toward the axios feature set. All additions are backward
compatible; existing code continues to work unchanged.

### Added

- Cancellation: `AbortController`/`AbortSignal` and the legacy `CancelToken`
  (`NewCancelToken`), wired through `RequestConfig.Signal`/`CancelToken` and
  `Config.Signal`, plus `IsCancel` and the `ErrCodeCanceled` classification.
- Progress: `OnUploadProgress`/`OnDownloadProgress` callbacks receiving
  `ProgressEvent` values, backed by a counting reader.
- Transforms: `TransformRequest`/`TransformResponse` pipelines that rewrite the
  raw body bytes and may mutate headers, complementing interceptors.
- Response types: `ResponseType` (default/json/text/bytes/stream) with a true
  streaming mode exposing `Response.Stream` and `Response.Close`.
- Redirects: `MaxRedirects` (cap, disable, or default) and a `RedirectPolicy`
  hook.
- Size guards: `MaxContentLength` and `MaxBodyLength`.
- Query parameters: `ParamsSerializer` hook, `ArrayFormat`
  (repeat/brackets/indices/comma) via `SerializeParams`, and nested map support
  via `ParamsMap`/`FlattenParams`.
- Automatic gzip/deflate decompression with `Accept-Encoding` negotiation,
  toggleable via `Config.Decompress`.
- XSRF/CSRF token cookie-to-header copying (`XSRFCookieName`/`XSRFHeaderName`).
- Helpers: `All`/`Spread`, instance and package-level `Create` with deep-merged
  defaults, `IsAxiosError`, `AsError`, `Error.ToJSON`/`MarshalJSON`, `GetUri`,
  and default header groups (`HeaderDefaults`, `DefaultHeaderGroups`).
- Proxy configuration (`ProxyConfig`) and typed error codes (`ErrCode*`).

## 0.1.0

- Initial release: configurable client, verb methods, automatic body
  encoding/decoding, interceptors, retries with backoff, typed `*Error` with
  `ValidateStatus`, and the generic `GetJSON` helper.
