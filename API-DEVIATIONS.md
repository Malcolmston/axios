# API deviations from upstream axios

This port aims for behavioural parity with `axios@1.7.9`, verified by the
cross-language harness in the aggregator's `parity/axios/`. The cases below are
deliberate, documented divergences: places where matching upstream exactly is
impossible or undesirable in idiomatic Go. Every other measured case matches.

## `params-multi-key-order` — query parameter insertion order

**Upstream:** axios serialises `params` from a JavaScript object, which preserves
insertion order, so `{z:1, a:2}` yields `?z=1&a=2`.

**Port:** query parameters enter through Go maps (`map[string]any` via
`ParamsMap`, or `url.Values` via `Params`). Go maps have no defined iteration
order, so there is no insertion order to preserve. To keep output deterministic
the port emits keys in sorted order, yielding `?a=2&z=1`.

This is a language/API constraint, not a behavioural bug: the encoding of any
individual key/value pair matches axios exactly (bracketed arrays and nested
objects included). Callers who need a specific parameter order can pre-render the
query string into the URL, or supply a `ParamsSerializer`.

## `headers-default-user-agent` — the default `User-Agent`

**Upstream:** axios sends `User-Agent: axios/<version>` (e.g. `axios/1.7.9`).

**Port:** the port sends `User-Agent: axios/<port-version>` (see `Version`, e.g.
`axios/0.3.0`). It advertises an axios-style identity but reports its own release,
so it will never string-match a specific pinned upstream version. Reporting the
port's real version is the honest choice; impersonating a particular upstream
build would be misleading.

## Note: request `Content-Type` for raw bodies

Not a deviation (it matches upstream and is listed here only to record the
reasoning): axios inherits `application/x-www-form-urlencoded` from
`defaults.headers[method]` for POST/PUT/PATCH bodies that carry no type of their
own — raw strings, byte buffers and empty bodies included. The port mirrors this
so the wire bytes match. Structured values still serialise to `application/json`,
`URLSearchParams`/`url.Values` to `application/x-www-form-urlencoded;charset=utf-8`,
and `FormData` to `multipart/form-data`.
