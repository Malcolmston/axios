# Changelog

All notable changes to this project are documented here. This project adheres
to semantic versioning.

## 0.3.0

Further parity push toward the axios feature set. All additions are backward
compatible and standard-library only.

### Added

- Multipart uploads: a `FormData` builder (`NewFormData`) mirroring the browser
  FormData object axios accepts as a body, with `AddField`, `AddFile`,
  `AddFileBytes`, `AddFilePart`, `SetBoundary`/`Boundary`, `ContentType`,
  `Len`, `Reader`, `Bytes` and `WriteTo`. Encoding is deterministic (fixed
  default boundary, insertion order preserved).
- `FormToJSON`: parses flat bracket-notation form values (`a[b][c]`, `a[]`,
  `a[0]`) into a nested `map[string]any`, mirroring the axios `formToJSON`
  helper and inverting bracket flattening.
- `MergeConfig`: exposes the deep config merge used by `Create`/`Client.Create`
  (axios `mergeConfig`).
- Typed request helpers `PostJSON`/`PutJSON`/`PatchJSON`/`DeleteJSON` and the
  method-agnostic `RequestJSON`, complementing `GetJSON` for the typed
  `axios.post<T>()`-style calls.
- Response helpers: status classifiers `IsInformational`/`IsRedirect`/
  `IsClientError`/`IsServerError`, header accessors `ContentType`/
  `ContentLength`/`Location`/`Cookies`, and `RetryAfter` (parses both the
  seconds and HTTP-date forms of the `Retry-After` header).

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
