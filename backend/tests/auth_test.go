package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestAuth(t *testing.T) {
	h := NewTestHelper()
	defer h.Close()

	t.Run("HealthCheck", func(t *testing.T) {
		resp, err := h.App.Test(newHTTPRequest("GET", "/api/health", nil, nil))
		if err != nil {
			t.Fatalf("❌ Erreur requête: %v", err)
		}
		result := MustParseResponse(resp)
		AssertSuccess(t, result)
	})

	t.Run("DevLogin_Success", func(t *testing.T) {
		body := map[string]string{"email": "admin@test.dz"}
		resp, err := h.App.Test(newHTTPRequest("POST", "/api/auth/dev-login", body, nil))
		if err != nil {
			t.Fatalf("❌ Erreur requête: %v", err)
		}
		result := MustParseResponse(resp)
		AssertSuccess(t, result)

		data, ok := result["data"].(map[string]any)
		if !ok {
			t.Fatalf("❌ Champ data manquant ou invalide")
		}
		token, ok := data["token"].(string)
		if !ok || token == "" {
			t.Fatalf("❌ Token manquant ou vide")
		}
	})

	t.Run("DevLogin_NotFound", func(t *testing.T) {
		body := map[string]string{"email": "nonexistent@test.dz"}
		resp, err := h.App.Test(newHTTPRequest("POST", "/api/auth/dev-login", body, nil))
		if err != nil {
			t.Fatalf("❌ Erreur requête: %v", err)
		}
		result := MustParseResponse(resp)
		AssertErrorContains(t, result, "trouvé")
	})

	t.Run("DevLogin_InvalidEmail", func(t *testing.T) {
		body := map[string]string{"email": ""}
		resp, err := h.App.Test(newHTTPRequest("POST", "/api/auth/dev-login", body, nil))
		if err != nil {
			t.Fatalf("❌ Erreur requête: %v", err)
		}
		result := MustParseResponse(resp)
		AssertError(t, result)
	})

	t.Run("Me_Success", func(t *testing.T) {
		resp, err := h.App.Test(newHTTPRequest("GET", "/api/auth/me", nil, h.AuthHeader(SeedAdminID, "admin")))
		if err != nil {
			t.Fatalf("❌ Erreur requête: %v", err)
		}
		result := MustParseResponse(resp)
		AssertSuccess(t, result)

		data, ok := result["data"].(map[string]any)
		if !ok {
			t.Fatalf("❌ Champ data manquant")
		}
		if data["email"] != "admin@test.dz" {
			t.Fatalf("❌ Email incorrect: attendu admin@test.dz, obtenu %v", data["email"])
		}
	})

	t.Run("Me_NoAuth", func(t *testing.T) {
		resp, err := h.App.Test(newHTTPRequest("GET", "/api/auth/me", nil, nil))
		if err != nil {
			t.Fatalf("❌ Erreur requête: %v", err)
		}
		result := MustParseResponse(resp)
		AssertErrorContains(t, result, "Authentification requise")
	})

	t.Run("Me_InvalidToken", func(t *testing.T) {
		headers := map[string]string{"Authorization": "Bearer invalid-token"}
		resp, err := h.App.Test(newHTTPRequest("GET", "/api/auth/me", nil, headers))
		if err != nil {
			t.Fatalf("❌ Erreur requête: %v", err)
		}
		result := MustParseResponse(resp)
		AssertErrorContains(t, result, "Token invalide")
	})

	t.Run("Logout", func(t *testing.T) {
		resp, err := h.App.Test(newHTTPRequest("POST", "/api/auth/logout", nil, h.AuthHeader(SeedAdminID, "admin")))
		if err != nil {
			t.Fatalf("❌ Erreur requête: %v", err)
		}
		result := MustParseResponse(resp)
		AssertSuccess(t, result)
	})

	t.Run("Access_Unauthorized_Role", func(t *testing.T) {

		resp, err := h.App.Test(newHTTPRequest("GET", "/api/admin/dashboard", nil, h.AuthHeader(SeedStudentISIL1ID, "student")))
		if err != nil {
			t.Fatalf("❌ Erreur requête: %v", err)
		}
		result := MustParseResponse(resp)
		AssertErrorContains(t, result, "non autorisé")
	})

	t.Run("Access_NoAuth_Protected", func(t *testing.T) {
		resp, err := h.App.Test(newHTTPRequest("GET", "/api/admin/dashboard", nil, nil))
		if err != nil {
			t.Fatalf("❌ Erreur requête: %v", err)
		}
		result := MustParseResponse(resp)
		AssertErrorContains(t, result, "Authentification")
	})
}

func newHTTPRequest(method, url string, body any, headers map[string]string) *http.Request {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}

	req, _ := http.NewRequest(method, url, bytes.NewReader(reqBody))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}
	return req
}

func TestAuthNoAuth(t *testing.T) {
	h := NewTestHelper()
	defer h.Close()

	protectedEndpoints := []string{
		"/api/auth/me",
		"/api/admin/dashboard",
		"/api/teacher/dashboard",
		"/api/student/dashboard",
		"/api/company/dashboard",
		"/api/upload/avatar",
	}

	for _, endpoint := range protectedEndpoints {
		t.Run("NoAuth_"+strings.ReplaceAll(endpoint, "/", "_"), func(t *testing.T) {
			resp, err := h.App.Test(newHTTPRequest("GET", endpoint, nil, nil))
			if err != nil {
				t.Fatalf("❌ Erreur requête: %v", err)
			}
			result := MustParseResponse(resp)
			if result["success"] == true {
				t.Fatalf("❌ L'endpoint %s aurait dû être protégé mais a retourné success=true", endpoint)
			}
		})
	}
}
