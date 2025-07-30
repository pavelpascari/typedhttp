package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// Enhanced request struct demonstrating multi-source capabilities
type EnhancedAPIRequest struct {
	// Path parameters
	UserID string `path:"id" validate:"required"`
	Action string `path:"action" validate:"required"`

	// Multi-source authentication with precedence
	AuthToken string `header:"Authorization" cookie:"auth_token" precedence:"header,cookie"`
	SessionID string `cookie:"session_id" header:"X-Session-ID" precedence:"cookie,header"`

	// Query parameters with defaults
	Page  int    `query:"page" default:"1" validate:"min=1"`
	Limit int    `query:"limit" default:"20" validate:"min=1,max=100"`
	Sort  string `query:"sort" default:"created_at"`

	// Headers with transformations
	ClientIP  net.IP `header:"X-Forwarded-For" transform:"first_ip"`
	UserAgent string `header:"User-Agent"`

	// Form data (for POST requests)
	Name        string                `form:"name" json:"name" precedence:"form,json"`
	Email       string                `form:"email" json:"email" precedence:"form,json"`
	Avatar      *multipart.FileHeader `form:"avatar"`
	Description string                `form:"description"`

	// Language preference from multiple sources
	Language string `cookie:"lang" header:"Accept-Language" default:"en" precedence:"cookie,header"`
}

type EnhancedAPIResponse struct {
	Status    string                 `json:"status"`
	Message   string                 `json:"message"`
	UserID    string                 `json:"user_id"`
	Data      map[string]interface{} `json:"data"`
	RequestID string                 `json:"request_id"`
}

type EnhancedAPIHandler struct{}

func (h *EnhancedAPIHandler) Handle(ctx context.Context, req EnhancedAPIRequest) (EnhancedAPIResponse, error) {
	fmt.Printf("=== Enhanced API Request Processing ===\n")
	fmt.Printf("User ID: %s\n", req.UserID)
	fmt.Printf("Action: %s\n", req.Action)
	fmt.Printf("Auth Token: %s\n", req.AuthToken)
	fmt.Printf("Session ID: %s\n", req.SessionID)
	fmt.Printf("Page: %d, Limit: %d, Sort: %s\n", req.Page, req.Limit, req.Sort)
	fmt.Printf("Client IP: %s\n", req.ClientIP)
	fmt.Printf("Language: %s\n", req.Language)
	fmt.Printf("User Agent: %s\n", req.UserAgent)

	if req.Name != "" {
		fmt.Printf("Name: %s\n", req.Name)
	}
	if req.Email != "" {
		fmt.Printf("Email: %s\n", req.Email)
	}
	if req.Avatar != nil {
		fmt.Printf("Avatar: %s (%d bytes)\n", req.Avatar.Filename, req.Avatar.Size)
	}
	if req.Description != "" {
		fmt.Printf("Description: %s\n", req.Description)
	}

	return EnhancedAPIResponse{
		Status:    "success",
		Message:   "Enhanced request processed successfully",
		UserID:    req.UserID,
		RequestID: "req-123456",
		Data: map[string]interface{}{
			"action":      req.Action,
			"client_ip":   req.ClientIP.String(),
			"language":    req.Language,
			"auth_source": determineAuthSource(req),
		},
	}, nil
}

func determineAuthSource(req EnhancedAPIRequest) string {
	if req.AuthToken != "" && req.SessionID != "" {
		return "both_header_and_cookie"
	} else if req.AuthToken != "" {
		return "header_auth"
	} else if req.SessionID != "" {
		return "cookie_session"
	}
	return "none"
}

func main() {
	fmt.Println("=== TypedHTTP Enhanced Multi-Source Example ===\n")

	// Create router with enhanced decoder
	router := typedhttp.NewRouter()

	// Register the enhanced handler
	typedhttp.POST(router, "/api/users/{id}/{action}", &EnhancedAPIHandler{})

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	fmt.Printf("Test server running at: %s\n\n", server.URL)

	// Example 1: Multi-source authentication (header precedence)
	fmt.Println("=== Example 1: Header Auth Token (precedence over cookie) ===")
	testMultiSourceAuth(server.URL, true, true)

	// Example 2: Cookie-only authentication (fallback)
	fmt.Println("\n=== Example 2: Cookie Session Only (header fallback) ===")
	testMultiSourceAuth(server.URL, false, true)

	// Example 3: Query parameters with defaults
	fmt.Println("\n=== Example 3: Query Parameters with Defaults ===")
	testQueryDefaults(server.URL)

	// Example 4: Form data with file upload
	fmt.Println("\n=== Example 4: Form Data with File Upload ===")
	testFormWithFile(server.URL)

	// Example 5: Language preference from cookie vs header
	fmt.Println("\n=== Example 5: Language Preference (cookie precedence) ===")
	testLanguagePreference(server.URL)
}

func testMultiSourceAuth(baseURL string, includeHeader, includeCookie bool) {
	req, _ := http.NewRequest("POST", baseURL+"/api/users/user123/update", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")

	if includeHeader {
		req.Header.Set("Authorization", "Bearer header-token-abc123")
	}
	if includeCookie {
		req.AddCookie(&http.Cookie{Name: "auth_token", Value: "cookie-token-xyz789"})
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-456"})
	}

	// Add client info
	req.Header.Set("X-Forwarded-For", "192.168.1.100, 10.0.0.1")
	req.Header.Set("User-Agent", "TestClient/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Response Status: %s\n", resp.Status)
}

func testQueryDefaults(baseURL string) {
	// Request without query parameters (should use defaults)
	req, _ := http.NewRequest("POST", baseURL+"/api/users/user456/view", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-789"})

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Response Status: %s\n", resp.Status)

	// Request with custom query parameters
	req2, _ := http.NewRequest("POST", baseURL+"/api/users/user456/view?page=3&limit=50&sort=updated_at", nil)
	req2.Header.Set("X-Forwarded-For", "203.0.113.1")
	req2.AddCookie(&http.Cookie{Name: "session_id", Value: "session-789"})

	resp2, err := client.Do(req2)
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}
	defer resp2.Body.Close()

	fmt.Printf("Response Status (with params): %s\n", resp2.Status)
}

func testFormWithFile(baseURL string) {
	// Create multipart form with file
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add form fields
	writer.WriteField("name", "John Doe")
	writer.WriteField("email", "john@example.com")
	writer.WriteField("description", "Updated user profile with avatar")

	// Add file
	fileWriter, err := writer.CreateFormFile("avatar", "profile.jpg")
	if err != nil {
		log.Printf("Failed to create form file: %v", err)
		return
	}
	fileWriter.Write([]byte("fake image data for testing"))

	writer.Close()

	req, _ := http.NewRequest("POST", baseURL+"/api/users/user789/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer form-upload-token")
	req.Header.Set("X-Forwarded-For", "198.51.100.42")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Response Status: %s\n", resp.Status)
}

func testLanguagePreference(baseURL string) {
	req, _ := http.NewRequest("POST", baseURL+"/api/users/user999/settings", nil)

	// Cookie should take precedence over header
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.AddCookie(&http.Cookie{Name: "lang", Value: "es"})
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-lang-test"})
	req.Header.Set("X-Forwarded-For", "192.0.2.1")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Response Status: %s\n", resp.Status)
}

// Additional helper to show URL form encoding
func testURLEncodedForm(baseURL string) {
	form := url.Values{}
	form.Add("name", "Jane Smith")
	form.Add("email", "jane@example.com")
	form.Add("description", "URL encoded form submission")

	req, _ := http.NewRequest("POST", baseURL+"/api/users/form123/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer url-form-token")
	req.Header.Set("X-Forwarded-For", "10.0.0.50")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("URL Form Response Status: %s\n", resp.Status)
}