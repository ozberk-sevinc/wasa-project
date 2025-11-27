package test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ozberk-sevinc/wasa-project/service/api"
	"github.com/ozberk-sevinc/wasa-project/service/database"
	"github.com/sirupsen/logrus"
)

var (
	baseURL string
	server  *httptest.Server
)

// TestMain sets up the test server
func TestMain(m *testing.M) {
	// Remove old test database
	os.Remove("test_wasa.db")

	// Open SQLite connection
	sqlDB, err := sql.Open("sqlite3", "test_wasa.db")
	if err != nil {
		fmt.Printf("Failed to open SQLite: %v\n", err)
		os.Exit(1)
	}

	// Create database wrapper
	db, err := database.New(sqlDB)
	if err != nil {
		fmt.Printf("Failed to create database: %v\n", err)
		os.Exit(1)
	}

	// Create API
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	apirouter, err := api.New(api.Config{
		Logger:   logger,
		Database: db,
	})
	if err != nil {
		fmt.Printf("Failed to create API: %v\n", err)
		os.Exit(1)
	}

	// Create test server
	server = httptest.NewServer(apirouter.Handler())
	baseURL = server.URL

	fmt.Printf("Test server running at %s\n", baseURL)

	// testrun
	code := m.Run()

	// Cleanup
	server.Close()
	os.Remove("test_wasa.db")

	os.Exit(code)
}

// Helper functions
func doRequest(t *testing.T, method, path string, body interface{}, token string) *http.Response {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	return resp
}

func parseJSON(t *testing.T, resp *http.Response, v interface{}) {
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, v); err != nil {
		t.Fatalf("Failed to parse JSON response: %v, body: %s", err, string(body))
	}
}

// TEST: POST /session - Login/Register
func TestLogin_CreateNewUser(t *testing.T) {
	resp := doRequest(t, "POST", "/session", map[string]string{"name": "alice"}, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]string
	parseJSON(t, resp, &result)

	if result["identifier"] == "" {
		t.Fatal("Expected identifier in response")
	}
	t.Logf("✅ Created user 'alice' with ID: %s", result["identifier"])
}

func TestLogin_ExistingUser(t *testing.T) {
	// First login - creates user
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "bob"}, "")
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", resp1.StatusCode)
	}
	var result1 map[string]string
	parseJSON(t, resp1, &result1)
	firstID := result1["identifier"]

	// Second login - should return same user
	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "bob"}, "")
	if resp2.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", resp2.StatusCode)
	}
	var result2 map[string]string
	parseJSON(t, resp2, &result2)
	secondID := result2["identifier"]

	if firstID != secondID {
		t.Fatalf("Expected same ID, got %s and %s", firstID, secondID)
	}
	t.Logf("✅ Existing user 'bob' returned same ID: %s", firstID)
}

func TestLogin_InvalidUsername(t *testing.T) {
	// Too short
	resp := doRequest(t, "POST", "/session", map[string]string{"name": "ab"}, "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400 for short username, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Too long
	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "thisusernameiswaytoolong"}, "")
	if resp2.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400 for long username, got %d", resp2.StatusCode)
	}
	resp2.Body.Close()

	t.Log("✅ Invalid username validation works")
}

// ============================================================================
// TEST: GET /me - Get Current User
// ============================================================================

func TestGetMe_Success(t *testing.T) {
	// Create user first
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "charlie"}, "")
	var loginResult map[string]string
	parseJSON(t, resp1, &loginResult)
	token := loginResult["identifier"]

	// Get user profile
	resp := doRequest(t, "GET", "/me", nil, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var user map[string]interface{}
	parseJSON(t, resp, &user)

	if user["id"] != token {
		t.Fatalf("Expected ID %s, got %s", token, user["id"])
	}
	if user["name"] != "charlie" {
		t.Fatalf("Expected name 'charlie', got %s", user["name"])
	}
	t.Logf("✅ GET /me returns correct user: %v", user)
}

func TestGetMe_Unauthorized(t *testing.T) {
	resp := doRequest(t, "GET", "/me", nil, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected 401, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	t.Log("✅ GET /me without token returns 401")
}

func TestGetMe_InvalidToken(t *testing.T) {
	resp := doRequest(t, "GET", "/me", nil, "invalid-token-12345")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected 401, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	t.Log("✅ GET /me with invalid token returns 401")
}

// ============================================================================
// TEST: PUT /me/username - Change Username
// ============================================================================

func TestSetUsername_Success(t *testing.T) {
	// Create user
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "david"}, "")
	var loginResult map[string]string
	parseJSON(t, resp1, &loginResult)
	token := loginResult["identifier"]

	// Change username
	resp := doRequest(t, "PUT", "/me/username", map[string]string{"name": "david_new"}, token)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var user map[string]interface{}
	parseJSON(t, resp, &user)

	if user["name"] != "david_new" {
		t.Fatalf("Expected name 'david_new', got %s", user["name"])
	}
	t.Logf("✅ Username changed to: %s", user["name"])
}

func TestSetUsername_AlreadyTaken(t *testing.T) {
	// Create two users
	doRequest(t, "POST", "/session", map[string]string{"name": "eve"}, "").Body.Close()

	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "frank"}, "")
	var loginResult map[string]string
	parseJSON(t, resp2, &loginResult)
	frankToken := loginResult["identifier"]

	// Try to change frank's username to eve (already taken)
	resp := doRequest(t, "PUT", "/me/username", map[string]string{"name": "eve"}, frankToken)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("Expected 409 Conflict, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	t.Log("✅ Duplicate username returns 409 Conflict")
}

// ============================================================================
// TEST: GET /users - Search Users
// ============================================================================

func TestSearchUsers_All(t *testing.T) {
	// Create a user and get token
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "george"}, "")
	var loginResult map[string]string
	parseJSON(t, resp1, &loginResult)
	token := loginResult["identifier"]

	// Get all users
	resp := doRequest(t, "GET", "/users", nil, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string][]map[string]interface{}
	parseJSON(t, resp, &result)

	users := result["users"]
	if len(users) == 0 {
		t.Fatal("Expected at least one user")
	}
	t.Logf("✅ GET /users returns %d users", len(users))
}

func TestSearchUsers_ByQuery(t *testing.T) {
	// Create user
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "harry"}, "")
	var loginResult map[string]string
	parseJSON(t, resp1, &loginResult)
	token := loginResult["identifier"]

	// Search for 'har'
	resp := doRequest(t, "GET", "/users?q=har", nil, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string][]map[string]interface{}
	parseJSON(t, resp, &result)

	users := result["users"]
	found := false
	for _, u := range users {
		if u["name"] == "harry" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Expected to find 'harry' in search results")
	}
	t.Logf("✅ GET /users?q=har finds 'harry'")
}

func TestSearchUsers_Unauthorized(t *testing.T) {
	resp := doRequest(t, "GET", "/users", nil, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected 401, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	t.Log("✅ GET /users without token returns 401")
}

// ============================================================================
// TEST: GET /liveness - Health Check
// ============================================================================

func TestLiveness(t *testing.T) {
	resp := doRequest(t, "GET", "/liveness", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	t.Log("✅ GET /liveness returns 200")
}

// ============================================================================
// INTEGRATION TEST: Two Users Texting Each Other
// ============================================================================

func TestTwoUsersScenario(t *testing.T) {
	t.Log("=== Integration Test: Two Users Scenario ===")

	// 1. Create User 1 (Alice)
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "user_alice"}, "")
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create alice: %d", resp1.StatusCode)
	}
	var alice map[string]string
	parseJSON(t, resp1, &alice)
	aliceToken := alice["identifier"]
	t.Logf("👤 Created Alice with ID: %s", aliceToken)

	// 2. Create User 2 (Bob)
	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "user_bob"}, "")
	if resp2.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create bob: %d", resp2.StatusCode)
	}
	var bob map[string]string
	parseJSON(t, resp2, &bob)
	bobToken := bob["identifier"]
	t.Logf("👤 Created Bob with ID: %s", bobToken)

	// 3. Alice searches for Bob
	resp3 := doRequest(t, "GET", "/users?q=user_bob", nil, aliceToken)
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("Alice failed to search: %d", resp3.StatusCode)
	}
	var searchResult map[string][]map[string]interface{}
	parseJSON(t, resp3, &searchResult)
	if len(searchResult["users"]) == 0 {
		t.Fatal("Alice didn't find Bob")
	}
	t.Log("🔍 Alice found Bob in search results")

	// 4. Bob gets his profile
	resp4 := doRequest(t, "GET", "/me", nil, bobToken)
	if resp4.StatusCode != http.StatusOK {
		t.Fatalf("Bob failed to get profile: %d", resp4.StatusCode)
	}
	var bobProfile map[string]interface{}
	parseJSON(t, resp4, &bobProfile)
	if bobProfile["name"] != "user_bob" {
		t.Fatalf("Bob's name incorrect: %v", bobProfile["name"])
	}
	t.Log("👤 Bob is bob for sure")

	// 5. Alice changes her username
	resp5 := doRequest(t, "PUT", "/me/username", map[string]string{"name": "alice_pro"}, aliceToken)
	if resp5.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp5.Body)
		t.Fatalf("Alice failed to change username: %d - %s", resp5.StatusCode, string(body))
	}
	var aliceUpdated map[string]interface{}
	parseJSON(t, resp5, &aliceUpdated)
	if aliceUpdated["name"] != "alice_pro" {
		t.Fatalf("Alice's new name incorrect: %v", aliceUpdated["name"])
	}
	t.Log("✏️ Alice changed her username to 'alice_pro'")

	// 6. Bob searches for Alice's new name
	resp6 := doRequest(t, "GET", "/users?q=alice_pro", nil, bobToken)
	if resp6.StatusCode != http.StatusOK {
		t.Fatalf("Bob failed to search: %d", resp6.StatusCode)
	}
	var searchResult2 map[string][]map[string]interface{}
	parseJSON(t, resp6, &searchResult2)
	if len(searchResult2["users"]) == 0 {
		t.Fatal("Bob didn't find Alice's new username")
	}
	t.Log("🔍 Bob found Alice with new username")

	t.Log("=== ✅ Integration Test Passed! ===")
}
