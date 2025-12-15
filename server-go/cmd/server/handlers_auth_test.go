package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
}

// createTestAppState creates an AppState for testing
func createTestAppState(t *testing.T, password string) *AppState {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	return &AppState{
		Config: &AppConfig{
			AdminPasswordHash: string(hash),
		},
		ConfigMu: sync.RWMutex{},
	}
}

// TestLogin tests the login handler
func TestLogin(t *testing.T) {
	testPassword := "testpassword123"
	appState := createTestAppState(t, testPassword)

	t.Run("Successful login", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/login", appState.Login)

		body := LoginRequest{Password: testPassword}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response LoginResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response.Token == "" {
			t.Error("Expected token in response")
		}
	})

	t.Run("Invalid password", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/login", appState.Login)

		body := LoginRequest{Password: "wrongpassword"}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("Invalid JSON request", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/login", appState.Login)

		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("Empty password", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/login", appState.Login)

		body := LoginRequest{Password: ""}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})
}

// TestVerifyToken tests the token verification handler
func TestVerifyToken(t *testing.T) {
	appState := createTestAppState(t, "password")

	t.Run("Verify returns valid status", func(t *testing.T) {
		router := gin.New()
		router.GET("/api/verify", appState.VerifyToken)

		req := httptest.NewRequest("GET", "/api/verify", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["status"] != "valid" {
			t.Errorf("Expected status 'valid', got '%s'", response["status"])
		}
	})
}

// TestChangePassword tests the password change handler
func TestChangePassword(t *testing.T) {
	testPassword := "oldpassword"

	t.Run("Successful password change", func(t *testing.T) {
		appState := createTestAppState(t, testPassword)
		router := gin.New()
		router.POST("/api/change-password", appState.ChangePassword)

		body := ChangePasswordRequest{
			CurrentPassword: testPassword,
			NewPassword:     "newpassword123",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Verify new password works
		err := bcrypt.CompareHashAndPassword([]byte(appState.Config.AdminPasswordHash), []byte("newpassword123"))
		if err != nil {
			t.Error("New password should be valid after change")
		}
	})

	t.Run("Wrong current password", func(t *testing.T) {
		appState := createTestAppState(t, testPassword)
		router := gin.New()
		router.POST("/api/change-password", appState.ChangePassword)

		body := ChangePasswordRequest{
			CurrentPassword: "wrongpassword",
			NewPassword:     "newpassword123",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("Invalid JSON request", func(t *testing.T) {
		appState := createTestAppState(t, testPassword)
		router := gin.New()
		router.POST("/api/change-password", appState.ChangePassword)

		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

// BenchmarkLogin benchmarks the login handler
func BenchmarkLogin(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	password := "benchmarkpassword"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	appState := &AppState{
		Config: &AppConfig{
			AdminPasswordHash: string(hash),
		},
	}

	router := gin.New()
	router.POST("/api/login", appState.Login)

	body := LoginRequest{Password: password}
	jsonBody, _ := json.Marshal(body)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

