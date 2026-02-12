<template>
  <div v-if="show" class="panel-overlay" @click.self="close">
    <div class="panel-container">
      <!-- Header -->
      <div class="panel-header">
        <h2>Group Info</h2>
        <button class="btn-close-panel" @click="close">✕</button>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="panel-loading">
        <div class="spinner-border" />
      </div>

      <!-- Error -->
      <div v-else-if="error" class="panel-error">{{ error }}</div>

      <!-- Content -->
      <div v-else-if="group" class="panel-body">
        <!-- Group Avatar -->
        <div class="group-avatar-section">
          <div class="group-avatar-large">
            <img
              v-if="group.photoUrl"
              :src="getPhotoUrl(group.photoUrl)"
              :alt="group.name"
              class="avatar-img"
            >
            <div v-else class="avatar-placeholder">
              {{ getInitials(group.name) }}
            </div>
          </div>
        </div>

        <!-- Group Name -->
        <div class="group-name-section">
          <label class="field-label">Group Name</label>
          <div v-if="!editingName" class="name-display">
            <span class="name-value">{{ group.name }}</span>
            <button class="btn-edit" title="Edit group name" @click="startEditName">
              ✏️
            </button>
          </div>
          <div v-else class="name-edit">
            <input
              v-model="newGroupName"
              type="text"
              class="edit-input"
              placeholder="Enter new group name"
              @keyup.enter="saveGroupName"
            >
            <div class="edit-actions">
              <button class="btn-cancel" @click="cancelEditName">Cancel</button>
              <button class="btn-save" @click="saveGroupName">Save</button>
            </div>
          </div>
        </div>

        <!-- Members -->
        <div class="members-section">
          <div class="members-header">
            <span class="field-label">Members ({{ group.members.length }})</span>
            <button class="btn-add-member" @click="showAddMember = true">+ Add</button>
          </div>

          <div class="members-list">
            <div
              v-for="member in group.members"
              :key="member.id"
              class="member-item"
            >
              <div class="member-avatar">
                <img
                  v-if="member.photoUrl"
                  :src="getPhotoUrl(member.photoUrl)"
                  :alt="member.name"
                  class="avatar-img-sm"
                >
                <div v-else class="avatar-placeholder-sm">
                  {{ getInitials(member.name) }}
                </div>
              </div>
              <div class="member-info">
                <span class="member-name">{{ member.displayName || member.name }}</span>
                <span class="member-username">@{{ member.name }}</span>
              </div>
              <span v-if="member.id === group.createdBy" class="creator-badge">Creator</span>
            </div>
          </div>
        </div>

        <!-- Leave Group -->
        <button class="btn-leave" @click="confirmLeaveGroup">
          🚪 Leave Group
        </button>
      </div>
    </div>

    <!-- Add Member Sub-Modal -->
    <div v-if="showAddMember" class="panel-overlay sub-modal" @click.self="showAddMember = false">
      <div class="panel-container panel-small">
        <div class="panel-header">
          <h2>Add Member</h2>
          <button class="btn-close-panel" @click="showAddMember = false">✕</button>
        </div>
        <div class="panel-body">
          <input
            v-model="memberSearchQuery"
            type="text"
            class="edit-input"
            placeholder="Search users..."
            @input="searchUsers"
          >

          <div v-if="searchingUsers" class="panel-loading small">
            <div class="spinner-border" />
          </div>

          <div v-else-if="searchResults.length > 0" class="members-list">
            <div
              v-for="user in searchResults"
              :key="user.id"
              class="member-item clickable"
              :class="{ disabled: isAlreadyMember(user.id) }"
              @click="!isAlreadyMember(user.id) && addMember(user.id)"
            >
              <div class="member-avatar">
                <img
                  v-if="user.photoUrl"
                  :src="getPhotoUrl(user.photoUrl)"
                  :alt="user.name"
                  class="avatar-img-sm"
                >
                <div v-else class="avatar-placeholder-sm">
                  {{ getInitials(user.name) }}
                </div>
              </div>
              <div class="member-info">
                <span class="member-name">{{ user.displayName || user.name }}</span>
                <span class="member-username">@{{ user.name }}</span>
              </div>
              <span v-if="isAlreadyMember(user.id)" class="already-badge">Added</span>
            </div>
          </div>

          <div v-else-if="memberSearchQuery" class="empty-text">
            No users found
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { groupAPI, userAPI } from "@/services/api.js";
import { API_URL } from "@/services/api.js";

export default {
	name: "GroupInfoPanel",
	props: {
		show: {
			type: Boolean,
			required: true,
		},
		groupId: {
			type: String,
			required: true,
		},
		currentUserId: {
			type: String,
			required: false,
		},
	},
	emits: ["group-updated", "left-group", "close"],
	
	data() {
		return {
			group: null,
			loading: true,
			error: null,
			editingName: false,
			newGroupName: "",
			showAddMember: false,
			memberSearchQuery: "",
			searchResults: [],
			searchingUsers: false,
		};
	},
	computed: {
		isCreator() {
			return this.currentUserId && this.group && this.group.createdBy === this.currentUserId;
		},
	},
	watch: {
		show(val) {
			if (val) {
				this.loadGroupInfo();
			}
		},
	},
	mounted() {
			if (this.show) {
					this.loadGroupInfo();
			}
	},
	methods: {
		async loadGroupInfo() {
			this.loading = true;
			this.error = null;
			try {
				const response = await groupAPI.getById(this.groupId);
				this.group = response.data;
			} catch (e) {
				this.error = e.response?.data?.message || "Failed to load group info";
			} finally {
				this.loading = false;
			}
		},

		getInitials(name) {
			if (!name) return "?";
			const words = name.trim().split(/\s+/);
			if (words.length === 1) return words[0].substring(0, 2).toUpperCase();
			return (words[0][0] + words[1][0]).toUpperCase();
		},

		startEditName() {
			this.editingName = true;
			this.newGroupName = this.group.name;
		},

		cancelEditName() {
			this.editingName = false;
			this.newGroupName = "";
		},

		async saveGroupName() {
			if (!this.newGroupName.trim()) {
				alert("Group name cannot be empty");
				return;
			}

			try {
				const response = await groupAPI.setName(this.groupId, this.newGroupName.trim());
				this.group.name = response.data.name;
				this.editingName = false;
				this.newGroupName = "";
				this.$emit("group-updated", response.data);
			} catch (e) {
				alert(e.response?.data?.message || "Failed to update group name");
			}
		},

		async searchUsers() {
			if (!this.memberSearchQuery.trim()) {
				this.searchResults = [];
				return;
			}

			this.searchingUsers = true;
			try {
				const response = await userAPI.searchUsers(this.memberSearchQuery.trim());
				this.searchResults = response.data.users || [];
			} catch (e) {
				console.error("Failed to search users:", e);
				this.searchResults = [];
			} finally {
				this.searchingUsers = false;
			}
		},

		isAlreadyMember(userId) {
			return this.group.members.some((m) => m.id === userId);
		},

		async addMember(userId) {
			if (this.isAlreadyMember(userId)) return;

			try {
				const response = await groupAPI.addMember(this.groupId, userId);
				this.group = response.data;
				this.showAddMember = false;
				this.memberSearchQuery = "";
				this.searchResults = [];
				this.$emit("group-updated", response.data);
			} catch (e) {
				alert(e.response?.data?.message || "Failed to add member");
			}
		},

		async confirmLeaveGroup() {
			if (!confirm("Are you sure you want to leave this group?")) return;

			try {
				await groupAPI.leave(this.groupId);
				this.$emit("left-group");
				this.close();
			} catch (e) {
				alert(e.response?.data?.message || "Failed to leave group");
			}
		},

		close() {
			this.$emit("close");
		},

		getPhotoUrl(photoUrl) {
			if (!photoUrl) return null;
			if (photoUrl.startsWith('http')) return photoUrl;
			return `${API_URL}${photoUrl}`;
		},
	},
};
</script>

<style scoped>
/* Overlay */
.panel-overlay {
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

.panel-overlay.sub-modal {
	z-index: 1050;
}

/* Container */
.panel-container {
	background: #252435;
	border-radius: 12px;
	width: 100%;
	max-width: 400px;
	max-height: 85vh;
	display: flex;
	flex-direction: column;
	border: 1px solid #3d3a52;
	overflow: hidden;
}

.panel-small {
	max-width: 360px;
}

/* Header */
.panel-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 16px 20px;
	border-bottom: 1px solid #3d3a52;
}

.panel-header h2 {
	margin: 0;
	font-size: 1.15rem;
	font-weight: 600;
	color: #e2e8f0;
}

.btn-close-panel {
	background: none;
	border: none;
	color: #94a3b8;
	font-size: 1.1rem;
	cursor: pointer;
	padding: 4px 8px;
	border-radius: 6px;
	line-height: 1;
}

.btn-close-panel:hover {
	background: #3d3a52;
	color: #e2e8f0;
}

/* Body */
.panel-body {
	padding: 20px;
	overflow-y: auto;
	-webkit-overflow-scrolling: touch;
}

/* Loading */
.panel-loading {
	display: flex;
	justify-content: center;
	padding: 40px 20px;
}

.panel-loading.small {
	padding: 20px;
}

.spinner-border {
	color: #8b5cf6;
	width: 32px;
	height: 32px;
}

/* Error */
.panel-error {
	margin: 20px;
	padding: 12px 16px;
	background: rgba(214, 48, 49, 0.15);
	border: 1px solid #d63031;
	border-radius: 8px;
	color: #ff7675;
	font-size: 0.9rem;
}

/* Group Avatar */
.group-avatar-section {
	display: flex;
	justify-content: center;
	margin-bottom: 20px;
}

.group-avatar-large {
	width: 96px;
	height: 96px;
	border-radius: 50%;
	overflow: hidden;
	border: 3px solid #3d3a52;
}

.avatar-img {
	width: 100%;
	height: 100%;
	object-fit: cover;
	display: block;
}

.avatar-placeholder {
	width: 100%;
	height: 100%;
	background: #8b5cf6;
	color: #1a1d29;
	display: flex;
	align-items: center;
	justify-content: center;
	font-weight: 700;
	font-size: 2rem;
}

/* Group Name */
.group-name-section {
	margin-bottom: 24px;
}

.field-label {
	display: block;
	font-size: 0.78rem;
	color: #64748b;
	text-transform: uppercase;
	letter-spacing: 0.05em;
	margin-bottom: 6px;
	font-weight: 500;
}

.name-display {
	display: flex;
	align-items: center;
	gap: 10px;
}

.name-value {
	font-size: 1.15rem;
	font-weight: 600;
	color: #e2e8f0;
}

.btn-edit {
	background: none;
	border: none;
	cursor: pointer;
	font-size: 0.9rem;
	padding: 4px 8px;
	border-radius: 6px;
	opacity: 0.6;
	transition: opacity 0.15s;
}

.btn-edit:hover {
	opacity: 1;
	background: #3d3a52;
}

.name-edit {
	display: flex;
	flex-direction: column;
	gap: 10px;
}

.edit-input {
	background: #1a1d29;
	border: 1px solid #3d3a52;
	color: #e2e8f0;
	border-radius: 8px;
	padding: 10px 14px;
	font-size: 0.95rem;
	outline: none;
	width: 100%;
}

.edit-input:focus {
	border-color: #8b5cf6;
}

.edit-input::placeholder {
	color: #4a4860;
}

.edit-actions {
	display: flex;
	gap: 8px;
	justify-content: flex-end;
}

.btn-save {
	background: #8b5cf6;
	color: #fff;
	border: none;
	padding: 7px 18px;
	border-radius: 6px;
	font-size: 0.85rem;
	font-weight: 500;
	cursor: pointer;
}

.btn-save:hover {
	background: #7c3aed;
}

.btn-cancel {
	background: #3d3a52;
	color: #cbd5e1;
	border: none;
	padding: 7px 18px;
	border-radius: 6px;
	font-size: 0.85rem;
	cursor: pointer;
}

.btn-cancel:hover {
	background: #4d4763;
	color: #e2e8f0;
}

/* Members */
.members-section {
	margin-bottom: 24px;
}

.members-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 12px;
}

.btn-add-member {
	background: #8b5cf6;
	color: #fff;
	border: none;
	padding: 5px 14px;
	border-radius: 6px;
	font-size: 0.8rem;
	font-weight: 500;
	cursor: pointer;
}

.btn-add-member:hover {
	background: #7c3aed;
}

.members-list {
	display: flex;
	flex-direction: column;
	gap: 2px;
	margin-top: 8px;
}

.member-item {
	display: flex;
	align-items: center;
	padding: 10px 12px;
	border-radius: 8px;
	gap: 12px;
	transition: background 0.15s;
}

.member-item:hover {
	background: #1a1d29;
}

.member-item.clickable {
	cursor: pointer;
}

.member-item.disabled {
	opacity: 0.5;
	cursor: default;
}

.member-avatar {
	flex-shrink: 0;
}

.avatar-img-sm {
	width: 40px;
	height: 40px;
	border-radius: 50%;
	object-fit: cover;
	display: block;
}

.avatar-placeholder-sm {
	width: 40px;
	height: 40px;
	border-radius: 50%;
	background: #8b5cf6;
	color: #1a1d29;
	display: flex;
	align-items: center;
	justify-content: center;
	font-weight: 600;
	font-size: 0.85rem;
}

.member-info {
	flex: 1;
	min-width: 0;
	display: flex;
	flex-direction: column;
}

.member-name {
	font-size: 0.95rem;
	font-weight: 500;
	color: #e2e8f0;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.member-username {
	font-size: 0.8rem;
	color: #64748b;
}

.creator-badge {
	font-size: 0.7rem;
	background: #8b5cf6;
	color: #fff;
	padding: 3px 10px;
	border-radius: 12px;
	flex-shrink: 0;
	font-weight: 500;
}

.already-badge {
	font-size: 0.7rem;
	background: #3d3a52;
	color: #94a3b8;
	padding: 3px 10px;
	border-radius: 12px;
	flex-shrink: 0;
}

/* Leave Button */
.btn-leave {
	width: 100%;
	background: rgba(214, 48, 49, 0.15);
	color: #ff7675;
	border: 1px solid rgba(214, 48, 49, 0.3);
	padding: 12px;
	border-radius: 8px;
	font-size: 0.95rem;
	font-weight: 500;
	cursor: pointer;
	transition: background 0.15s;
}

.btn-leave:hover {
	background: rgba(214, 48, 49, 0.25);
}

/* Empty text */
.empty-text {
	text-align: center;
	color: #64748b;
	padding: 20px;
	font-size: 0.9rem;
}

@media (max-width: 400px) {
	.panel-container {
		max-width: 100%;
		max-height: 90vh;
	}

	.group-avatar-large {
		width: 80px;
		height: 80px;
	}

	.avatar-placeholder {
		font-size: 1.6rem;
	}
}
</style>
