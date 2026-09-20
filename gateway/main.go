package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type tokenClaims struct {
	Type          string `json:"typ"`
	Subject       string `json:"sub,omitempty"`
	ClientID      string `json:"client_id,omitempty"`
	RedirectURI   string `json:"redirect_uri,omitempty"`
	CodeChallenge string `json:"code_challenge,omitempty"`
	Scope         string `json:"scope,omitempty"`
	ExpiresAt     int64  `json:"exp"`
	IssuedAt      int64  `json:"iat"`
}

var (
	ownerPassword string
	signingSecret []byte
	publicBaseURL string
	upstreamToken string
	upstreamURL   *url.URL
	proxy         *httputil.ReverseProxy
)

func env(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}

func main() {
	ownerPassword = os.Getenv("OWNER_PASSWORD")
	if ownerPassword == "" {
		log.Fatal("OWNER_PASSWORD is required")
	}
	secret := os.Getenv("OAUTH_SIGNING_SECRET")
	if len(secret) < 24 {
		log.Fatal("OAUTH_SIGNING_SECRET is required and should be at least 24 characters")
	}
	signingSecret = []byte(secret)

	publicBaseURL = strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/")
	upstreamToken = os.Getenv("UPSTREAM_AUTH_TOKEN")

	var err error
	upstreamURL, err = url.Parse(env("UPSTREAM_URL", "http://127.0.0.1:18061"))
	if err != nil {
		log.Fatalf("invalid UPSTREAM_URL: %v", err)
	}

	proxy = httputil.NewSingleHostReverseProxy(upstreamURL)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		if upstreamToken != "" {
			req.Header.Set("Authorization", "Bearer "+upstreamToken)
		} else {
			req.Header.Del("Authorization")
		}
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("upstream proxy error: %v", err)
		http.Error(w, "upstream MCP unavailable", http.StatusBadGateway)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/privacy", privacyHandler)
	mux.HandleFunc("/.well-known/oauth-protected-resource", protectedResourceHandler)
	mux.HandleFunc("/.well-known/oauth-authorization-server", authorizationServerMetadataHandler)
	mux.HandleFunc("/.well-known/openid-configuration", authorizationServerMetadataHandler)
	mux.HandleFunc("/.well-known/openai-apps-challenge", openAIChallengeHandler)
	mux.HandleFunc("/register", registerHandler)
	mux.HandleFunc("/authorize", authorizeHandler)
	mux.HandleFunc("/token", tokenHandler)
	mux.HandleFunc("/mcp", mcpHandler)
	mux.HandleFunc("/", rootHandler)

	port := env("PORT", "8080")
	log.Printf("sol-xiaohongshu gateway listening on :%s", port)
	log.Printf("upstream MCP: %s", upstreamURL.String())
	if publicBaseURL != "" {
		log.Printf("public base URL: %s", publicBaseURL)
	}
	if err := http.ListenAndServe(":"+port, securityHeaders(mux)); err != nil {
		log.Fatal(err)
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func baseURL(r *http.Request) string {
	if publicBaseURL != "" {
		return publicBaseURL
	}
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return proto + "://" + host
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"service": "sol-xiaohongshu",
	})
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintln(w, "sol-xiaohongshu MCP gateway")
	_, _ = fmt.Fprintln(w, "MCP endpoint: /mcp")
	_, _ = fmt.Fprintln(w, "Privacy: /privacy")
}

func privacyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, `<!doctype html>
<html><head><meta charset="utf-8"><title>sol-xiaohongshu Privacy</title></head>
<body style="max-width:760px;margin:40px auto;font-family:system-ui,sans-serif;line-height:1.6">
<h1>Privacy notice</h1>
<p>This deployment is a single-owner personal MCP gateway.</p>
<ul>
<li>Xiaohongshu session cookies and browser profile data are stored on the operator-controlled persistent disk.</li>
<li>The gateway does not intentionally persist ChatGPT prompts or MCP request bodies.</li>
<li>OAuth access and refresh tokens are self-contained signed tokens and are not stored in a database by this gateway.</li>
<li>The hosting provider and the upstream Xiaohongshu MCP implementation may process network and operational data under their own policies.</li>
<li>Deleting the deployment's persistent data removes the locally stored Xiaohongshu session state.</li>
</ul>
<p>This service is not affiliated with or endorsed by Xiaohongshu or OpenAI.</p>
</body></html>`)
}

func protectedResourceHandler(w http.ResponseWriter, r *http.Request) {
	b := baseURL(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"resource":               b,
		"authorization_servers":  []string{b},
		"scopes_supported":       []string{"xhs:read", "xhs:write"},
		"resource_documentation": b + "/privacy",
	})
}

func authorizationServerMetadataHandler(w http.ResponseWriter, r *http.Request) {
	b := baseURL(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                b,
		"authorization_endpoint":                b + "/authorize",
		"token_endpoint":                        b + "/token",
		"registration_endpoint":                 b + "/register",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"scopes_supported":                      []string{"xhs:read", "xhs:write"},
	})
}

func openAIChallengeHandler(w http.ResponseWriter, r *http.Request) {
	token := os.Getenv("OPENAI_APPS_CHALLENGE")
	if token == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(token))
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		RedirectURIs            []string `json:"redirect_uris"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
		ClientName              string   `json:"client_name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	for _, u := range req.RedirectURIs {
		if !redirectAllowed(u) {
			http.Error(w, "redirect_uri not allowed", http.StatusBadRequest)
			return
		}
	}

	clientID := "sol-xiaohongshu-chatgpt"
	writeJSON(w, http.StatusCreated, map[string]any{
		"client_id":                  clientID,
		"client_id_issued_at":        time.Now().Unix(),
		"redirect_uris":              req.RedirectURIs,
		"client_name":                req.ClientName,
		"token_endpoint_auth_method": "none",
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
	})
}

var authPage = template.Must(template.New("authorize").Parse(`<!doctype html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Authorize sol-xiaohongshu</title>
<style>
body{font-family:system-ui,sans-serif;background:#f7f7f8;margin:0;padding:32px}
.card{max-width:460px;margin:8vh auto;background:white;padding:28px;border-radius:16px;box-shadow:0 8px 30px rgba(0,0,0,.08)}
input{width:100%;box-sizing:border-box;padding:12px;border:1px solid #ccc;border-radius:10px;margin:8px 0 16px}
button{width:100%;padding:12px;border:0;border-radius:10px;background:#111;color:white;font-weight:600}
.small{color:#666;font-size:13px}
</style>
</head>
<body><div class="card">
<h2>Connect sol-xiaohongshu</h2>
<p>This grants the connected ChatGPT client access to the Xiaohongshu session stored in this deployment.</p>
<form method="post" action="/authorize">
<input type="hidden" name="response_type" value="{{.ResponseType}}">
<input type="hidden" name="client_id" value="{{.ClientID}}">
<input type="hidden" name="redirect_uri" value="{{.RedirectURI}}">
<input type="hidden" name="state" value="{{.State}}">
<input type="hidden" name="scope" value="{{.Scope}}">
<input type="hidden" name="code_challenge" value="{{.CodeChallenge}}">
<input type="hidden" name="code_challenge_method" value="{{.CodeChallengeMethod}}">
<label>Owner password</label>
<input type="password" name="password" autocomplete="current-password" required autofocus>
<button type="submit">Authorize</button>
</form>
<p class="small">Only continue if this is your own deployment.</p>
</div></body></html>`))

type authPageData struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	State               string
	Scope               string
	CodeChallenge       string
	CodeChallengeMethod string
}

func authorizeHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query()
		data := authPageData{
			ResponseType:        q.Get("response_type"),
			ClientID:            q.Get("client_id"),
			RedirectURI:         q.Get("redirect_uri"),
			State:               q.Get("state"),
			Scope:               q.Get("scope"),
			CodeChallenge:       q.Get("code_challenge"),
			CodeChallengeMethod: q.Get("code_challenge_method"),
		}
		if err := validateAuthorizeRequest(data); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = authPage.Execute(w, data)

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		data := authPageData{
			ResponseType:        r.Form.Get("response_type"),
			ClientID:            r.Form.Get("client_id"),
			RedirectURI:         r.Form.Get("redirect_uri"),
			State:               r.Form.Get("state"),
			Scope:               r.Form.Get("scope"),
			CodeChallenge:       r.Form.Get("code_challenge"),
			CodeChallengeMethod: r.Form.Get("code_challenge_method"),
		}
		if err := validateAuthorizeRequest(data); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if subtle.ConstantTimeCompare([]byte(r.Form.Get("password")), []byte(ownerPassword)) != 1 {
			http.Error(w, "invalid owner password", http.StatusUnauthorized)
			return
		}

		now := time.Now().Unix()
		code, err := signToken(tokenClaims{
			Type:          "code",
			ClientID:      data.ClientID,
			RedirectURI:   data.RedirectURI,
			CodeChallenge: data.CodeChallenge,
			Scope:         normalizedScope(data.Scope),
			IssuedAt:      now,
			ExpiresAt:     now + 300,
		})
		if err != nil {
			http.Error(w, "could not issue code", http.StatusInternalServerError)
			return
		}

		dest, _ := url.Parse(data.RedirectURI)
		q := dest.Query()
		q.Set("code", code)
		if data.State != "" {
			q.Set("state", data.State)
		}
		dest.RawQuery = q.Encode()
		http.Redirect(w, r, dest.String(), http.StatusFound)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func validateAuthorizeRequest(d authPageData) error {
	if d.ResponseType != "code" {
		return errors.New("only response_type=code is supported")
	}
	if d.ClientID == "" {
		return errors.New("client_id is required")
	}
	if !redirectAllowed(d.RedirectURI) {
		return errors.New("redirect_uri not allowed")
	}
	if d.CodeChallenge == "" || d.CodeChallengeMethod != "S256" {
		return errors.New("PKCE S256 is required")
	}
	return nil
}

func redirectAllowed(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Hostname() == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	allowed := strings.Split(env("OAUTH_ALLOWED_REDIRECT_HOSTS", "chatgpt.com,openai.com,localhost,127.0.0.1"), ",")
	for _, item := range allowed {
		item = strings.TrimSpace(strings.ToLower(item))
		if item == "" {
			continue
		}
		if host == item || strings.HasSuffix(host, "."+item) {
			return true
		}
	}
	return false
}

func tokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		oauthError(w, "invalid_request", "invalid form")
		return
	}

	switch r.Form.Get("grant_type") {
	case "authorization_code":
		exchangeAuthorizationCode(w, r)
	case "refresh_token":
		exchangeRefreshToken(w, r)
	default:
		oauthError(w, "unsupported_grant_type", "unsupported grant_type")
	}
}

func exchangeAuthorizationCode(w http.ResponseWriter, r *http.Request) {
	codeClaims, err := verifyToken(r.Form.Get("code"))
	if err != nil || codeClaims.Type != "code" {
		oauthError(w, "invalid_grant", "invalid authorization code")
		return
	}
	if r.Form.Get("redirect_uri") != codeClaims.RedirectURI {
		oauthError(w, "invalid_grant", "redirect_uri mismatch")
		return
	}
	if cid := r.Form.Get("client_id"); cid != "" && cid != codeClaims.ClientID {
		oauthError(w, "invalid_grant", "client_id mismatch")
		return
	}
	verifier := r.Form.Get("code_verifier")
	sum := sha256.Sum256([]byte(verifier))
	got := base64.RawURLEncoding.EncodeToString(sum[:])
	if subtle.ConstantTimeCompare([]byte(got), []byte(codeClaims.CodeChallenge)) != 1 {
		oauthError(w, "invalid_grant", "PKCE verification failed")
		return
	}
	issueTokens(w, codeClaims.ClientID, codeClaims.Scope)
}

func exchangeRefreshToken(w http.ResponseWriter, r *http.Request) {
	claims, err := verifyToken(r.Form.Get("refresh_token"))
	if err != nil || claims.Type != "refresh" {
		oauthError(w, "invalid_grant", "invalid refresh token")
		return
	}
	if cid := r.Form.Get("client_id"); cid != "" && cid != claims.ClientID {
		oauthError(w, "invalid_grant", "client_id mismatch")
		return
	}
	issueTokens(w, claims.ClientID, claims.Scope)
}

func issueTokens(w http.ResponseWriter, clientID, scope string) {
	now := time.Now().Unix()
	access, _ := signToken(tokenClaims{
		Type:      "access",
		Subject:   "owner",
		ClientID:  clientID,
		Scope:     normalizedScope(scope),
		IssuedAt:  now,
		ExpiresAt: now + int64((12 * time.Hour).Seconds()),
	})
	refresh, _ := signToken(tokenClaims{
		Type:      "refresh",
		Subject:   "owner",
		ClientID:  clientID,
		Scope:     normalizedScope(scope),
		IssuedAt:  now,
		ExpiresAt: now + int64((30 * 24 * time.Hour).Seconds()),
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  access,
		"token_type":    "Bearer",
		"expires_in":    int64((12 * time.Hour).Seconds()),
		"refresh_token": refresh,
		"scope":         normalizedScope(scope),
	})
}

func oauthError(w http.ResponseWriter, code, description string) {
	writeJSON(w, http.StatusBadRequest, map[string]any{
		"error":             code,
		"error_description": description,
	})
}

func normalizedScope(scope string) string {
	if strings.TrimSpace(scope) == "" {
		return "xhs:read xhs:write"
	}
	return strings.Join(strings.Fields(scope), " ")
}

func mcpHandler(w http.ResponseWriter, r *http.Request) {
	claims, err := authenticateRequest(r)
	if err != nil {
		b := baseURL(r)
		w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+b+`/.well-known/oauth-protected-resource", scope="xhs:read xhs:write"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	r.Header.Set("X-Sol-XHS-Subject", claims.Subject)
	proxy.ServeHTTP(w, r)
}

func authenticateRequest(r *http.Request) (*tokenClaims, error) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return nil, errors.New("missing bearer token")
	}
	raw := strings.TrimSpace(h[len("Bearer "):])
	claims, err := verifyToken(raw)
	if err != nil {
		return nil, err
	}
	if claims.Type != "access" || claims.Subject != "owner" {
		return nil, errors.New("invalid access token")
	}
	return claims, nil
}

func signToken(c tokenClaims) (string, error) {
	headerJSON := []byte(`{"alg":"HS256","typ":"JWT"}`)
	payloadJSON, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	unsigned := header + "." + payload
	mac := hmac.New(sha256.New, signingSecret)
	_, _ = mac.Write([]byte(unsigned))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return unsigned + "." + sig, nil
}

func verifyToken(raw string) (*tokenClaims, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed token")
	}
	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, signingSecret)
	_, _ = mac.Write([]byte(unsigned))
	expected := mac.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(got, expected) {
		return nil, errors.New("bad signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("bad payload")
	}
	var claims tokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, errors.New("bad claims")
	}
	if claims.ExpiresAt <= time.Now().Unix() {
		return nil, errors.New("token expired")
	}
	return &claims, nil
}

func init() {
	// Keep strconv linked in older Go toolchains where future metadata extensions
	// may use numeric client IDs. It is intentionally harmless.
	_ = strconv.IntSize
}
