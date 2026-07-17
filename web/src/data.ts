// Library content for the axios documentation site. Mirrors the shape used by
// the malcolmston/go landing site's data.ts so the sibling sites stay in sync.
export interface Lib {
  id: string; name: string; icon: string; accent: string; pkg: string; node: string;
  repo: string; docs: string; tagline: string; blurb: string; tags: string[];
  features: string[]; node_code: string; go_code: string; integrate: string;
}

export const NODE_ACCENT = '#8cc84b';

export const AXIOS: Lib = {
  id:"axios", name:"axios", icon:'<i class="fa-solid fa-arrow-right-arrow-left"></i>', accent:"#a78bfa",
  pkg:"github.com/malcolmston/axios", node:"axios/axios",
  repo:"https://github.com/malcolmston/axios", docs:"https://malcolmston.github.io/axios/",
  tagline:"An ergonomic, axios-style HTTP client for Go.",
  blurb:"A from-scratch, standard-library-only Go HTTP client that brings the ergonomics of JavaScript's axios "+
    "to net/http. Create a configured client with axios.New(Config{...}) — BaseURL, default Headers and query "+
    "Params, a Timeout, Basic or Bearer auth, retries, and request/response interceptors — or reach for the "+
    "package-level Get/Post helpers backed by a default client. Verb methods encode request bodies automatically "+
    "by their dynamic type (JSON for structs and maps, form-urlencoded for url.Values, raw for []byte/string/"+
    "io.Reader) and return a rich Response with JSON, Text, Bytes, OK and Header helpers. Non-2xx statuses "+
    "resolve to a typed *Error that still carries the parsed Response, ValidateStatus lets you redefine success, "+
    "and the generic GetJSON[T] fetches and decodes in a single call. No cgo, no third-party dependencies.",
  tags:["net/http","BaseURL","interceptors","retries","Bearer/Basic auth","auto body encoding","typed errors","GetJSON[T]","form + JSON","ValidateStatus","zero deps"],
  features:[
    "Configurable client via <code>New</code> and <code>Config</code> (<code>BaseURL</code>, <code>Headers</code>, <code>Params</code>, <code>Timeout</code>, <code>Context</code>) plus package-level <code>Get</code>/<code>Post</code>/… helpers backed by <code>Default</code>/<code>SetDefault</code>",
    "Full set of verb methods — <code>Get</code>, <code>Delete</code>, <code>Head</code>, <code>Options</code>, <code>Post</code>, <code>Put</code>, <code>Patch</code> — all built on the low-level <code>Request</code>",
    "Automatic body encoding by dynamic type via <code>EncodeBody</code>: JSON for structs/maps, form for <code>url.Values</code>, raw for <code>[]byte</code>/<code>string</code>/<code>io.Reader</code>",
    "Rich <code>Response</code> with <code>JSON</code>, <code>Text</code>, <code>Bytes</code>, <code>OK</code> and <code>Header</code> helpers over a fully-buffered body",
    "Authentication built in — <code>BearerToken</code> and <code>BasicAuth</code>, overridable per request through <code>RequestConfig</code>",
    "Ordered request &amp; response interceptors — <code>RequestInterceptor</code> and <code>ResponseInterceptor</code> — that mutate or transform in place",
    "Configurable retries via <code>RetryConfig</code> with <code>DefaultBackoff</code> (exponential) and <code>DefaultRetryOn</code> (transport errors + 5xx)",
    "Typed <code>*Error</code> that carries the parsed <code>Response</code> on rejected statuses, unwraps transport errors, and exposes <code>StatusCode</code>",
    "Redefine success with <code>ValidateStatus</code> (per client or per request), just like axios <code>validateStatus</code>",
    "Generic one-liner fetch + decode — <code>GetJSON[T]</code> and <code>GetJSONDefault[T]</code>",
    "Zero dependencies — pure Go standard library, nothing to audit but the toolchain"
  ],
  node_code:
`import axios from "axios";

const api = axios.create({
  baseURL: "https://api.example.com",
  timeout: 5000,
  headers: { Authorization: "Bearer secret-token" },
});

const { data } = await api.get("/users/1", { params: { expand: "profile" } });
console.log(data.name);

await api.post("/users", { name: "Ada", age: 36 });`,
  go_code:
`import "github.com/malcolmston/axios"

client := axios.New(axios.Config{
	BaseURL:     "https://api.example.com",
	Timeout:     5 * time.Second,
	BearerToken: "secret-token",
})

resp, _ := client.Get("/users/1", &axios.RequestConfig{
	Params: url.Values{"expand": {"profile"}},
})
var u User
_ = resp.JSON(&u)

// One-liner fetch + decode with generics.
u2, _ := axios.GetJSON[User](client, "/users/2")

client.Post("/users", User{Name: "Ada", Age: 36})`,
  integrate:
`<span class="tok-c">// Retry 5xx/network errors and tag every outgoing request via an interceptor.</span>
client := axios.New(axios.Config{
	BaseURL: "https://api.example.com",
	Retry:   &axios.RetryConfig{Retries: 3}, <span class="tok-c">// exponential backoff by default</span>
	RequestInterceptors: []axios.RequestInterceptor{
		func(req *http.Request) error { req.Header.Set("X-Trace", "abc"); return nil },
	},
})

<span class="tok-c">// POST a struct — it is JSON-encoded automatically by its dynamic type.</span>
resp, err := client.Post("/users", User{Name: "Ada", Age: 36})

<span class="tok-c">// Non-2xx becomes a typed *Error whose Response still holds the body.</span>
var aerr *axios.Error
if errors.As(err, &aerr) {
	fmt.Println(aerr.StatusCode(), aerr.Response.Text())
}

<span class="tok-c">// Redefine success: accept anything below 500.</span>
ok, _ := client.Get("/maybe", &axios.RequestConfig{
	ValidateStatus: func(status int) bool { return status < 500 },
})
fmt.Println(ok.OK(), ok.Header("Content-Type"))`
};
