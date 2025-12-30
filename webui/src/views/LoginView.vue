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
	mounted() {
		// If already logged in, redirect
		if (localStorage.getItem("wasatext_token")) {
			this.$router.push("/");
		}
	},
};
</script>

<template>
	<div class="login-container">
		<div class="login-card">
			<div class="login-header">
				<h1>💬 WASAText</h1>
				<p class="text-muted">Enter your username to start chatting</p>
			</div>

			<form @submit.prevent="handleLogin" class="login-form">
				<div class="mb-3">
					<label for="username" class="form-label">Username</label>
					<input
						type="text"
						class="form-control form-control-lg"
						id="username"
						v-model="username"
						placeholder="Enter username (3-16 chars)"
						:disabled="loading"
						autofocus
					/>
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
						<span class="spinner-border spinner-border-sm me-2" role="status"></span>
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
	background: #1e272e;
	padding: 16px;
}

.login-card {
	background: #2d3436;
	border-radius: 8px;
	padding: 32px 24px;
	width: 100%;
	max-width: 360px;
	box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
	border: 1px solid #3d4852;
}

.login-header {
	text-align: center;
	margin-bottom: 24px;
}

.login-header h1 {
	font-size: 1.8rem;
	margin-bottom: 8px;
	color: #dfe6e9;
	font-weight: 600;
}

.login-header .text-muted {
	color: #636e72 !important;
	font-size: 0.9rem;
}

.login-form .form-label {
	color: #b2bec3;
	font-size: 0.85rem;
}

.login-form .form-control {
	border-radius: 6px;
	padding: 12px 14px;
	background: #1e272e;
	border: 1px solid #3d4852;
	color: #dfe6e9;
	font-size: 1rem;
}

.login-form .form-control:focus {
	border-color: #00b894;
	box-shadow: 0 0 0 2px rgba(0, 184, 148, 0.2);
	background: #1e272e;
	color: #dfe6e9;
}

.login-form .form-control::placeholder {
	color: #636e72;
}

.login-form .btn-primary {
	border-radius: 6px;
	padding: 12px;
	font-weight: 500;
	background: #00b894;
	border: none;
}

.login-form .btn-primary:hover {
	background: #00a085;
}

.login-form .btn-primary:disabled {
	background: #636e72;
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
