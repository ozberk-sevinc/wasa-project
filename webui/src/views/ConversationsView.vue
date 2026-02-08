<script>
import { conversationAPI, userAPI, groupAPI } from "@/services/api.js";

export default {
	name: "ConversationsView",
	data() {
		return {
			conversations: [],
			loading: true,
			error: null,
			currentUser: null,
			showNewChat: false,
			searchQuery: "",
			searchResults: [],
			searchLoading: false,
			// Group creation
			showCreateGroup: false,
			groupName: "",
			groupSearchQuery: "",
			groupSearchResults: [],
			selectedMembers: [],
			creatingGroup: false,
			// Group photo edit
			editingGroupPhoto: null,
		};
	},
	computed: {
		sortedConversations() {
			return [...this.conversations].sort((a, b) => {
				const dateA = a.lastMessageAt ? new Date(a.lastMessageAt) : new Date(0);
				const dateB = b.lastMessageAt ? new Date(b.lastMessageAt) : new Date(0);
				return dateB - dateA;
			});
		},
	},
	methods: {
		async loadConversations() {
			this.loading = true;
			this.error = null;
			try {
				const response = await conversationAPI.getAll();
				this.conversations = response.data || [];
			} catch (e) {
				this.error = e.response?.data?.message || "Failed to load conversations";
			} finally {
				this.loading = false;
			}
		},

		async loadCurrentUser() {
			try {
				const response = await userAPI.getMe();
				this.currentUser = response.data;
				localStorage.setItem("wasatext_user", JSON.stringify(this.currentUser));
			} catch (e) {
				console.error("Failed to load user", e);
			}
		},

		async searchUsers() {
			if (!this.searchQuery.trim()) {
				this.searchResults = [];
				return;
			}
			this.searchLoading = true;
			try {
				const response = await userAPI.searchUsers(this.searchQuery);
				// Filter out current user from results
				this.searchResults = (response.data.users || []).filter(
					(u) => u.id !== this.currentUser?.id
				);
			} catch (e) {
				console.error("Search failed", e);
			} finally {
				this.searchLoading = false;
			}
		},

		async startConversation(userId) {
			try {
				const response = await conversationAPI.create(userId);
				this.showNewChat = false;
				this.searchQuery = "";
				this.searchResults = [];
				// Navigate to the conversation
				this.$router.push(`/chat/${response.data.id}`);
			} catch (e) {
				alert(e.response?.data?.message || "Failed to start conversation");
			}
		},

		async startSelfConversation() {
			try {
				const response = await conversationAPI.create(this.currentUser.id);
				this.showNewChat = false;
				this.$router.push(`/chat/${response.data.id}`);
			} catch (e) {
				alert(e.response?.data?.message || "Failed to create conversation");
			}
		},

		openConversation(convId) {
			this.$router.push(`/chat/${convId}`);
		},

		formatTime(dateString) {
			if (!dateString) return "";
			const date = new Date(dateString);
			const now = new Date();
			const diffDays = Math.floor((now - date) / (1000 * 60 * 60 * 24));

			if (diffDays === 0) {
				return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
			} else if (diffDays === 1) {
				return "Yesterday";
			} else if (diffDays < 7) {
				return date.toLocaleDateString([], { weekday: "short" });
			} else {
				return date.toLocaleDateString([], { month: "short", day: "numeric" });
			}
		},

		logout() {
			localStorage.removeItem("wasatext_token");
			localStorage.removeItem("wasatext_user");
			this.$router.push("/login");
		},

		getInitials(name) {
			return name ? name.substring(0, 2).toUpperCase() : "??";
		},

		// Group methods
		openCreateGroup() {
			this.showNewChat = false;
			this.showCreateGroup = true;
			this.groupName = "";
			this.groupSearchQuery = "";
			this.groupSearchResults = [];
			this.selectedMembers = [];
		},

		closeCreateGroup() {
			this.showCreateGroup = false;
			this.groupName = "";
			this.groupSearchQuery = "";
			this.groupSearchResults = [];
			this.selectedMembers = [];
		},

		async searchGroupMembers() {
			if (!this.groupSearchQuery.trim()) {
				this.groupSearchResults = [];
				return;
			}
			try {
				const response = await userAPI.searchUsers(this.groupSearchQuery);
				this.groupSearchResults = (response.data.users || []).filter(
					(u) => u.id !== this.currentUser?.id && !this.selectedMembers.find((m) => m.id === u.id)
				);
			} catch (e) {
				console.error("Search failed", e);
			}
		},

		addMember(user) {
			if (!this.selectedMembers.find((m) => m.id === user.id)) {
				this.selectedMembers.push(user);
			}
			this.groupSearchQuery = "";
			this.groupSearchResults = [];
		},

		removeMember(userId) {
			this.selectedMembers = this.selectedMembers.filter((m) => m.id !== userId);
		},

		async createGroup() {
			if (!this.groupName.trim()) {
				alert("Please enter a group name");
				return;
			}
			this.creatingGroup = true;
			try {
				const memberIds = this.selectedMembers.map((m) => m.id);
				const response = await groupAPI.create(this.groupName.trim(), memberIds);
				this.closeCreateGroup();
				// The group is also a conversation, navigate to it
				this.$router.push(`/chat/${response.data.id}`);
			} catch (e) {
				alert(e.response?.data?.message || "Failed to create group");
			} finally {
				this.creatingGroup = false;
			}
		},

		// Group photo methods
		openGroupPhotoEdit(event, conv) {
			event.stopPropagation();
			this.editingGroupPhoto = conv;
		},

		closeGroupPhotoEdit() {
			this.editingGroupPhoto = null;
		},

		triggerGroupPhotoInput() {
			this.$refs.groupPhotoInput.click();
		},

		async handleGroupPhotoSelect(event) {
			const file = event.target.files[0];
			if (!file) return;

			try {
				const response = await groupAPI.setPhoto(this.editingGroupPhoto.id, file);
				// Update local state with returned photo URL
				const conv = this.conversations.find(c => c.id === this.editingGroupPhoto.id);
				if (conv && response.data?.photoUrl) {
					conv.photoUrl = response.data.photoUrl;
				}
				this.closeGroupPhotoEdit();
			} catch (e) {
				alert(e.response?.data?.message || "Failed to update group photo");
			} finally {
				event.target.value = "";
			}
		},

		async removeGroupPhoto() {
			if (!confirm("Remove group photo?")) return;
			try {
				await groupAPI.setPhoto(this.editingGroupPhoto.id, "");
				const conv = this.conversations.find(c => c.id === this.editingGroupPhoto.id);
				if (conv) {
					conv.photoUrl = null;
				}
				this.closeGroupPhotoEdit();
			} catch (e) {
				alert(e.response?.data?.message || "Failed to remove photo");
			}
		},
	},
	mounted() {
		this.loadCurrentUser();
		this.loadConversations();
	},
};
</script>

<template>
	<div class="conversations-container">
		<!-- Header -->
		<header class="conv-header">
			<div class="header-left">
				<img src="/logo.png" alt="WASAText" class="header-logo" />
				<h1>WASAText</h1>
			</div>
			<div class="header-right">
				<button class="btn btn-light btn-sm me-2" @click="$router.push('/profile')">
					{{ currentUser?.name || "Profile" }}
				</button>
				<button class="btn btn-outline-light btn-sm" @click="logout">Logout</button>
			</div>
		</header>

		<!-- Toolbar -->
		<div class="conv-toolbar">
			<button class="btn btn-primary" @click="showNewChat = true">
				<span class="me-1">+</span> New Chat
			</button>
			<button class="btn btn-outline-secondary ms-2" @click="loadConversations">
				🔄 Refresh
			</button>
		</div>

		<!-- Loading -->
		<div v-if="loading" class="text-center p-5">
			<div class="spinner-border text-primary" role="status"></div>
			<p class="mt-2 text-muted">Loading conversations...</p>
		</div>

		<!-- Error -->
		<div v-else-if="error" class="alert alert-danger m-3">
			{{ error }}
			<button class="btn btn-sm btn-outline-danger ms-2" @click="loadConversations">Retry</button>
		</div>

		<!-- Empty State -->
		<div v-else-if="conversations.length === 0" class="empty-state">
			<div class="empty-icon">💬</div>
			<h3>No conversations yet</h3>
			<p class="text-muted">Start a new chat to begin messaging</p>
			<button class="btn btn-primary" @click="showNewChat = true">Start New Chat</button>
		</div>

		<!-- Conversation List -->
		<div v-else class="conversation-list">
			<div
				v-for="conv in sortedConversations"
				:key="conv.id"
				class="conversation-item"
				@click="openConversation(conv.id)"
			>
				<div class="conv-avatar" :class="{ 'group-avatar': conv.type === 'group' }">
					<div v-if="conv.photoUrl" class="avatar-img">
						<img :src="conv.photoUrl" :alt="conv.title" />
					</div>
					<div v-else class="avatar-placeholder">
						{{ getInitials(conv.title) }}
					</div>
					<!-- Edit icon for groups -->
					<button
						v-if="conv.type === 'group'"
						class="avatar-edit-btn"
						@click="openGroupPhotoEdit($event, conv)"
						title="Change group photo"
					>
						📷
					</button>
				</div>
				<div class="conv-details">
					<div class="conv-top">
						<span class="conv-title">{{ conv.title }}</span>
						<span class="conv-time">{{ formatTime(conv.lastMessageAt) }}</span>
					</div>
					<div class="conv-bottom">
						<span class="conv-preview">
							<span v-if="conv.lastMessageIsPhoto">📷 Photo</span>
							<span v-else>{{ conv.lastMessageSnippet || "No messages yet" }}</span>
						</span>
						<span v-if="conv.type === 'group'" class="conv-badge">Group</span>
					</div>
				</div>
			</div>
		</div>

		<!-- New Chat Modal -->
		<div v-if="showNewChat" class="modal-overlay" @click.self="showNewChat = false">
			<div class="modal-content">
				<div class="modal-header">
					<h5>New Chat</h5>
					<button class="btn-close" @click="showNewChat = false"></button>
				</div>
				<div class="modal-body">
					<!-- Message Yourself -->
					<div class="self-chat-option" @click="startSelfConversation">
						<div class="avatar-placeholder">📝</div>
						<div class="self-chat-text">
							<strong>Message Yourself</strong>
							<span class="text-muted">Save notes, links, and reminders</span>
						</div>
					</div>

					<!-- Create Group -->
					<div class="self-chat-option" @click="openCreateGroup" style="margin-top: 10px;">
						<div class="avatar-placeholder">👥</div>
						<div class="self-chat-text">
							<strong>Create Group</strong>
							<span class="text-muted">Chat with multiple people</span>
						</div>
					</div>

					<hr />

					<!-- Search Users -->
					<div class="search-section">
						<input
							type="text"
							class="form-control"
							placeholder="Search users by name..."
							v-model="searchQuery"
							@input="searchUsers"
						/>
					</div>

					<div v-if="searchLoading" class="text-center p-3">
						<div class="spinner-border spinner-border-sm"></div>
					</div>

					<div v-else-if="searchResults.length > 0" class="search-results">
						<div
							v-for="user in searchResults"
							:key="user.id"
							class="search-result-item"
							@click="startConversation(user.id)"
						>
							<div class="avatar-placeholder small">
								{{ getInitials(user.name) }}
							</div>
							<div class="user-info">
								<strong>{{ user.name }}</strong>
								<span v-if="user.displayName" class="text-muted">{{ user.displayName }}</span>
							</div>
						</div>
					</div>

					<div v-else-if="searchQuery && !searchLoading" class="text-center p-3 text-muted">
						No users found
					</div>
				</div>
			</div>
		</div>

		<!-- Create Group Modal -->
		<div v-if="showCreateGroup" class="modal-overlay" @click.self="closeCreateGroup">
			<div class="modal-content">
				<div class="modal-header">
					<h5>Create Group</h5>
					<button class="btn-close" @click="closeCreateGroup"></button>
				</div>
				<div class="modal-body">
					<!-- Group Name -->
					<div class="form-group">
						<label>Group Name</label>
						<input
							type="text"
							class="form-control"
							placeholder="Enter group name..."
							v-model="groupName"
							:disabled="creatingGroup"
						/>
					</div>

					<!-- Selected Members -->
					<div v-if="selectedMembers.length > 0" class="selected-members">
						<label>Members ({{ selectedMembers.length }})</label>
						<div class="member-chips">
							<span v-for="member in selectedMembers" :key="member.id" class="member-chip">
								{{ member.name }}
								<button @click="removeMember(member.id)" class="remove-chip">×</button>
							</span>
						</div>
					</div>

					<!-- Search Members -->
					<div class="search-section">
						<label>Add Members</label>
						<input
							type="text"
							class="form-control"
							placeholder="Search users to add..."
							v-model="groupSearchQuery"
							@input="searchGroupMembers"
							:disabled="creatingGroup"
						/>
					</div>

					<div v-if="groupSearchResults.length > 0" class="search-results">
						<div
							v-for="user in groupSearchResults"
							:key="user.id"
							class="search-result-item"
							@click="addMember(user)"
						>
							<div class="avatar-placeholder small">
								{{ getInitials(user.name) }}
							</div>
							<div class="user-info">
								<strong>{{ user.name }}</strong>
							</div>
						</div>
					</div>

					<!-- Create Button -->
					<button
						class="btn btn-primary w-100 mt-3"
						@click="createGroup"
						:disabled="!groupName.trim() || creatingGroup"
					>
						<span v-if="creatingGroup">Creating...</span>
						<span v-else>Create Group</span>
					</button>
				</div>
			</div>
		</div>

		<!-- Hidden file input for group photo -->
		<input
			type="file"
			ref="groupPhotoInput"
			accept="image/*"
			style="display: none"
			@change="handleGroupPhotoSelect"
		/>

		<!-- Group Photo Edit Modal -->
		<div v-if="editingGroupPhoto" class="modal-overlay" @click.self="closeGroupPhotoEdit">
			<div class="modal-content modal-small">
				<div class="modal-header">
					<h5>Change Group Photo</h5>
					<button class="btn-close" @click="closeGroupPhotoEdit"></button>
				</div>
				<div class="modal-body text-center">
					<!-- Current Photo Preview -->
					<div class="photo-preview">
						<div v-if="editingGroupPhoto.photoUrl" class="avatar-img large">
							<img :src="editingGroupPhoto.photoUrl" :alt="editingGroupPhoto.title" />
						</div>
						<div v-else class="avatar-placeholder large">
							{{ getInitials(editingGroupPhoto.title) }}
						</div>
					</div>
					<p class="group-name-label">{{ editingGroupPhoto.title }}</p>

					<!-- Action Buttons -->
					<div class="photo-actions">
						<button class="btn btn-primary w-100 mb-2" @click="triggerGroupPhotoInput">
							📷 Choose Photo
						</button>
						<button
							v-if="editingGroupPhoto.photoUrl"
							class="btn btn-outline-danger w-100"
							@click="removeGroupPhoto"
						>
							🗑️ Remove Photo
						</button>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<style scoped>
.conversations-container {
	height: 100vh;
	height: 100dvh;
	display: flex;
	flex-direction: column;
	background: #1a1d29;
}

.conv-header {
	background: #252435;
	color: #e2e8f0;
	padding: 12px 16px;
	display: flex;
	flex-wrap: wrap;
	justify-content: space-between;
	align-items: center;
	gap: 8px;
	border-bottom: 1px solid #3d3a52;
}

.header-left {
	display: flex;
	align-items: center;
	gap: 12px;
}

.header-logo {
	width: 40px;
	height: 40px;
	border-radius: 8px;
	object-fit: contain;
}

.conv-header h1 {
	margin: 0;
	font-size: 1.3rem;
	font-weight: 600;
}

.header-right {
	display: flex;
	gap: 8px;
	flex-wrap: wrap;
}

.header-right .btn {
	padding: 6px 12px;
	font-size: 0.85rem;
}

.btn-light {
	background: #3d3a52;
	color: #e2e8f0;
	border: none;
}

.btn-light:hover {
	background: #4d4763;
	color: #fff;
}

.btn-outline-light {
	color: #cbd5e1;
	border-color: #3d3a52;
}

.btn-outline-light:hover {
	background: #3d3a52;
	color: #e2e8f0;
}

.conv-toolbar {
	padding: 12px 16px;
	background: #252435;
	border-bottom: 1px solid #3d3a52;
	display: flex;
	flex-wrap: wrap;
	gap: 8px;
}

.conv-toolbar .btn-primary {
	background: #8b5cf6;
	border: none;
}

.conv-toolbar .btn-primary:hover {
	background: #7c3aed;
}

.conv-toolbar .btn-outline-secondary {
	color: #cbd5e1;
	border-color: #3d3a52;
}

.conv-toolbar .btn-outline-secondary:hover {
	background: #3d3a52;
	color: #e2e8f0;
}

.conversation-list {
	flex: 1;
	overflow-y: auto;
	-webkit-overflow-scrolling: touch;
}

.conversation-item {
	display: flex;
	padding: 14px 16px;
	background: #252435;
	border-bottom: 1px solid #3d3a52;
	cursor: pointer;
	transition: background 0.15s;
}

.conversation-item:hover {
	background: #3d3a52;
}

.conversation-item:active {
	background: #4d4763;
}

.conv-avatar {
	margin-right: 12px;
	flex-shrink: 0;
	position: relative;
}

.conv-avatar.group-avatar:hover .avatar-edit-btn {
	opacity: 1;
}

.avatar-edit-btn {
	position: absolute;
	bottom: -2px;
	right: -2px;
	width: 24px;
	height: 24px;
	border-radius: 50%;
	background: #8b5cf6;
	border: 2px solid #252435;
	font-size: 0.7rem;
	cursor: pointer;
	display: flex;
	align-items: center;
	justify-content: center;
	opacity: 0;
	transition: opacity 0.2s;
	z-index: 2;
}

.avatar-edit-btn:hover {
	background: #7c3aed;
}

.avatar-img img {
	width: 48px;
	height: 48px;
	border-radius: 50%;
	object-fit: cover;
}

.avatar-placeholder {
	width: 48px;
	height: 48px;
	border-radius: 50%;
	background: #8b5cf6;
	color: #1a1d29;
	display: flex;
	align-items: center;
	justify-content: center;
	font-weight: 600;
	font-size: 1rem;
}

.avatar-placeholder.small {
	width: 40px;
	height: 40px;
	font-size: 0.85rem;
}

.avatar-placeholder.large,
.avatar-img.large img {
	width: 100px;
	height: 100px;
	font-size: 2rem;
}

.avatar-img.large img {
	border-radius: 50%;
}

.conv-details {
	flex: 1;
	min-width: 0;
	display: flex;
	flex-direction: column;
	justify-content: center;
}

.conv-top {
	display: flex;
	justify-content: space-between;
	align-items: baseline;
	margin-bottom: 4px;
	gap: 8px;
}

.conv-title {
	font-weight: 500;
	color: #e2e8f0;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.conv-time {
	font-size: 0.75rem;
	color: #64748b;
	flex-shrink: 0;
}

.conv-bottom {
	display: flex;
	justify-content: space-between;
	align-items: center;
	gap: 8px;
}

.conv-preview {
	color: #cbd5e1;
	font-size: 0.85rem;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
	flex: 1;
}

.conv-badge {
	font-size: 0.7rem;
	background: #3d3a52;
	padding: 2px 8px;
	border-radius: 10px;
	color: #cbd5e1;
	flex-shrink: 0;
}

.empty-state {
	flex: 1;
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	text-align: center;
	padding: 32px 20px;
}

.empty-state h3 {
	color: #e2e8f0;
	margin-bottom: 8px;
}

.empty-state .text-muted {
	color: #64748b !important;
}

.empty-icon {
	font-size: 3rem;
	margin-bottom: 16px;
}

.spinner-border {
	color: #8b5cf6 !important;
}

.text-muted {
	color: #64748b !important;
}

.alert-danger {
	background: rgba(214, 48, 49, 0.15);
	border: 1px solid #d63031;
	color: #ff7675;
}

/* Modal */
.modal-overlay {
	position: fixed;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	background: rgba(0, 0, 0, 0.7);
	display: flex;
	align-items: center;
	justify-content: center;
	padding: 16px;
	z-index: 1000;
}

.modal-content {
	background: #252435;
	border-radius: 8px;
	width: 100%;
	max-width: 380px;
	max-height: 85vh;
	overflow: hidden;
	display: flex;
	flex-direction: column;
	border: 1px solid #3d3a52;
}

.modal-content.modal-small {
	max-width: 300px;
}

.photo-preview {
	margin: 16px auto;
	display: flex;
	justify-content: center;
}

.group-name-label {
	color: #e2e8f0;
	font-weight: 600;
	margin-bottom: 20px;
}

.photo-actions {
	margin-top: 10px;
}

.modal-header {
	padding: 14px 16px;
	border-bottom: 1px solid #3d3a52;
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.modal-header h5 {
	margin: 0;
	color: #e2e8f0;
	font-size: 1.1rem;
}

.btn-close {
	filter: invert(1);
	opacity: 0.6;
}

.modal-body {
	padding: 16px;
	overflow-y: auto;
	-webkit-overflow-scrolling: touch;
}

.self-chat-option {
	display: flex;
	align-items: center;
	padding: 12px;
	background: #1a1d29;
	border-radius: 8px;
	cursor: pointer;
	transition: background 0.15s;
}

.self-chat-option:hover {
	background: #3d3a52;
}

.self-chat-text {
	margin-left: 12px;
	display: flex;
	flex-direction: column;
	gap: 2px;
}

.self-chat-text strong {
	color: #e2e8f0;
}

.self-chat-text .text-muted {
	font-size: 0.85rem;
}

.search-section {
	margin-top: 12px;
}

.search-section .form-control {
	background: #1e272e;
	border: 1px solid #3d4852;
	color: #dfe6e9;
	border-radius: 6px;
}

.search-section .form-control:focus {
	border-color: #00b894;
	box-shadow: none;
}

.search-section .form-control::placeholder {
	color: #636e72;
}

.search-results {
	margin-top: 12px;
	display: flex;
	flex-direction: column;
	gap: 4px;
}

.search-result-item {
	display: flex;
	align-items: center;
	padding: 10px;
	border-radius: 6px;
	cursor: pointer;
	transition: background 0.15s;
}

.search-result-item:hover {
	background: #3d4852;
}

.user-info {
	margin-left: 10px;
	display: flex;
	flex-direction: column;
}

.user-info strong {
	color: #dfe6e9;
}

.user-info .text-muted {
	font-size: 0.8rem;
}

/* Group creation styles */
.form-group {
	margin-bottom: 16px;
}

.form-group label,
.search-section label,
.selected-members label {
	display: block;
	color: #b2bec3;
	font-size: 0.85rem;
	margin-bottom: 6px;
}

.selected-members {
	margin-bottom: 16px;
}

.member-chips {
	display: flex;
	flex-wrap: wrap;
	gap: 6px;
}

.member-chip {
	background: #00b894;
	color: #1e272e;
	padding: 4px 10px;
	border-radius: 16px;
	font-size: 0.85rem;
	display: flex;
	align-items: center;
	gap: 6px;
}

.remove-chip {
	background: none;
	border: none;
	color: #1e272e;
	font-size: 1rem;
	cursor: pointer;
	padding: 0;
	line-height: 1;
	opacity: 0.7;
}

.remove-chip:hover {
	opacity: 1;
}

.mt-3 {
	margin-top: 12px;
}

.w-100 {
	width: 100%;
}

.btn-primary {
	background: #00b894;
	border: none;
	color: #1e272e;
	padding: 10px 16px;
	border-radius: 6px;
	font-weight: 500;
	cursor: pointer;
}

.btn-primary:hover:not(:disabled) {
	background: #00a085;
}

.btn-primary:disabled {
	background: #3d4852;
	color: #636e72;
	cursor: not-allowed;
}

hr {
	border: none;
	border-top: 1px solid #3d4852;
	margin: 16px 0;
}

@media (max-width: 400px) {
	.conv-header h1 {
		font-size: 1.1rem;
	}
	.avatar-placeholder {
		width: 44px;
		height: 44px;
	}
}
</style>
