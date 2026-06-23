// Package client implements a Go client for the ScaleGrid API as exposed by the
// ScaleGrid console (https://console.scalegrid.io) and used by the official
// ScaleGrid CLI. The API is session/cookie based: callers authenticate with an
// account email + password via POST /login, and subsequent requests carry the
// returned session cookie.
//
// Responses use a common envelope: every body contains an "error" object whose
// "code" is "Success" on success. Asynchronous operations additionally return
// an "actionID" that can be polled via GET /actions/{id}.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the ScaleGrid console endpoint used by the CLI.
	DefaultBaseURL = "https://console.scalegrid.io"

	// DefaultTimeout is the per-request HTTP timeout.
	DefaultTimeout = 90 * time.Second

	userAgentBase = "terraform-provider-scalegrid"
)

// Config holds everything required to build and authenticate a Client.
type Config struct {
	// BaseURL is the console root. Defaults to DefaultBaseURL when empty.
	BaseURL string

	// Email and Password authenticate the ScaleGrid account.
	Email    string
	Password string

	// TwoFactorCode is an optional TOTP/2FA code. It is only valid at the
	// moment of login, so it is generally only useful for one-shot runs.
	TwoFactorCode string

	// UserAgent is prepended to the default User-Agent string.
	UserAgent string

	// HTTPClient lets callers inject a custom *http.Client (test servers,
	// proxies). A cookie jar is added automatically if the client lacks one.
	HTTPClient *http.Client

	// SkipLogin is used by tests to construct a Client without performing the
	// login round-trip.
	SkipLogin bool
}

// Client talks to the ScaleGrid console API.
type Client struct {
	baseURL    string
	userAgent  string
	httpClient *http.Client
}

// sgError mirrors the "error" envelope returned on every ScaleGrid response.
type sgError struct {
	Code                    string `json:"code"`
	ErrorMessageWithDetails string `json:"errorMessageWithDetails"`
	RecommendedAction       string `json:"recommendedAction"`
}

// errorEnvelope is used to peek at the error code before decoding the payload.
type errorEnvelope struct {
	Error sgError `json:"error"`
}

// codeSuccess and the restart-warning codes are treated as non-fatal, matching
// the CLI's behaviour.
const (
	codeSuccess               = "Success"
	codePostgreSQLRestartWarn = "PostgreSQLRestartWarning"
	codeMySQLRestartWarn      = "MySQLRestartWarning"
)

// APIError represents a ScaleGrid error response.
type APIError struct {
	Code              string
	Message           string
	RecommendedAction string
	StatusCode        int
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = e.Code
	}
	if e.RecommendedAction != "" {
		return fmt.Sprintf("scalegrid: %s (code %q): %s", msg, e.Code, e.RecommendedAction)
	}
	return fmt.Sprintf("scalegrid: %s (code %q)", msg, e.Code)
}

// IsNotFound reports whether err indicates a missing resource. ScaleGrid does
// not use HTTP 404; not-found conditions surface as error codes/messages, so we
// match on well-known substrings.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	hay := strings.ToLower(apiErr.Code + " " + apiErr.Message)
	return strings.Contains(hay, "notfound") ||
		strings.Contains(hay, "not found") ||
		strings.Contains(hay, "was not found") ||
		strings.Contains(hay, "does not exist")
}

// NewClient validates configuration, performs login (unless SkipLogin), and
// returns a ready Client.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Email == "" || cfg.Password == "" {
		if !cfg.SkipLogin {
			return nil, errors.New("scalegrid: email and password are required")
		}
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DefaultTimeout}
	}
	if httpClient.Jar == nil {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, fmt.Errorf("scalegrid: creating cookie jar: %w", err)
		}
		httpClient.Jar = jar
	}

	ua := userAgentBase
	if cfg.UserAgent != "" {
		ua = cfg.UserAgent + " " + userAgentBase
	}

	c := &Client{
		baseURL:    baseURL,
		userAgent:  ua,
		httpClient: httpClient,
	}

	if cfg.SkipLogin {
		return c, nil
	}
	if err := c.login(ctx, cfg.Email, cfg.Password, cfg.TwoFactorCode); err != nil {
		return nil, err
	}
	return c, nil
}

// login authenticates and stores the session cookie in the client's jar.
func (c *Client) login(ctx context.Context, email, password, twoFactor string) error {
	body := map[string]string{"username": email, "password": password}
	if twoFactor != "" {
		body["inputCode"] = twoFactor
	}

	var env errorEnvelope
	if err := c.do(ctx, http.MethodPost, "/login", body, &env); err != nil {
		return err
	}
	if env.Error.Code == "TwoFactorAuthNeeded" {
		return errors.New("scalegrid: two-factor authentication required; supply two_factor_code " +
			"(note: TOTP codes expire quickly) or disable 2FA on the automation account")
	}
	return nil
}

// do performs an HTTP request and decodes the response into out, returning an
// *APIError when the response envelope reports a non-success code.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("scalegrid: encoding request body: %w", err)
		}
		reqBody = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("scalegrid: building request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("scalegrid: performing %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("scalegrid: reading response body: %w", err)
	}

	// Some endpoints (e.g. logout via redirect) return no JSON body.
	if len(bytes.TrimSpace(raw)) == 0 {
		if resp.StatusCode >= 400 {
			return &APIError{Code: "HTTPError", Message: resp.Status, StatusCode: resp.StatusCode}
		}
		return nil
	}

	var env errorEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		// Body is not the standard envelope; treat HTTP status as authoritative.
		if resp.StatusCode >= 400 {
			return &APIError{Code: "HTTPError", Message: string(raw), StatusCode: resp.StatusCode}
		}
		return fmt.Errorf("scalegrid: decoding response: %w", err)
	}

	if !isSuccessCode(env.Error.Code) {
		return &APIError{
			Code:              env.Error.Code,
			Message:           env.Error.ErrorMessageWithDetails,
			RecommendedAction: env.Error.RecommendedAction,
			StatusCode:        resp.StatusCode,
		}
	}

	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("scalegrid: decoding response payload: %w", err)
		}
	}
	return nil
}

func isSuccessCode(code string) bool {
	switch code {
	case codeSuccess, codePostgreSQLRestartWarn, codeMySQLRestartWarn, "":
		return true
	default:
		return false
	}
}
