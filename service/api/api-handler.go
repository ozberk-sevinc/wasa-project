package api

import (
	"net/http"
)

// Handler returns an instance of httprouter.Router that handle APIs registered here
func (rt *_router) Handler() http.Handler {
	// ========================================
	// SESSION (no auth required)
	// ========================================
	rt.router.POST("/session", rt.wrap(rt.doLogin))

	// ========================================
	// CURRENT USER /me (auth required)
	// ========================================
	rt.router.GET("/me", rt.authWrap(rt.getMe))
	rt.router.PUT("/me/username", rt.authWrap(rt.setMyUsername))

	// ========================================
	// USERS (auth required)
	// ========================================
	rt.router.GET("/users", rt.authWrap(rt.searchUsers))

	// ========================================
	// SPECIAL ROUTES
	// ========================================
	rt.router.GET("/liveness", rt.liveness)

	return rt.router
}
