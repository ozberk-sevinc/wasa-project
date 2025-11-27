package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/ozberk-sevinc/wasa-project/service/api/reqcontext"
	"github.com/ozberk-sevinc/wasa-project/service/database"
	"github.com/sirupsen/logrus"
)

type contextKey string

const userContextKey contextKey = "user"

// GetUserFromContext retrieves the authenticated user from request context
func GetUserFromContext(ctx context.Context) *database.User {
	user, ok := ctx.Value(userContextKey).(*database.User)
	if !ok {
		return nil
	}
	return user
}

// authWrap wraps a handler with Bearer token authentication
// It validates the token, looks up the user, and injects into context
func (rt *_router) authWrap(fn httpRouterHandler) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// Create request context with UUID and logger
		reqUUID, err := uuid.NewV4()
		if err != nil {
			rt.baseLogger.WithError(err).Error("can't generate a request UUID")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		ctx := reqcontext.RequestContext{
			ReqUUID: reqUUID,
			Logger: rt.baseLogger.WithFields(logrus.Fields{
				"reqid":     reqUUID.String(),
				"remote-ip": r.RemoteAddr,
			}),
		}

		// Get Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			sendUnauthorized(w, "Authorization header is required")
			return
		}

		// Extract Bearer token
		if !strings.HasPrefix(authHeader, "Bearer ") {
			sendUnauthorized(w, "Authorization header must use Bearer scheme")
			return
		}
		userID := strings.TrimPrefix(authHeader, "Bearer ")

		// Look up user
		user, err := rt.db.GetUserByID(userID)
		if err != nil {
			ctx.Logger.WithError(err).Error("database error looking up user")
			sendInternalError(w, "Database error")
			return
		}
		if user == nil {
			sendUnauthorized(w, "Invalid identifier")
			return
		}

		// Add user to request context
		reqCtx := context.WithValue(r.Context(), userContextKey, user)

		// Call the handler
		fn(w, r.WithContext(reqCtx), ps, ctx)
	}
}
