<script>
import { userAPI } from "@/services/api.js";
import { API_URL } from "@/services/api.js";

export default {
	name: "ProfileView",
	data() {
		return {
			user: null,
			loading: true,
			error: null,
			editingUsername: false,
			newUsername: "",
			saving: false,
		};
	},
	mounted() {
		this.loadProfile();
	},
	methods: {
		async loadProfile() {
			this.loading = true;
			this.error = null;
			try {
				const response = await userAPI.getMe();
				this.user = response.data;
				this.newUsername = this.user.name;
			} catch (e) {
				this.error = e.response?.data?.message || "Failed to load profile";
			} finally {
				this.loading = false;
			}
		},

		startEditUsername() {
			this.editingUsername = true;
			this.newUsername = this.user.name;
		},

		cancelEditUsername() {
			this.editingUsername = false;
			this.newUsername = this.user.name;
		},

		async saveUsername() {
			if (!this.newUsername.trim()) return;
			if (this.newUsername.length < 3 || this.newUsername.length > 16) {
				alert("Username must be 3-16 characters");
				return;
			}

			this.saving = true;
			try {
				const response = await userAPI.setUsername(this.newUsername.trim());
				this.user = response.data;
				localStorage.setItem("wasatext_user", JSON.stringify(this.user));
				this.editingUsername = false;
			} catch (e) {
				alert(e.response?.data?.message || "Failed to update username");
			} finally {
				this.saving = false;
			}
		},

		triggerPhotoInput() {
			this.$refs.photoInput.click();
		},

		async handlePhotoSelect(event) {
			const file = event.target.files[0];
			if (!file) return;

			try {
				const response = await userAPI.setPhoto(file);
				this.user = response.data;
				localStorage.setItem("wasatext_user", JSON.stringify(this.user));
			} catch (e) {
				alert(e.response?.data?.message || "Failed to update photo");
			} finally {
				event.target.value = "";
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

		getPhotoUrl(photoUrl) {
			if (!photoUrl) return null;
			if (photoUrl.startsWith('http')) return photoUrl;
			return `${API_URL}${photoUrl}`;
		},
	},
};
</script>

<template>
  <div class="profile-container">
    <!-- Header -->
    <header class="profile-header">
      <button class="btn-back" @click="$router.push('/')">← Back</button>
      <h1>Profile</h1>
      <div />
    </header>

    <!-- Loading -->
    <div v-if="loading" class="text-center p-5">
      <div class="spinner-border text-primary" />
    </div>

    <!-- Error -->
    <div v-else-if="error" class="alert alert-danger m-3">
      {{ error }}
    </div>

    <!-- Profile Content -->
    <div v-else class="profile-content">
      <!-- Hidden file input for profile photo -->
      <input
        ref="photoInput"
        type="file"
        accept="image/*"
        style="display: none"
        @change="handlePhotoSelect"
      >

      <!-- Avatar -->
      <div class="profile-avatar-section">
        <div class="profile-avatar" @click="triggerPhotoInput">
          <img v-if="user.photoUrl" :src="getPhotoUrl(user.photoUrl)" :alt="user.name">
          <div v-else class="avatar-placeholder large">
            {{ getInitials(user.name) }}
          </div>
          <div class="avatar-overlay">
            <span>📷</span>
          </div>
        </div>
        <p class="text-muted small mt-2">Tap to change photo</p>
      </div>

      <!-- Info Cards -->
      <div class="profile-cards">
        <!-- Username -->
        <div class="profile-card">
          <div class="card-label">Username</div>
          <div v-if="!editingUsername" class="card-value-row">
            <span class="card-value">{{ user.name }}</span>
            <button class="btn btn-sm btn-outline-primary" @click="startEditUsername">
              Edit
            </button>
          </div>
          <div v-else class="card-edit">
            <input
              v-model="newUsername"
              type="text"
              class="form-control"
              placeholder="New username (3-16 chars)"
              :disabled="saving"
            >
            <div class="edit-buttons">
              <button
                class="btn btn-sm btn-secondary"
                :disabled="saving"
                @click="cancelEditUsername"
              >
                Cancel
              </button>
              <button
                class="btn btn-sm btn-primary"
                :disabled="saving || !newUsername.trim()"
                @click="saveUsername"
              >
                <span v-if="saving">Saving...</span>
                <span v-else>Save</span>
              </button>
            </div>
          </div>
        </div>

        <!-- User ID -->
        <div class="profile-card">
          <div class="card-label">User ID</div>
          <div class="card-value-row">
            <span class="card-value id-value">{{ user.id }}</span>
          </div>
        </div>

        <!-- Display Name (if set) -->
        <div v-if="user.displayName" class="profile-card">
          <div class="card-label">Display Name</div>
          <div class="card-value-row">
            <span class="card-value">{{ user.displayName }}</span>
          </div>
        </div>
      </div>

      <!-- Logout -->
      <div class="logout-section">
        <button class="btn btn-danger btn-lg w-100" @click="logout">
          Logout
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.profile-container {
	min-height: 100vh;
	min-height: 100dvh;
	background: #1a1d29;
	display: flex;
	flex-direction: column;
}

.profile-header {
	background: #252435;
	color: #e2e8f0;
	padding: 12px 16px;
	display: flex;
	justify-content: space-between;
	align-items: center;
	border-bottom: 1px solid #3d3a52;
}

.profile-header h1 {
	margin: 0;
	font-size: 1.2rem;
	font-weight: 500;
}

.btn-back {
	background: none;
	border: none;
	color: #cbd5e1;
	font-size: 0.95rem;
	cursor: pointer;
	padding: 6px 10px;
	border-radius: 4px;
}

.btn-back:hover {
	background: #3d3a52;
	color: #e2e8f0;
}

.profile-content {
	padding: 20px 16px;
	display: flex;
	flex-direction: column;
	gap: 24px;
}

.profile-avatar-section {
	text-align: center;
}

.profile-avatar {
	width: 100px;
	height: 100px;
	border-radius: 50%;
	margin: 0 auto 8px;
	position: relative;
	cursor: pointer;
	overflow: hidden;
}

.profile-avatar img {
	width: 100%;
	height: 100%;
	object-fit: cover;
}

.avatar-placeholder.large {
	width: 100%;
	height: 100%;
	background: #8b5cf6;
	color: #1a1d29;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 2rem;
	font-weight: 600;
}

.avatar-overlay {
	position: absolute;
	bottom: 0;
	left: 0;
	right: 0;
	background: rgba(0, 0, 0, 0.6);
	color: #e2e8f0;
	padding: 6px;
	text-align: center;
	opacity: 0;
	transition: opacity 0.2s;
}

.profile-avatar:hover .avatar-overlay {
	opacity: 1;
}

.text-muted {
	color: #64748b !important;
}

.profile-cards {
	display: flex;
	flex-direction: column;
	gap: 12px;
}

.profile-card {
	background: #252435;
	border-radius: 8px;
	padding: 14px 16px;
	border: 1px solid #3d3a52;
}

.card-label {
	font-size: 0.8rem;
	color: #64748b;
	margin-bottom: 6px;
}

.card-value-row {
	display: flex;
	justify-content: space-between;
	align-items: center;
	flex-wrap: wrap;
	gap: 8px;
}

.card-value {
	font-size: 1rem;
	font-weight: 500;
	color: #e2e8f0;
}

.card-value.id-value {
	font-size: 0.75rem;
	font-family: monospace;
	color: #cbd5e1;
	word-break: break-all;
}

.card-edit {
	display: flex;
	flex-direction: column;
	gap: 10px;
}

.card-edit .form-control {
	background: #1a1d29;
	border: 1px solid #3d3a52;
	color: #e2e8f0;
	border-radius: 6px;
	padding: 10px 12px;
}

.card-edit .form-control:focus {
	border-color: #8b5cf6;
	box-shadow: none;
	outline: none;
}

.edit-buttons {
	display: flex;
	gap: 8px;
	justify-content: flex-end;
	flex-wrap: wrap;
}

.btn-sm {
	padding: 6px 14px;
	font-size: 0.85rem;
	border-radius: 6px;
}

.btn-outline-primary {
	color: #8b5cf6;
	border-color: #8b5cf6;
}

.btn-outline-primary:hover {
	background: #8b5cf6;
	color: #1a1d29;
}

.btn-primary {
	background: #8b5cf6;
	border: none;
	color: #1a1d29;
}

.btn-primary:hover {
	background: #7c3aed;
}

.btn-secondary {
	background: #3d3a52;
	border: none;
	color: #cbd5e1;
}

.btn-secondary:hover {
	background: #4d4763;
	color: #e2e8f0;
}

.logout-section {
	margin-top: auto;
	padding-top: 16px;
}

.btn-danger {
	background: #d63031;
	border: none;
	border-radius: 6px;
	padding: 12px;
	font-weight: 500;
}

.btn-danger:hover {
	background: #c0392b;
}

.spinner-border {
	color: #8b5cf6 !important;
}

.alert-danger {
	background: rgba(214, 48, 49, 0.15);
	border: 1px solid #d63031;
	color: #ff7675;
	border-radius: 6px;
}

@media (max-width: 400px) {
	.profile-avatar {
		width: 80px;
		height: 80px;
	}
	.avatar-placeholder.large {
		font-size: 1.6rem;
	}
}
</style>
