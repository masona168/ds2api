package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	adminauth "ds2api/internal/auth"

	"github.com/go-chi/chi/v5"
)

func TestAPIRoutesRemainRegistered(t *testing.T) {
	t.Setenv("DS2API_CONFIG_JSON", `{"keys":["k1"],"accounts":[{"email":"u@example.com","password":"p"}]}`)
	t.Setenv("DS2API_ENV_WRITEBACK", "0")

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	routes, ok := app.Router.(chi.Routes)
	if !ok {
		t.Fatalf("app router does not expose chi routes: %T", app.Router)
	}

	got := map[string]bool{}
	if err := chi.Walk(routes, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		got[fmt.Sprintf("%s %s", method, route)] = true
		return nil
	}); err != nil {
		t.Fatalf("walk routes: %v", err)
	}

	for _, want := range []string{
		"GET /v1/models",
		"GET /v1/models/{model_id}",
		"POST /v1/chat/completions",
		"POST /v1/responses",
		"GET /v1/responses/{response_id}",
		"POST /v1/files",
		"GET /v1/files/{file_id}",
		"POST /v1/embeddings",
		"GET /models",
		"GET /models/{model_id}",
		"POST /chat/completions",
		"POST /responses",
		"GET /responses/{response_id}",
		"POST /files",
		"GET /files/{file_id}",
		"POST /embeddings",
		"GET /anthropic/v1/models",
		"POST /anthropic/v1/messages",
		"POST /anthropic/v1/messages/count_tokens",
		"POST /v1/messages",
		"POST /messages",
		"POST /v1/messages/count_tokens",
		"POST /messages/count_tokens",
		"POST /v1beta/models/{model}:generateContent",
		"POST /v1beta/models/{model}:streamGenerateContent",
		"POST /v1/models/{model}:generateContent",
		"POST /v1/models/{model}:streamGenerateContent",
		"POST /admin/login",
		"GET /admin/verify",
		"GET /admin/config",
		"POST /admin/config",
		"GET /admin/settings",
		"PUT /admin/settings",
		"POST /admin/settings/password",
		"POST /admin/config/import",
		"GET /admin/config/export",
		"POST /admin/keys",
		"PUT /admin/keys/{key}",
		"DELETE /admin/keys/{key}",
		"GET /admin/proxies",
		"POST /admin/proxies",
		"PUT /admin/proxies/{proxyID}",
		"DELETE /admin/proxies/{proxyID}",
		"POST /admin/proxies/test",
		"GET /admin/accounts",
		"POST /admin/accounts",
		"PUT /admin/accounts/{identifier}",
		"DELETE /admin/accounts/{identifier}",
		"PUT /admin/accounts/{identifier}/proxy",
		"GET /admin/queue/status",
		"POST /admin/accounts/test",
		"POST /admin/accounts/test-all",
		"POST /admin/accounts/sessions/delete-all",
		"POST /admin/import",
		"POST /admin/test",
		"POST /admin/dev/raw-samples/capture",
		"GET /admin/dev/raw-samples/query",
		"POST /admin/dev/raw-samples/save",
		"POST /admin/vercel/sync",
		"GET /admin/vercel/status",
		"POST /admin/vercel/status",
		"GET /admin/export",
		"GET /admin/dev/captures",
		"DELETE /admin/dev/captures",
		"GET /admin/chat-history",
		"GET /admin/chat-history/{id}",
		"DELETE /admin/chat-history",
		"DELETE /admin/chat-history/{id}",
		"PUT /admin/chat-history/settings",
		"GET /admin/version",
	} {
		if !got[want] {
			t.Fatalf("expected route %s to be registered", want)
		}
	}
}

func TestAdminProxiesBrowserNavigationServesWebUIShell(t *testing.T) {
	staticDir := t.TempDir()
	indexPath := filepath.Join(staticDir, "index.html")
	if err := os.WriteFile(indexPath, []byte(`<!doctype html><html><head><title>Admin</title></head><body>admin shell</body></html>`), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	t.Setenv("DS2API_STATIC_ADMIN_DIR", staticDir)
	t.Setenv("DS2API_CONFIG_JSON", `{"keys":["k1"],"accounts":[{"email":"u@example.com","password":"p"}]}`)
	t.Setenv("DS2API_ENV_WRITEBACK", "0")

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/proxies", nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Dest", "document")
	rec := httptest.NewRecorder()
	app.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected WebUI shell, got status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("expected text/html content type, got %q", rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), "admin shell") {
		t.Fatalf("expected index.html response, got %s", rec.Body.String())
	}
}

func TestAdminProxiesAPIRequestsKeepAuthBehavior(t *testing.T) {
	t.Setenv("DS2API_CONFIG_JSON", `{"keys":["k1"],"accounts":[{"email":"u@example.com","password":"p"}],"proxies":[{"id":"proxy-1","name":"Node 1","type":"socks5h","host":"127.0.0.1","port":1080}]}`)
	t.Setenv("DS2API_ENV_WRITEBACK", "0")

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}

	unauthReq := httptest.NewRequest(http.MethodGet, "/admin/proxies", nil)
	unauthReq.Header.Set("Accept", "application/json")
	unauthRec := httptest.NewRecorder()
	app.Router.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated API status 401, got %d body=%s", unauthRec.Code, unauthRec.Body.String())
	}
	if !strings.Contains(unauthRec.Body.String(), "authentication required") {
		t.Fatalf("expected auth error detail, got %s", unauthRec.Body.String())
	}

	token := adminauth.AdminKey()
	authReq := httptest.NewRequest(http.MethodGet, "/admin/proxies", nil)
	authReq.Header.Set("Accept", "application/json")
	authReq.Header.Set("Authorization", "Bearer "+token)
	authRec := httptest.NewRecorder()
	app.Router.ServeHTTP(authRec, authReq)
	if authRec.Code != http.StatusOK {
		t.Fatalf("expected authenticated API status 200, got %d body=%s", authRec.Code, authRec.Body.String())
	}
	if !strings.Contains(authRec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("expected JSON content type, got %q", authRec.Header().Get("Content-Type"))
	}
	if !strings.Contains(authRec.Body.String(), `"items"`) || !strings.Contains(authRec.Body.String(), `"proxy-1"`) {
		t.Fatalf("expected proxy JSON payload, got %s", authRec.Body.String())
	}
}
