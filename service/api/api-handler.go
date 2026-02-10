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
	rt.router.PUT("/me/photo", rt.authWrap(rt.setMyPhoto))

	// ========================================
	// USERS (auth required)
	// ========================================
	rt.router.GET("/users", rt.authWrap(rt.searchUsers))

	// ========================================
	// CONVERSATIONS (auth required)
	// ========================================
	rt.router.POST("/conversations", rt.authWrap(rt.createConversation))
	rt.router.GET("/conversations", rt.authWrap(rt.getMyConversations))
	rt.router.GET("/conversations/:conversationId", rt.authWrap(rt.getConversation))
	rt.router.POST("/conversations/:conversationId/messages", rt.authWrap(rt.sendMessage))
	rt.router.POST("/conversations/:conversationId/photos", rt.authWrap(rt.uploadMessagePhoto))
	rt.router.DELETE("/conversations/:conversationId/messages/:messageId", rt.authWrap(rt.deleteMessage))
	rt.router.POST("/conversations/:conversationId/messages/:messageId/forward", rt.authWrap(rt.forwardMessage))
	rt.router.POST("/conversations/:conversationId/messages/:messageId/comments", rt.authWrap(rt.commentMessage))
	rt.router.DELETE("/conversations/:conversationId/messages/:messageId/comments/:commentId", rt.authWrap(rt.uncommentMessage))

	// ========================================
	// GROUPS (auth required)
	// ========================================
	rt.router.POST("/groups", rt.authWrap(rt.createGroup))
	rt.router.GET("/groups/:groupId", rt.authWrap(rt.getGroup))
	rt.router.POST("/groups/:groupId/members", rt.authWrap(rt.addToGroup))
	rt.router.DELETE("/groups/:groupId/members/me", rt.authWrap(rt.leaveGroup))
	rt.router.PUT("/groups/:groupId/name", rt.authWrap(rt.setGroupName))
	rt.router.PUT("/groups/:groupId/photo", rt.authWrap(rt.setGroupPhoto))

	// ========================================
	// SPECIAL ROUTES
	// ========================================
	rt.router.GET("/liveness", rt.liveness)
	rt.router.GET("/ws", rt.wrap(rt.handleWebSocket))

	// Serve uploaded files
	rt.router.ServeFiles("/uploads/*filepath", http.Dir("./uploads"))

	return rt.router
}
