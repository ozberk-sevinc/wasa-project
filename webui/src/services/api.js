import axios from "axios";

export const API_URL = __API_URL__;

// Create axios instance
const api = axios.create({
	baseURL: __API_URL__,
	timeout: 30000, // Increased from 10s to 30s
	headers: {
		"Content-Type": "application/json",
	},
});

// Add auth token to requests
api.interceptors.request.use((config) => {
	const token = localStorage.getItem("wasatext_token");
	if (token) {
		config.headers.Authorization = `Bearer ${token}`;
	}
	return config;
});

// Handle auth errors
api.interceptors.response.use(
	(response) => response,
	(error) => {
		if (error.response?.status === 401) {
			localStorage.removeItem("wasatext_token");
			localStorage.removeItem("wasatext_user");
			window.location.href = "/login";
		}
		return Promise.reject(error);
	}
);

// ============================================================================
// AUTH API
// ============================================================================

export const authAPI = {
	login: (name) => api.post("/session", { name }),
};

// ============================================================================
// USER API
// ============================================================================

export const userAPI = {
	getMe: () => api.get("/me"),
	setUsername: (name) => api.put("/me/username", { name }),
	setPhoto: (file) => {
		const formData = new FormData();
		formData.append("photo", file);
		return api.put("/me/photo", formData, {
			headers: { "Content-Type": "multipart/form-data" },
		});
	},
	searchUsers: (query = "") => api.get(`/users${query ? `?q=${encodeURIComponent(query)}` : ""}`),
};

// ============================================================================
// CONVERSATION API
// ============================================================================

export const conversationAPI = {
	getAll: () => api.get("/conversations"),
	getById: (id) => api.get(`/conversations/${id}`),
	create: (userId) => api.post("/conversations", { userId }),
};

// ============================================================================
// MESSAGE API
// ============================================================================

export const messageAPI = {
	uploadPhoto: (conversationId, file) => {
		const formData = new FormData();
		formData.append("photo", file);
		return api.post(`/conversations/${conversationId}/photos`, formData, {
			headers: { "Content-Type": "multipart/form-data" },
		});
	},
	send: (conversationId, { contentType, text, photoUrl, replyToMessageId }) =>
		api.post(`/conversations/${conversationId}/messages`, {
			contentType,
			text,
			photoUrl,
			replyToMessageId,
		}),
	delete: (conversationId, messageId) =>
		api.delete(`/conversations/${conversationId}/messages/${messageId}`),
	forward: (conversationId, messageId, targetConversationId) =>
		api.post(`/conversations/${conversationId}/messages/${messageId}/forward`, {
			targetConversationId,
		}),
	addReaction: (conversationId, messageId, emoji) =>
		api.post(`/conversations/${conversationId}/messages/${messageId}/comments`, { emoji }),
	removeReaction: (conversationId, messageId, reactionId) =>
		api.delete(`/conversations/${conversationId}/messages/${messageId}/comments/${reactionId}`),
};

// ============================================================================
// GROUP API
// ============================================================================

export const groupAPI = {
	create: (name, memberIds = []) => api.post("/groups", { name, memberIds }),
	getById: (id) => api.get(`/groups/${id}`),
	addMember: (groupId, userId) => api.post(`/groups/${groupId}/members`, { userId }),
	leave: (groupId) => api.delete(`/groups/${groupId}/members/me`),
	setName: (groupId, name) => api.put(`/groups/${groupId}/name`, { name }),
	setPhoto: (groupId, file) => {
		const formData = new FormData();
		formData.append("photo", file);
		return api.put(`/groups/${groupId}/photo`, formData, {
			headers: { "Content-Type": "multipart/form-data" },
		});
	},
};

export default api;
