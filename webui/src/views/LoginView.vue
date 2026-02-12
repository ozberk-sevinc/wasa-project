<script>
import { authAPI } from "@/services/api.js";

export default {
	name: "LoginView",
	data() {
		return {
			username: "",
			loading: false,
			error: null,
		};
	},
	mounted() {
		// If already logged in, redirect
		if (localStorage.getItem("wasatext_token")) {
			this.$router.push("/");
		}
	},
	methods: {
		async handleLogin() {
			if (!this.username.trim()) {
				this.error = "Please enter a username";
				return;
			}

			if (this.username.length < 3 || this.username.length > 16) {
				this.error = "Username must be 3-16 characters";
				return;
			}

			this.loading = true;
			this.error = null;

			try {
				const response = await authAPI.login(this.username.trim());
				const token = response.data.identifier;

				// Store token and username
				localStorage.setItem("wasatext_token", token);
				localStorage.setItem("wasatext_user", JSON.stringify({
					id: token,
					name: this.username.trim(),
				}));

				// Redirect to conversations
				this.$router.push("/");
			} catch (e) {
				if (e.response?.data?.message) {
					this.error = e.response.data.message;
				} else {
					this.error = "Login failed. Please try again.";
				}
			} finally {
				this.loading = false;
			}
		},
	},
};
</script>

<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-header">
        <img src="/logo.png" alt="WASAText Logo" class="login-logo">
        <h1>WASAText</h1>
        <p class="text-muted">Enter your username to start chatting</p>
      </div>

      <form class="login-form" @submit.prevent="handleLogin">
        <div class="mb-3">
          <label for="username" class="form-label">Username</label>
          <input
            id="username"
            v-model="username"
            type="text"
            class="form-control form-control-lg"
            placeholder="Enter username (3-16 chars)"
            :disabled="loading"
            autofocus
          >
        </div>

        <div v-if="error" class="alert alert-danger" role="alert">
          {{ error }}
        </div>

        <button
          type="submit"
          class="btn btn-primary btn-lg w-100"
          :disabled="loading || !username.trim()"
        >
          <span v-if="loading">
            <span class="spinner-border spinner-border-sm me-2" role="status" />
            Logging in...
          </span>
          <span v-else>Continue</span>
        </button>

        <p class="text-muted text-center mt-3 small">
          If this username is new, a new account will be created.
        </p>
      </form>
    </div>
  </div>
</template>

<style scoped>
.login-container {
	min-height: 100vh;
	min-height: 100dvh;
	display: flex;
	align-items: center;
	justify-content: center;
	background: #1a1d29;
	padding: 16px;
}

.login-card {
	background: #252435;
	border-radius: 8px;
	padding: 32px 24px;
	width: 100%;
	max-width: 360px;
	box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
	border: 1px solid #3d3a52;
}

.login-header {
	text-align: center;
	margin-bottom: 24px;
}

.login-logo {
	width: 120px;
	height: 120px;
	margin: 0 auto 16px;
	display: block;
	border-radius: 20px;
	object-fit: contain;
}

.login-header h1 {
	font-size: 1.8rem;
	margin-bottom: 8px;
	color: #e2e8f0;
	font-weight: 600;
}

.login-header .text-muted {
	color: #64748b !important;
	font-size: 0.9rem;
}

.login-form .form-label {
	color: #cbd5e1;
	font-size: 0.85rem;
}

.login-form .form-control {
	border-radius: 6px;
	padding: 12px 14px;
	background: #1a1d29;
	border: 1px solid #3d3a52;
	color: #e2e8f0;
	font-size: 1rem;
}

.login-form .form-control:focus {
	border-color: #8b5cf6;
	box-shadow: 0 0 0 2px rgba(139, 92, 246, 0.2);
	background: #1a1d29;
	color: #e2e8f0;
}

.login-form .form-control::placeholder {
	color: #64748b;
}

.login-form .btn-primary {
	border-radius: 6px;
	padding: 12px;
	font-weight: 500;
	background: #8b5cf6;
	border: none;
}

.login-form .btn-primary:hover {
	background: #7c3aed;
}

.login-form .btn-primary:disabled {
	background: #64748b;
}

.alert-danger {
	background: rgba(214, 48, 49, 0.15);
	border: 1px solid #d63031;
	color: #ff7675;
	border-radius: 6px;
}

@media (max-width: 400px) {
	.login-card {
		padding: 24px 18px;
	}
	.login-header h1 {
		font-size: 1.5rem;
	}
}
</style>
