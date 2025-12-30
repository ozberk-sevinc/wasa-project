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

// ============================================================================
// TEST: POST /conversations - Start a Direct Conversation
// ============================================================================

func TestCreateConversation_Success(t *testing.T) {
	// Create two users
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "conv_user1"}, "")
	var user1 map[string]string
	parseJSON(t, resp1, &user1)
	user1Token := user1["identifier"]

	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "conv_user2"}, "")
	var user2 map[string]string
	parseJSON(t, resp2, &user2)
	user2ID := user2["identifier"]

	// User1 starts a conversation with User2
	resp := doRequest(t, "POST", "/conversations", map[string]string{"userId": user2ID}, user1Token)
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(body))
	}

	var conv map[string]interface{}
	parseJSON(t, resp, &conv)

	if conv["id"] == "" {
		t.Fatal("Expected conversation ID")
	}
	if conv["type"] != "direct" {
		t.Fatalf("Expected type 'direct', got %s", conv["type"])
	}
	t.Logf("✅ Created conversation: %s", conv["id"])
}

func TestCreateConversation_AlreadyExists(t *testing.T) {
	// Create two users
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "existing1"}, "")
	var user1 map[string]string
	parseJSON(t, resp1, &user1)
	user1Token := user1["identifier"]

	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "existing2"}, "")
	var user2 map[string]string
	parseJSON(t, resp2, &user2)
	user2ID := user2["identifier"]

	// First conversation creation
	resp3 := doRequest(t, "POST", "/conversations", map[string]string{"userId": user2ID}, user1Token)
	var conv1 map[string]interface{}
	parseJSON(t, resp3, &conv1)
	firstConvID := conv1["id"]

	// Second attempt - should return existing
	resp4 := doRequest(t, "POST", "/conversations", map[string]string{"userId": user2ID}, user1Token)
	if resp4.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 for existing conversation, got %d", resp4.StatusCode)
	}

	var conv2 map[string]interface{}
	parseJSON(t, resp4, &conv2)

	if conv2["id"] != firstConvID {
		t.Fatalf("Expected same conversation ID %s, got %s", firstConvID, conv2["id"])
	}
	t.Log("✅ Returns existing conversation on duplicate request")
}

func TestCreateConversation_WithSelf(t *testing.T) {
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "selfuser"}, "")
	var user1 map[string]string
	parseJSON(t, resp1, &user1)
	userToken := user1["identifier"]

	// Create conversation with self (Message Yourself feature)
	resp := doRequest(t, "POST", "/conversations", map[string]string{"userId": userToken}, userToken)
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201 for self-conversation, got %d: %s", resp.StatusCode, string(body))
	}

	var conv map[string]interface{}
	parseJSON(t, resp, &conv)

	if conv["title"] != "Message Yourself" {
		t.Fatalf("Expected title 'Message Yourself', got %s", conv["title"])
	}

	// Should only have 1 participant (yourself)
	participants := conv["participants"].([]interface{})
	if len(participants) != 1 {
		t.Fatalf("Expected 1 participant for self-conversation, got %d", len(participants))
	}

	t.Log("✅ Created 'Message Yourself' conversation")
}

func TestCreateConversation_UserNotFound(t *testing.T) {
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "orphanuser"}, "")
	var user1 map[string]string
	parseJSON(t, resp1, &user1)
	userToken := user1["identifier"]

	// Try to create conversation with non-existent user
	resp := doRequest(t, "POST", "/conversations", map[string]string{"userId": "nonexistent-user-id"}, userToken)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected 404 for non-existent user, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	t.Log("✅ Returns 404 for non-existent target user")
}

// ============================================================================
// TEST: POST /conversations/{id}/messages - Send Message
// ============================================================================

func TestSendMessage_Text(t *testing.T) {
	// Create two users
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "msg_sender"}, "")
	var user1 map[string]string
	parseJSON(t, resp1, &user1)
	senderToken := user1["identifier"]

	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "msg_receiver"}, "")
	var user2 map[string]string
	parseJSON(t, resp2, &user2)
	receiverID := user2["identifier"]

	// Create conversation
	resp3 := doRequest(t, "POST", "/conversations", map[string]string{"userId": receiverID}, senderToken)
	var conv map[string]interface{}
	parseJSON(t, resp3, &conv)
	convID := conv["id"].(string)

	// Send text message
	msgBody := map[string]interface{}{
		"contentType": "text",
		"text":        "Hello, World!",
	}
	resp := doRequest(t, "POST", "/conversations/"+convID+"/messages", msgBody, senderToken)
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(body))
	}

	var msg map[string]interface{}
	parseJSON(t, resp, &msg)

	if msg["id"] == "" {
		t.Fatal("Expected message ID")
	}
	if msg["contentType"] != "text" {
		t.Fatalf("Expected contentType 'text', got %s", msg["contentType"])
	}
	if msg["text"] != "Hello, World!" {
		t.Fatalf("Expected text 'Hello, World!', got %s", msg["text"])
	}
	if msg["status"] != "sent" {
		t.Fatalf("Expected status 'sent', got %s", msg["status"])
	}
	t.Logf("✅ Sent text message: %s", msg["id"])
}

func TestSendMessage_Photo(t *testing.T) {
	// Create two users and conversation
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "photo_sender"}, "")
	var user1 map[string]string
	parseJSON(t, resp1, &user1)
	senderToken := user1["identifier"]

	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "photo_receiver"}, "")
	var user2 map[string]string
	parseJSON(t, resp2, &user2)
	receiverID := user2["identifier"]

	resp3 := doRequest(t, "POST", "/conversations", map[string]string{"userId": receiverID}, senderToken)
	var conv map[string]interface{}
	parseJSON(t, resp3, &conv)
	convID := conv["id"].(string)

	// Send photo message
	msgBody := map[string]interface{}{
		"contentType": "photo",
		"photoUrl":    "https://example.com/photo.jpg",
	}
	resp := doRequest(t, "POST", "/conversations/"+convID+"/messages", msgBody, senderToken)
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(body))
	}

	var msg map[string]interface{}
	parseJSON(t, resp, &msg)

	if msg["contentType"] != "photo" {
		t.Fatalf("Expected contentType 'photo', got %s", msg["contentType"])
	}
	if msg["photoUrl"] != "https://example.com/photo.jpg" {
		t.Fatalf("Expected photoUrl, got %s", msg["photoUrl"])
	}
	t.Log("✅ Sent photo message")
}

func TestSendMessage_Reply(t *testing.T) {
	// Create users and conversation
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "reply_user1"}, "")
	var user1 map[string]string
	parseJSON(t, resp1, &user1)
	user1Token := user1["identifier"]

	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "reply_user2"}, "")
	var user2 map[string]string
	parseJSON(t, resp2, &user2)
	user2ID := user2["identifier"]

	resp3 := doRequest(t, "POST", "/conversations", map[string]string{"userId": user2ID}, user1Token)
	var conv map[string]interface{}
	parseJSON(t, resp3, &conv)
	convID := conv["id"].(string)

	// Send original message
	msg1Body := map[string]interface{}{
		"contentType": "text",
		"text":        "Original message",
	}
	resp4 := doRequest(t, "POST", "/conversations/"+convID+"/messages", msg1Body, user1Token)
	var originalMsg map[string]interface{}
	parseJSON(t, resp4, &originalMsg)
	originalMsgID := originalMsg["id"].(string)

	// Send reply
	replyBody := map[string]interface{}{
		"contentType":      "text",
		"text":             "This is a reply",
		"replyToMessageId": originalMsgID,
	}
	resp := doRequest(t, "POST", "/conversations/"+convID+"/messages", replyBody, user1Token)
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(body))
	}

	var replyMsg map[string]interface{}
	parseJSON(t, resp, &replyMsg)

	if replyMsg["repliedToMessageId"] != originalMsgID {
		t.Fatalf("Expected repliedToMessageId %s, got %v", originalMsgID, replyMsg["repliedToMessageId"])
	}
	t.Log("✅ Sent reply message with reference to original")
}

func TestSendMessage_InvalidContentType(t *testing.T) {
	// Create users and conversation
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "invalid_ct_user1"}, "")
	var user1 map[string]string
	parseJSON(t, resp1, &user1)
	user1Token := user1["identifier"]

	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "invalid_ct_user2"}, "")
	var user2 map[string]string
	parseJSON(t, resp2, &user2)
	user2ID := user2["identifier"]

	resp3 := doRequest(t, "POST", "/conversations", map[string]string{"userId": user2ID}, user1Token)
	var conv map[string]interface{}
	parseJSON(t, resp3, &conv)
	convID := conv["id"].(string)

	// Send with invalid content type
	msgBody := map[string]interface{}{
		"contentType": "video",
		"text":        "Test",
	}
	resp := doRequest(t, "POST", "/conversations/"+convID+"/messages", msgBody, user1Token)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400 for invalid contentType, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	t.Log("✅ Invalid contentType returns 400")
}

func TestSendMessage_NotParticipant(t *testing.T) {
	// Create three users
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "notpart_user1"}, "")
	var user1 map[string]string
	parseJSON(t, resp1, &user1)
	user1Token := user1["identifier"]

	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "notpart_user2"}, "")
	var user2 map[string]string
	parseJSON(t, resp2, &user2)
	user2ID := user2["identifier"]

	resp3 := doRequest(t, "POST", "/session", map[string]string{"name": "notpart_user3"}, "")
	var user3 map[string]string
	parseJSON(t, resp3, &user3)
	user3Token := user3["identifier"]

	// User1 creates conversation with User2
	resp4 := doRequest(t, "POST", "/conversations", map[string]string{"userId": user2ID}, user1Token)
	var conv map[string]interface{}
	parseJSON(t, resp4, &conv)
	convID := conv["id"].(string)

	// User3 tries to send message (not a participant)
	msgBody := map[string]interface{}{
		"contentType": "text",
		"text":        "Sneaky message",
	}
	resp := doRequest(t, "POST", "/conversations/"+convID+"/messages", msgBody, user3Token)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected 404 for non-participant, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	t.Log("✅ Non-participant cannot send message")
}

// ============================================================================
// TEST: DELETE /conversations/{id}/messages/{id} - Delete Message
// ============================================================================

func TestDeleteMessage_Success(t *testing.T) {
	// Create users and conversation
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "del_user1"}, "")
	var user1 map[string]string
	parseJSON(t, resp1, &user1)
	user1Token := user1["identifier"]

	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "del_user2"}, "")
	var user2 map[string]string
	parseJSON(t, resp2, &user2)
	user2ID := user2["identifier"]

	resp3 := doRequest(t, "POST", "/conversations", map[string]string{"userId": user2ID}, user1Token)
	var conv map[string]interface{}
	parseJSON(t, resp3, &conv)
	convID := conv["id"].(string)

	// Send message
	msgBody := map[string]interface{}{
		"contentType": "text",
		"text":        "Message to delete",
	}
	resp4 := doRequest(t, "POST", "/conversations/"+convID+"/messages", msgBody, user1Token)
	var msg map[string]interface{}
	parseJSON(t, resp4, &msg)
	msgID := msg["id"].(string)

	// Delete message
	resp := doRequest(t, "DELETE", "/conversations/"+convID+"/messages/"+msgID, nil, user1Token)
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 204, got %d: %s", resp.StatusCode, string(body))
	}
	resp.Body.Close()
	t.Log("✅ Message deleted successfully")
}

func TestDeleteMessage_NotOwner(t *testing.T) {
	// Create users and conversation
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "delown_user1"}, "")
	var user1 map[string]string
	parseJSON(t, resp1, &user1)
	user1Token := user1["identifier"]

	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "delown_user2"}, "")
	var user2 map[string]string
	parseJSON(t, resp2, &user2)
	user2ID := user2["identifier"]
	user2Token := user2["identifier"]

	resp3 := doRequest(t, "POST", "/conversations", map[string]string{"userId": user2ID}, user1Token)
	var conv map[string]interface{}
	parseJSON(t, resp3, &conv)
	convID := conv["id"].(string)

	// User1 sends message
	msgBody := map[string]interface{}{
		"contentType": "text",
		"text":        "User1's message",
	}
	resp4 := doRequest(t, "POST", "/conversations/"+convID+"/messages", msgBody, user1Token)
	var msg map[string]interface{}
	parseJSON(t, resp4, &msg)
	msgID := msg["id"].(string)

	// User2 tries to delete User1's message
	resp := doRequest(t, "DELETE", "/conversations/"+convID+"/messages/"+msgID, nil, user2Token)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	t.Log("✅ Cannot delete other user's message")
}

// ============================================================================
// TEST: Message Status (Checkmarks)
// ============================================================================

func TestMessageStatus_ReceivedAndRead(t *testing.T) {
	t.Log("=== Testing Message Status Flow ===")

	// Create two users
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "status_sender"}, "")
	var sender map[string]string
	parseJSON(t, resp1, &sender)
	senderToken := sender["identifier"]

	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "status_receiver"}, "")
	var receiver map[string]string
	parseJSON(t, resp2, &receiver)
	receiverID := receiver["identifier"]
	receiverToken := receiver["identifier"]

	// Sender creates conversation
	resp3 := doRequest(t, "POST", "/conversations", map[string]string{"userId": receiverID}, senderToken)
	var conv map[string]interface{}
	parseJSON(t, resp3, &conv)
	convID := conv["id"].(string)

	// Sender sends message
	msgBody := map[string]interface{}{
		"contentType": "text",
		"text":        "Check my status!",
	}
	resp4 := doRequest(t, "POST", "/conversations/"+convID+"/messages", msgBody, senderToken)
	var msg map[string]interface{}
	parseJSON(t, resp4, &msg)
	msgID := msg["id"].(string)

	// Verify initial status is "sent"
	if msg["status"] != "sent" {
		t.Fatalf("Expected initial status 'sent', got %s", msg["status"])
	}
	t.Log("📤 Message sent with status 'sent'")

	// Receiver fetches conversation list -> should mark as "received"
	resp5 := doRequest(t, "GET", "/conversations", nil, receiverToken)
	if resp5.StatusCode != http.StatusOK {
		t.Fatalf("Failed to get conversations: %d", resp5.StatusCode)
	}
	resp5.Body.Close()
	t.Log("📬 Receiver fetched conversation list")

	// Sender checks conversation to see updated status
	resp6 := doRequest(t, "GET", "/conversations/"+convID, nil, senderToken)
	var convDetails map[string]interface{}
	parseJSON(t, resp6, &convDetails)
	messages := convDetails["messages"].([]interface{})

	var foundMsg map[string]interface{}
	for _, m := range messages {
		msgMap := m.(map[string]interface{})
		if msgMap["id"] == msgID {
			foundMsg = msgMap
			break
		}
	}
	if foundMsg == nil {
		t.Fatal("Message not found in conversation")
	}
	if foundMsg["status"] != "received" {
		t.Fatalf("Expected status 'received' after list fetch, got %s", foundMsg["status"])
	}
	t.Log("✓ Message status updated to 'received' (one checkmark)")

	// Receiver opens the conversation -> should mark as "read"
	resp7 := doRequest(t, "GET", "/conversations/"+convID, nil, receiverToken)
	if resp7.StatusCode != http.StatusOK {
		t.Fatalf("Failed to get conversation: %d", resp7.StatusCode)
	}
	resp7.Body.Close()
	t.Log("📖 Receiver opened the conversation")

	// Sender checks again to see "read" status
	resp8 := doRequest(t, "GET", "/conversations/"+convID, nil, senderToken)
	var convDetails2 map[string]interface{}
	parseJSON(t, resp8, &convDetails2)
	messages2 := convDetails2["messages"].([]interface{})

	for _, m := range messages2 {
		msgMap := m.(map[string]interface{})
		if msgMap["id"] == msgID {
			foundMsg = msgMap
			break
		}
	}
	if foundMsg["status"] != "read" {
		t.Fatalf("Expected status 'read' after conversation open, got %s", foundMsg["status"])
	}
	t.Log("✓✓ Message status updated to 'read' (two checkmarks)")

	t.Log("=== ✅ Message Status Test Passed! ===")
}

// ============================================================================
// TEST: POST .../messages/{id}/comments - Add Reaction
// ============================================================================

func TestCommentMessage_Success(t *testing.T) {
	// Create users and conversation
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "react_user1"}, "")
	var user1 map[string]string
	parseJSON(t, resp1, &user1)
	user1Token := user1["identifier"]

	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "react_user2"}, "")
	var user2 map[string]string
	parseJSON(t, resp2, &user2)
	user2ID := user2["identifier"]
	user2Token := user2["identifier"]

	resp3 := doRequest(t, "POST", "/conversations", map[string]string{"userId": user2ID}, user1Token)
	var conv map[string]interface{}
	parseJSON(t, resp3, &conv)
	convID := conv["id"].(string)

	// User1 sends message
	msgBody := map[string]interface{}{
		"contentType": "text",
		"text":        "React to this!",
	}
	resp4 := doRequest(t, "POST", "/conversations/"+convID+"/messages", msgBody, user1Token)
	var msg map[string]interface{}
	parseJSON(t, resp4, &msg)
	msgID := msg["id"].(string)

	// User2 reacts with emoji
	reactionBody := map[string]string{"emoji": "👍"}
	resp := doRequest(t, "POST", "/conversations/"+convID+"/messages/"+msgID+"/comments", reactionBody, user2Token)
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(body))
	}

	var reaction map[string]interface{}
	parseJSON(t, resp, &reaction)

	if reaction["emoji"] != "👍" {
		t.Fatalf("Expected emoji '👍', got %s", reaction["emoji"])
	}
	t.Log("✅ Added reaction to message")
}

// ============================================================================
// TEST: DELETE .../comments/{id} - Remove Reaction
// ============================================================================

func TestUncommentMessage_Success(t *testing.T) {
	// Create users, conversation, and message
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "unreact_user1"}, "")
	var user1 map[string]string
	parseJSON(t, resp1, &user1)
	user1Token := user1["identifier"]

	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "unreact_user2"}, "")
	var user2 map[string]string
	parseJSON(t, resp2, &user2)
	user2ID := user2["identifier"]
	user2Token := user2["identifier"]

	resp3 := doRequest(t, "POST", "/conversations", map[string]string{"userId": user2ID}, user1Token)
	var conv map[string]interface{}
	parseJSON(t, resp3, &conv)
	convID := conv["id"].(string)

	msgBody := map[string]interface{}{
		"contentType": "text",
		"text":        "React then unreact!",
	}
	resp4 := doRequest(t, "POST", "/conversations/"+convID+"/messages", msgBody, user1Token)
	var msg map[string]interface{}
	parseJSON(t, resp4, &msg)
	msgID := msg["id"].(string)

	// Add reaction
	reactionBody := map[string]string{"emoji": "❤️"}
	resp5 := doRequest(t, "POST", "/conversations/"+convID+"/messages/"+msgID+"/comments", reactionBody, user2Token)
	var reaction map[string]interface{}
	parseJSON(t, resp5, &reaction)
	reactionID := reaction["id"].(string)

	// Remove reaction
	resp := doRequest(t, "DELETE", "/conversations/"+convID+"/messages/"+msgID+"/comments/"+reactionID, nil, user2Token)
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 204, got %d: %s", resp.StatusCode, string(body))
	}
	resp.Body.Close()
	t.Log("✅ Removed reaction from message")
}

// ============================================================================
// TEST: POST .../messages/{id}/forward - Forward Message
// ============================================================================

func TestForwardMessage_Success(t *testing.T) {
	// Create three users
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "fwd_user1"}, "")
	var user1 map[string]string
	parseJSON(t, resp1, &user1)
	user1Token := user1["identifier"]

	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "fwd_user2"}, "")
	var user2 map[string]string
	parseJSON(t, resp2, &user2)
	user2ID := user2["identifier"]

	resp3 := doRequest(t, "POST", "/session", map[string]string{"name": "fwd_user3"}, "")
	var user3 map[string]string
	parseJSON(t, resp3, &user3)
	user3ID := user3["identifier"]

	// Create conversation 1 (user1 <-> user2)
	resp4 := doRequest(t, "POST", "/conversations", map[string]string{"userId": user2ID}, user1Token)
	var conv1 map[string]interface{}
	parseJSON(t, resp4, &conv1)
	conv1ID := conv1["id"].(string)

	// Create conversation 2 (user1 <-> user3)
	resp5 := doRequest(t, "POST", "/conversations", map[string]string{"userId": user3ID}, user1Token)
	var conv2 map[string]interface{}
	parseJSON(t, resp5, &conv2)
	conv2ID := conv2["id"].(string)

	// User1 sends message in conv1
	msgBody := map[string]interface{}{
		"contentType": "text",
		"text":        "Forward this message!",
	}
	resp6 := doRequest(t, "POST", "/conversations/"+conv1ID+"/messages", msgBody, user1Token)
	var originalMsg map[string]interface{}
	parseJSON(t, resp6, &originalMsg)
	originalMsgID := originalMsg["id"].(string)

	// User1 forwards message to conv2
	forwardBody := map[string]string{"targetConversationId": conv2ID}
	resp := doRequest(t, "POST", "/conversations/"+conv1ID+"/messages/"+originalMsgID+"/forward", forwardBody, user1Token)
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(body))
	}

	var forwardedMsg map[string]interface{}
	parseJSON(t, resp, &forwardedMsg)

	if forwardedMsg["text"] != "Forward this message!" {
		t.Fatalf("Forwarded message text mismatch")
	}
	if forwardedMsg["conversationId"] != conv2ID {
		t.Fatalf("Expected conversationId %s, got %s", conv2ID, forwardedMsg["conversationId"])
	}
	t.Log("✅ Message forwarded to another conversation")
}

// ============================================================================
// TEST: GET /conversations - Get My Conversations
// ============================================================================

func TestGetMyConversations_Success(t *testing.T) {
	// Create user
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "list_user"}, "")
	var user1 map[string]string
	parseJSON(t, resp1, &user1)
	user1Token := user1["identifier"]

	// Get conversations (might be empty initially)
	resp := doRequest(t, "GET", "/conversations", nil, user1Token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var convs []map[string]interface{}
	parseJSON(t, resp, &convs)

	// Should return an array (possibly empty)
	t.Logf("✅ GET /conversations returned %d conversations", len(convs))
}

// ============================================================================
// TEST: GET /conversations/{id} - Get Conversation Details
// ============================================================================

func TestGetConversation_Success(t *testing.T) {
	// Create users and conversation
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "getconv_user1"}, "")
	var user1 map[string]string
	parseJSON(t, resp1, &user1)
	user1Token := user1["identifier"]

	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "getconv_user2"}, "")
	var user2 map[string]string
	parseJSON(t, resp2, &user2)
	user2ID := user2["identifier"]

	resp3 := doRequest(t, "POST", "/conversations", map[string]string{"userId": user2ID}, user1Token)
	var conv map[string]interface{}
	parseJSON(t, resp3, &conv)
	convID := conv["id"].(string)

	// Get conversation details
	resp := doRequest(t, "GET", "/conversations/"+convID, nil, user1Token)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var convDetails map[string]interface{}
	parseJSON(t, resp, &convDetails)

	if convDetails["id"] != convID {
		t.Fatalf("Expected conversation ID %s, got %s", convID, convDetails["id"])
	}
	if convDetails["type"] != "direct" {
		t.Fatalf("Expected type 'direct', got %s", convDetails["type"])
	}
	participants := convDetails["participants"].([]interface{})
	if len(participants) != 2 {
		t.Fatalf("Expected 2 participants, got %d", len(participants))
	}
	t.Log("✅ GET /conversations/{id} returns full conversation details")
}

func TestGetConversation_NotParticipant(t *testing.T) {
	// Create three users
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "nopart_user1"}, "")
	var user1 map[string]string
	parseJSON(t, resp1, &user1)
	user1Token := user1["identifier"]

	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "nopart_user2"}, "")
	var user2 map[string]string
	parseJSON(t, resp2, &user2)
	user2ID := user2["identifier"]

	resp3 := doRequest(t, "POST", "/session", map[string]string{"name": "nopart_user3"}, "")
	var user3 map[string]string
	parseJSON(t, resp3, &user3)
	user3Token := user3["identifier"]

	// User1 creates conversation with User2
	resp4 := doRequest(t, "POST", "/conversations", map[string]string{"userId": user2ID}, user1Token)
	var conv map[string]interface{}
	parseJSON(t, resp4, &conv)
	convID := conv["id"].(string)

	// User3 tries to access (not a participant)
	resp := doRequest(t, "GET", "/conversations/"+convID, nil, user3Token)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected 404 for non-participant, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	t.Log("✅ Non-participant cannot access conversation")
}

// ============================================================================
// INTEGRATION TEST: Full Messaging Scenario
// ============================================================================

func TestFullMessagingScenario(t *testing.T) {
	t.Log("=== Integration Test: Full Messaging Scenario ===")

	// 1. Create two users
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "maria"}, "")
	var maria map[string]string
	parseJSON(t, resp1, &maria)
	mariaToken := maria["identifier"]
	t.Logf("👤 Created Maria: %s", mariaToken)

	resp2 := doRequest(t, "POST", "/session", map[string]string{"name": "john"}, "")
	var john map[string]string
	parseJSON(t, resp2, &john)
	johnToken := john["identifier"]
	johnID := john["identifier"]
	t.Logf("👤 Created John: %s", johnToken)

	// 2. Maria searches for John
	resp3 := doRequest(t, "GET", "/users?q=john", nil, mariaToken)
	var searchResult map[string][]map[string]interface{}
	parseJSON(t, resp3, &searchResult)
	if len(searchResult["users"]) == 0 {
		t.Fatal("Maria didn't find John")
	}
	t.Log("🔍 Maria found John")

	// 3. Maria starts conversation with John
	resp4 := doRequest(t, "POST", "/conversations", map[string]string{"userId": johnID}, mariaToken)
	var conv map[string]interface{}
	parseJSON(t, resp4, &conv)
	convID := conv["id"].(string)
	t.Logf("💬 Maria started conversation with John: %s", convID)

	// 4. Maria sends a message
	msgBody := map[string]interface{}{
		"contentType": "text",
		"text":        "Hey John! How are you?",
	}
	resp5 := doRequest(t, "POST", "/conversations/"+convID+"/messages", msgBody, mariaToken)
	var msg1 map[string]interface{}
	parseJSON(t, resp5, &msg1)
	msg1ID := msg1["id"].(string)
	t.Logf("📤 Maria sent: '%s'", msg1["text"])

	// 5. John fetches his conversation list (marks as received)
	resp6 := doRequest(t, "GET", "/conversations", nil, johnToken)
	var johnConvs []map[string]interface{}
	parseJSON(t, resp6, &johnConvs)
	t.Log("📬 John fetched his conversation list")

	// 6. John opens the conversation (marks as read)
	resp7 := doRequest(t, "GET", "/conversations/"+convID, nil, johnToken)
	var johnConvDetails map[string]interface{}
	parseJSON(t, resp7, &johnConvDetails)
	t.Log("📖 John opened the conversation")

	// 7. John replies
	replyBody := map[string]interface{}{
		"contentType":      "text",
		"text":             "Hi Maria! I'm good, thanks!",
		"replyToMessageId": msg1ID,
	}
	resp8 := doRequest(t, "POST", "/conversations/"+convID+"/messages", replyBody, johnToken)
	var msg2 map[string]interface{}
	parseJSON(t, resp8, &msg2)
	t.Logf("📤 John replied: '%s'", msg2["text"])

	// 8. Maria reacts to John's message
	reactionBody := map[string]string{"emoji": "😊"}
	resp9 := doRequest(t, "POST", "/conversations/"+convID+"/messages/"+msg2["id"].(string)+"/comments", reactionBody, mariaToken)
	if resp9.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp9.Body)
		t.Fatalf("Failed to add reaction: %s", string(body))
	}
	resp9.Body.Close()
	t.Log("👍 Maria reacted to John's message with 😊")

	// 9. Maria sends a photo
	photoBody := map[string]interface{}{
		"contentType": "photo",
		"photoUrl":    "https://example.com/vacation.jpg",
	}
	resp10 := doRequest(t, "POST", "/conversations/"+convID+"/messages", photoBody, mariaToken)
	var photoMsg map[string]interface{}
	parseJSON(t, resp10, &photoMsg)
	t.Logf("📷 Maria sent a photo: %s", photoMsg["photoUrl"])

	// 10. Verify final conversation state
	resp11 := doRequest(t, "GET", "/conversations/"+convID, nil, mariaToken)
	var finalConv map[string]interface{}
	parseJSON(t, resp11, &finalConv)
	messages := finalConv["messages"].([]interface{})
	t.Logf("📋 Conversation now has %d messages", len(messages))

	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(messages))
	}

	t.Log("=== ✅ Full Messaging Scenario Passed! ===")
}

// ============================================================================
// TEST: Message Yourself Feature (like WhatsApp)
// ============================================================================

func TestMessageYourself_FullScenario(t *testing.T) {
	t.Log("=== Integration Test: Message Yourself Feature ===")

	// 1. Create user
	resp1 := doRequest(t, "POST", "/session", map[string]string{"name": "solo_user"}, "")
	var user map[string]string
	parseJSON(t, resp1, &user)
	userToken := user["identifier"]
	t.Logf("👤 Created user: %s", userToken)

	// 2. Create "Message Yourself" conversation
	resp2 := doRequest(t, "POST", "/conversations", map[string]string{"userId": userToken}, userToken)
	if resp2.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("Failed to create self-conversation: %d - %s", resp2.StatusCode, string(body))
	}
	var conv map[string]interface{}
	parseJSON(t, resp2, &conv)
	convID := conv["id"].(string)

	if conv["title"] != "Message Yourself" {
		t.Fatalf("Expected title 'Message Yourself', got %s", conv["title"])
	}
	t.Logf("💬 Created 'Message Yourself' conversation: %s", convID)

	// 3. Send a note to yourself
	msgBody := map[string]interface{}{
		"contentType": "text",
		"text":        "Remember to buy groceries! 🛒",
	}
	resp3 := doRequest(t, "POST", "/conversations/"+convID+"/messages", msgBody, userToken)
	if resp3.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp3.Body)
		t.Fatalf("Failed to send message: %d - %s", resp3.StatusCode, string(body))
	}
	var msg1 map[string]interface{}
	parseJSON(t, resp3, &msg1)
	t.Logf("📝 Sent note: '%s'", msg1["text"])

	// 4. Send a photo to yourself
	photoBody := map[string]interface{}{
		"contentType": "photo",
		"photoUrl":    "https://example.com/shopping-list.jpg",
	}
	resp4 := doRequest(t, "POST", "/conversations/"+convID+"/messages", photoBody, userToken)
	var msg2 map[string]interface{}
	parseJSON(t, resp4, &msg2)
	t.Logf("📷 Sent photo: %s", msg2["photoUrl"])

	// 5. Reply to your own message
	replyBody := map[string]interface{}{
		"contentType":      "text",
		"text":             "Don't forget milk!",
		"replyToMessageId": msg1["id"],
	}
	resp5 := doRequest(t, "POST", "/conversations/"+convID+"/messages", replyBody, userToken)
	var msg3 map[string]interface{}
	parseJSON(t, resp5, &msg3)
	t.Logf("↩️ Replied to self: '%s'", msg3["text"])

	// 6. React to your own message
	reactionBody := map[string]string{"emoji": "✅"}
	resp6 := doRequest(t, "POST", "/conversations/"+convID+"/messages/"+msg1["id"].(string)+"/comments", reactionBody, userToken)
	if resp6.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp6.Body)
		t.Fatalf("Failed to add reaction: %d - %s", resp6.StatusCode, string(body))
	}
	resp6.Body.Close()
	t.Log("✅ Reacted to own message")

	// 7. Verify conversation appears in list
	resp7 := doRequest(t, "GET", "/conversations", nil, userToken)
	var convList []map[string]interface{}
	parseJSON(t, resp7, &convList)

	found := false
	for _, c := range convList {
		if c["id"] == convID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Self-conversation not found in conversation list")
	}
	t.Log("📋 Self-conversation appears in conversation list")

	// 8. Verify conversation details
	resp8 := doRequest(t, "GET", "/conversations/"+convID, nil, userToken)
	var convDetails map[string]interface{}
	parseJSON(t, resp8, &convDetails)

	messages := convDetails["messages"].([]interface{})
	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(messages))
	}

	participants := convDetails["participants"].([]interface{})
	if len(participants) != 1 {
		t.Fatalf("Expected 1 participant (self), got %d", len(participants))
	}
	t.Logf("📋 Conversation has %d messages and %d participant", len(messages), len(participants))

	// 9. Try to create duplicate self-conversation (should return existing)
	resp9 := doRequest(t, "POST", "/conversations", map[string]string{"userId": userToken}, userToken)
	if resp9.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 for duplicate self-conversation, got %d", resp9.StatusCode)
	}
	var existingConv map[string]interface{}
	parseJSON(t, resp9, &existingConv)
	if existingConv["id"] != convID {
		t.Fatalf("Expected same conversation ID, got different")
	}
	t.Log("🔄 Duplicate request returns existing self-conversation")

	// 10. Delete a message
	resp10 := doRequest(t, "DELETE", "/conversations/"+convID+"/messages/"+msg2["id"].(string), nil, userToken)
	if resp10.StatusCode != http.StatusNoContent {
		t.Fatalf("Failed to delete message: %d", resp10.StatusCode)
	}
	resp10.Body.Close()
	t.Log("🗑️ Deleted photo message")

	// 11. Final verification
	resp11 := doRequest(t, "GET", "/conversations/"+convID, nil, userToken)
	var finalConv map[string]interface{}
	parseJSON(t, resp11, &finalConv)
	finalMessages := finalConv["messages"].([]interface{})
	if len(finalMessages) != 2 {
		t.Fatalf("Expected 2 messages after deletion, got %d", len(finalMessages))
	}

	t.Log("=== ✅ Message Yourself Feature Test Passed! ===")
}
