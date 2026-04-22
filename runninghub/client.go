package runninghub

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL     *url.URL
	apiKey      string
	httpClient  *http.Client
	hostOverride string
	userAgent   string
	maxBodyBytes int64
}

type Option func(*Client) error

func WithBaseURL(raw string) Option {
	return func(c *Client) error {
		u, err := url.Parse(raw)
		if err != nil {
			return err
		}
		c.baseURL = u
		return nil
	}
}

func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) error {
		if hc == nil {
			return ErrNilClient
		}
		c.httpClient = hc
		return nil
	}
}

// WithHostOverride sets req.Host (rarely needed). Defaults to baseURL host.
func WithHostOverride(host string) Option {
	return func(c *Client) error {
		c.hostOverride = host
		return nil
	}
}

func WithUserAgent(ua string) Option {
	return func(c *Client) error {
		c.userAgent = ua
		return nil
	}
}

func WithMaxBodyBytes(n int64) Option {
	return func(c *Client) error {
		c.maxBodyBytes = n
		return nil
	}
}

func New(apiKey string, opts ...Option) (*Client, error) {
	base, _ := url.Parse("https://www.runninghub.cn")
	c := &Client{
		baseURL:     base,
		apiKey:      apiKey,
		httpClient:  &http.Client{Timeout: 60 * time.Second},
		userAgent:   "runninghub-go-sdk/0.1",
		maxBodyBytes: 10 << 20, // 10 MiB
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	if c.baseURL == nil {
		return nil, &url.Error{Op: "parse", URL: "", Err: io.EOF}
	}
	return c, nil
}

func (c *Client) newURL(path string, query url.Values) string {
	u := *c.baseURL
	u.Path = strings.TrimRight(u.Path, "/") + path
	if query != nil {
		u.RawQuery = query.Encode()
	}
	return u.String()
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, hdr http.Header, body io.Reader) (*http.Response, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	urlStr := c.newURL(path, query)
	req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
	if err != nil {
		return nil, nil, err
	}
	if c.hostOverride != "" {
		req.Host = c.hostOverride
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	for k, vals := range hdr {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	limit := c.maxBodyBytes
	if limit <= 0 {
		limit = 10 << 20
	}
	b, readErr := io.ReadAll(io.LimitReader(resp.Body, limit))
	if readErr != nil {
		return nil, nil, readErr
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, &HTTPError{StatusCode: resp.StatusCode, Body: string(b)}
	}
	return resp, b, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, hdr http.Header, in any, out any) error {
	var body io.Reader
	if in != nil {
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(in); err != nil {
			return err
		}
		body = buf
		if hdr == nil {
			hdr = make(http.Header)
		}
		hdr.Set("Content-Type", "application/json")
	}

	_, b, err := c.do(ctx, method, path, query, hdr, body)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		// Some endpoints are not strict schemas; retry without unknown-field disallow.
		dec2 := json.NewDecoder(bytes.NewReader(b))
		if err2 := dec2.Decode(out); err2 != nil {
			return err2
		}
	}
	return nil
}

func (c *Client) doRaw(ctx context.Context, method, path string, query url.Values, hdr http.Header, body io.Reader, out any) error {
	_, b, err := c.do(ctx, method, path, query, hdr, body)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	if err := dec.Decode(out); err != nil {
		return err
	}
	return nil
}

// DoJSON is a low-level helper for calling arbitrary endpoints.
// - If in != nil, it is encoded as JSON.
// - If out != nil, response body is decoded as JSON into out.
func (c *Client) DoJSON(ctx context.Context, method, path string, query url.Values, headers http.Header, in any, out any) error {
	return c.doJSON(ctx, method, path, query, headers, in, out)
}
