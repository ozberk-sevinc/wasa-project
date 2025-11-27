package api

import (
	"encoding/json"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/ozberk-sevinc/wasa-project/service/api/reqcontext"
)

// ============================================================================
// RESPONSE TYPES (matching api.yaml schemas)
// ============================================================================

// UserResponse matches the User schema
type UserResponse struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	PhotoURL *string `json:"photoUrl,omitempty"`
}

// LoginRequest is the request body for POST /session
type LoginRequest struct {
	Name string `json:"name"`
}

// LoginResponse is the response for POST /session
type LoginResponse struct {
	Identifier string `json:"identifier"`
}

// SetUsernameRequest is the request body for PUT /me/username
type SetUsernameRequest struct {
	Name string `json:"name"`
}

// SearchUsersResponse is the response for GET /users
type SearchUsersResponse struct {
	Users []UserResponse `json:"users"`
}

// ============================================================================
// SESSION / LOGIN ENDPOINTS
// ============================================================================

// doLogin handles POST /session
// - If username exists, return its ID
// - If username doesn't exist, create new user and return its ID
func (rt *_router) doLogin(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendBadRequest(w, "Invalid JSON")
		return
	}

	// Validate username length (3-16 characters)
	if len(req.Name) < 3 || len(req.Name) > 16 {
		sendBadRequest(w, "Username must be between 3 and 16 characters")
		return
	}

	// Check if user exists
	user, err := rt.db.GetUserByName(req.Name)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error")
		sendInternalError(w, "Database error")
		return
	}

	var userID string

	if user != nil {
		// User exists, return existing ID
		userID = user.ID
	} else {
		// User doesn't exist, create new one
		newID, err := uuid.NewV4()
		if err != nil {
			sendInternalError(w, "Error generating ID")
			return
		}
		userID = newID.String()

		if err := rt.db.CreateUser(userID, req.Name); err != nil {
			ctx.Logger.WithError(err).Error("error creating user")
			sendInternalError(w, "Error creating user")
			return
		}
	}

	sendJSON(w, http.StatusCreated, LoginResponse{
		Identifier: userID,
	})
}

// ============================================================================
// CURRENT USER (/me) ENDPOINTS
// ============================================================================

// getMe handles GET /me - returns the current user's profile
func (rt *_router) getMe(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		sendUnauthorized(w, "User not found in context")
		return
	}

	sendJSON(w, http.StatusOK, UserResponse{
		ID:       user.ID,
		Name:     user.Name,
		PhotoURL: user.PhotoURL,
	})
}

// setMyUsername handles PUT /me/username - change current user's username
func (rt *_router) setMyUsername(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		sendUnauthorized(w, "User not found in context")
		return
	}

	var req SetUsernameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendBadRequest(w, "Invalid JSON")
		return
	}

	// Validate username length
	if len(req.Name) < 3 || len(req.Name) > 16 {
		sendBadRequest(w, "Username must be between 3 and 16 characters")
		return
	}

	// Check if new username is already taken
	existing, err := rt.db.GetUserByName(req.Name)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error")
		sendInternalError(w, "Database error")
		return
	}
	if existing != nil && existing.ID != user.ID {
		sendConflict(w, "Username is already taken")
		return
	}

	// Update username
	if err := rt.db.UpdateUsername(user.ID, req.Name); err != nil {
		ctx.Logger.WithError(err).Error("error updating username")
		sendInternalError(w, "Error updating username")
		return
	}

	// Return updated user
	sendJSON(w, http.StatusOK, UserResponse{
		ID:       user.ID,
		Name:     req.Name,
		PhotoURL: user.PhotoURL,
	})
}

// ============================================================================
// USERS ENDPOINTS
// ============================================================================

// searchUsers handles GET /users - list or search users
func (rt *_router) searchUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	query := r.URL.Query().Get("q")

	var users []UserResponse

	if query == "" {
		// Return all users
		dbUsers, err := rt.db.GetAllUsers()
		if err != nil {
			ctx.Logger.WithError(err).Error("database error")
			sendInternalError(w, "Database error")
			return
		}
		for _, u := range dbUsers {
			users = append(users, UserResponse{
				ID:       u.ID,
				Name:     u.Name,
				PhotoURL: u.PhotoURL,
			})
		}
	} else {
		// Search users by query
		dbUsers, err := rt.db.SearchUsers(query)
		if err != nil {
			ctx.Logger.WithError(err).Error("database error")
			sendInternalError(w, "Database error")
			return
		}
		for _, u := range dbUsers {
			users = append(users, UserResponse{
				ID:       u.ID,
				Name:     u.Name,
				PhotoURL: u.PhotoURL,
			})
		}
	}

	// Ensure we return empty array instead of null
	if users == nil {
		users = []UserResponse{}
	}

	sendJSON(w, http.StatusOK, SearchUsersResponse{
		Users: users,
	})
}
