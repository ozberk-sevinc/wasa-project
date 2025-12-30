<script>
import { conversationAPI, messageAPI } from "@/services/api.js";

export default {
	name: "ChatView",
	data() {
		return {
			conversation: null,
			messages: [],
			loading: true,
			error: null,
			currentUser: null,
			newMessage: "",
			sending: false,
			replyingTo: null,
			showEmojiPicker: null,
			showAttachMenu: false,
			commonEmojis: ["👍", "❤️", "😂", "😮", "😢", "🙏", "🔥", "✅"],
		};
	},
	computed: {
		conversationId() {
			return this.$route.params.id;
		},
		sortedMessages() {
			// Sort oldest first so they appear top to bottom
			return [...this.messages].sort(
				(a, b) => new Date(a.createdAt) - new Date(b.createdAt)
			);
		},
	},
	methods: {
		async loadConversation() {
			this.loading = true;
			this.error = null;
			try {
				const response = await conversationAPI.getById(this.conversationId);
				this.conversation = response.data;
				this.messages = response.data.messages || [];
				this.$nextTick(() => this.scrollToBottom());
			} catch (e) {
				this.error = e.response?.data?.message || "Failed to load conversation";
			} finally {
				this.loading = false;
			}
		},

		async sendMessage() {
			if (!this.newMessage.trim() || this.sending) return;

			this.sending = true;
			try {
				const payload = {
					contentType: "text",
					text: this.newMessage.trim(),
				};
				if (this.replyingTo) {
					payload.replyToMessageId = this.replyingTo.id;
				}

				const response = await messageAPI.send(this.conversationId, payload);
				this.messages.push(response.data);
				this.newMessage = "";
				this.replyingTo = null;
				this.$nextTick(() => this.scrollToBottom());
			} catch (e) {
				alert(e.response?.data?.message || "Failed to send message");
			} finally {
				this.sending = false;
			}
		},

		// Attachment methods
		toggleAttachMenu() {
			this.showAttachMenu = !this.showAttachMenu;
		},

		triggerFileInput(type) {
			this.showAttachMenu = false;
			const input = this.$refs[`${type}Input`];
			if (input) input.click();
		},

		async handleFileSelect(event, type) {
			const file = event.target.files[0];
			if (!file) return;

			// For demo: use a fake URL (in real app, upload to server first)
			// In production, you'd upload to a file server and get back a URL
			const fakeUrl = URL.createObjectURL(file);
			
			this.sending = true;
			try {
				let payload = {};
				
				if (type === "photo") {
					payload = {
						contentType: "photo",
						photoUrl: fakeUrl,
					};
				} else if (type === "audio") {
					payload = {
						contentType: "audio",
						fileUrl: fakeUrl,
						fileName: file.name,
					};
				} else {
					payload = {
						contentType: "document",
						fileUrl: fakeUrl,
						fileName: file.name,
					};
				}

				if (this.replyingTo) {
					payload.replyToMessageId = this.replyingTo.id;
				}

				const response = await messageAPI.send(this.conversationId, payload);
				this.messages.push(response.data);
				this.replyingTo = null;
				this.$nextTick(() => this.scrollToBottom());
			} catch (e) {
				alert(e.response?.data?.message || "Failed to send file");
			} finally {
				this.sending = false;
				event.target.value = "";
			}
		},

		async sendPhotoUrl() {
			const url = prompt("Enter image URL:");
			if (!url) return;

			this.sending = true;
			try {
				const payload = {
					contentType: "photo",
					photoUrl: url,
				};
				if (this.replyingTo) {
					payload.replyToMessageId = this.replyingTo.id;
				}
				const response = await messageAPI.send(this.conversationId, payload);
				this.messages.push(response.data);
				this.replyingTo = null;
				this.$nextTick(() => this.scrollToBottom());
			} catch (e) {
				alert(e.response?.data?.message || "Failed to send photo");
			} finally {
				this.sending = false;
			}
		},

		getFileIcon(contentType) {
			switch (contentType) {
				case "audio": return "🎵";
				case "document": return "📄";
				case "file": return "📎";
				default: return "📁";
			}
		},

		async deleteMessage(messageId) {
			if (!confirm("Delete this message?")) return;
			try {
				await messageAPI.delete(this.conversationId, messageId);
				this.messages = this.messages.filter((m) => m.id !== messageId);
			} catch (e) {
				alert(e.response?.data?.message || "Failed to delete message");
			}
		},

		async addReaction(messageId, emoji) {
			try {
				const response = await messageAPI.addReaction(this.conversationId, messageId, emoji);
				// Update message reactions
				const msg = this.messages.find((m) => m.id === messageId);
				if (msg) {
					msg.reactions = msg.reactions || [];
					msg.reactions.push(response.data);
				}
				this.showEmojiPicker = null;
			} catch (e) {
				alert(e.response?.data?.message || "Failed to add reaction");
			}
		},

		async removeReaction(messageId, reactionId) {
			try {
				await messageAPI.removeReaction(this.conversationId, messageId, reactionId);
				const msg = this.messages.find((m) => m.id === messageId);
				if (msg) {
					msg.reactions = msg.reactions.filter((r) => r.id !== reactionId);
				}
			} catch (e) {
				alert(e.response?.data?.message || "Failed to remove reaction");
			}
		},

		setReply(message) {
			this.replyingTo = message;
			this.$refs.messageInput?.focus();
		},

		cancelReply() {
			this.replyingTo = null;
		},

		scrollToBottom() {
			const container = this.$refs.messagesContainer;
			if (container) {
				container.scrollTop = container.scrollHeight;
			}
		},

		formatTime(dateString) {
			const date = new Date(dateString);
			return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
		},

		formatDate(dateString) {
			const date = new Date(dateString);
			const today = new Date();
			const yesterday = new Date(today);
			yesterday.setDate(yesterday.getDate() - 1);

			if (date.toDateString() === today.toDateString()) {
				return "Today";
			} else if (date.toDateString() === yesterday.toDateString()) {
				return "Yesterday";
			} else {
				return date.toLocaleDateString([], {
					weekday: "long",
					month: "short",
					day: "numeric",
				});
			}
		},

		isNewDay(index) {
			if (index === 0) return true;
			const current = new Date(this.sortedMessages[index].createdAt).toDateString();
			const prev = new Date(this.sortedMessages[index - 1].createdAt).toDateString();
			return current !== prev;
		},

		isOwnMessage(message) {
			return message.sender?.id === this.currentUser?.id;
		},

		getStatusIcon(status) {
			switch (status) {
				case "sent":
					return "✓";
				case "received":
					return "✓";
				case "read":
					return "✓✓";
				default:
					return "";
			}
		},

		getStatusClass(status) {
			return status === "read" ? "status-read" : "status-default";
		},

		getRepliedMessage(messageId) {
			return this.messages.find((m) => m.id === messageId);
		},

		getInitials(name) {
			return name ? name.substring(0, 2).toUpperCase() : "??";
		},

		goBack() {
			this.$router.push("/");
		},
	},
	mounted() {
		const userData = localStorage.getItem("wasatext_user");
		if (userData) {
			this.currentUser = JSON.parse(userData);
		}
		this.loadConversation();
	},
	watch: {
		conversationId() {
			this.loadConversation();
		},
	},
};
</script>

<template>
	<div class="chat-container">
		<!-- Header -->
		<header class="chat-header">
			<button class="btn-back" @click="goBack">
				← Back
			</button>
			<div class="chat-info" v-if="conversation">
				<div class="avatar-placeholder small">
					{{ getInitials(conversation.title) }}
				</div>
				<div class="chat-title-section">
					<h2>{{ conversation.title }}</h2>
					<span class="chat-subtitle" v-if="conversation.type === 'group'">
						{{ conversation.participants?.length }} members
					</span>
				</div>
			</div>
			<button class="btn btn-light btn-sm" @click="loadConversation">🔄</button>
		</header>

		<!-- Loading -->
		<div v-if="loading" class="chat-loading">
			<div class="spinner-border text-primary"></div>
			<p>Loading messages...</p>
		</div>

		<!-- Error -->
		<div v-else-if="error" class="chat-error">
			<p>{{ error }}</p>
			<button class="btn btn-primary" @click="loadConversation">Retry</button>
		</div>

		<!-- Messages -->
		<div v-else class="messages-container" ref="messagesContainer">
			<div v-if="sortedMessages.length === 0" class="no-messages">
				<p>No messages yet. Say hello! 👋</p>
			</div>

			<template v-for="(message, index) in sortedMessages" :key="message.id">
				<!-- Date separator -->
				<div v-if="isNewDay(index)" class="date-separator">
					<span>{{ formatDate(message.createdAt) }}</span>
				</div>

				<!-- Message bubble -->
				<div
					class="message-wrapper"
					:class="{ 'own-message': isOwnMessage(message) }"
				>
					<div class="message-bubble">
						<!-- Reply reference -->
						<div
							v-if="message.repliedToMessageId"
							class="reply-reference"
							@click="scrollToMessage(message.repliedToMessageId)"
						>
							<div class="reply-bar"></div>
							<div class="reply-content">
								<strong>{{ getRepliedMessage(message.repliedToMessageId)?.sender?.name || "Unknown" }}</strong>
								<p>{{ getRepliedMessage(message.repliedToMessageId)?.text || "Message" }}</p>
							</div>
						</div>

						<!-- Sender name (for group chats) -->
						<div
							v-if="!isOwnMessage(message) && conversation?.type === 'group'"
							class="sender-name"
						>
							{{ message.sender?.name }}
						</div>

						<!-- Content -->
						<div class="message-content">
							<!-- Photo -->
							<img
								v-if="message.contentType === 'photo'"
								:src="message.photoUrl"
								class="message-photo"
								alt="Photo"
							/>
							<!-- Audio -->
							<div v-else-if="message.contentType === 'audio'" class="file-message">
								<span class="file-icon">🎵</span>
								<div class="file-info">
									<span class="file-name">{{ message.fileName || 'Audio' }}</span>
									<audio :src="message.fileUrl" controls class="audio-player"></audio>
								</div>
							</div>
							<!-- Document / File -->
							<a
								v-else-if="message.contentType === 'document' || message.contentType === 'file'"
								:href="message.fileUrl"
								target="_blank"
								class="file-message file-link"
							>
								<span class="file-icon">{{ getFileIcon(message.contentType) }}</span>
								<div class="file-info">
									<span class="file-name">{{ message.fileName || 'Document' }}</span>
									<span class="file-hint">Tap to open</span>
								</div>
							</a>
							<!-- Text -->
							<p v-else class="message-text">{{ message.text }}</p>
						</div>

						<!-- Footer -->
						<div class="message-footer">
							<span class="message-time">{{ formatTime(message.createdAt) }}</span>
							<span
								v-if="isOwnMessage(message)"
								class="message-status"
								:class="getStatusClass(message.status)"
							>
								{{ getStatusIcon(message.status) }}
							</span>
						</div>

						<!-- Reactions -->
						<div v-if="message.reactions?.length > 0" class="message-reactions">
							<span
								v-for="reaction in message.reactions"
								:key="reaction.id"
								class="reaction-badge"
								@click="removeReaction(message.id, reaction.id)"
								:title="reaction.user?.name"
							>
								{{ reaction.emoji }}
							</span>
						</div>
					</div>

					<!-- Message actions -->
					<div class="message-actions">
						<button @click="setReply(message)" title="Reply">↩️</button>
						<button @click="showEmojiPicker = message.id" title="React">😊</button>
						<button
							v-if="isOwnMessage(message)"
							@click="deleteMessage(message.id)"
							title="Delete"
						>
							🗑️
						</button>
					</div>

					<!-- Emoji picker -->
					<div v-if="showEmojiPicker === message.id" class="emoji-picker">
						<button
							v-for="emoji in commonEmojis"
							:key="emoji"
							@click="addReaction(message.id, emoji)"
						>
							{{ emoji }}
						</button>
						<button @click="showEmojiPicker = null">✕</button>
					</div>
				</div>
			</template>
		</div>

		<!-- Reply preview -->
		<div v-if="replyingTo" class="reply-preview">
			<div class="reply-preview-content">
				<strong>Replying to {{ replyingTo.sender?.name }}</strong>
				<p>{{ replyingTo.text?.substring(0, 50) }}{{ replyingTo.text?.length > 50 ? "..." : "" }}</p>
			</div>
			<button class="btn-cancel-reply" @click="cancelReply">✕</button>
		</div>

		<!-- Hidden file inputs -->
		<input
			type="file"
			ref="photoInput"
			accept="image/*"
			style="display: none"
			@change="(e) => handleFileSelect(e, 'photo')"
		/>
		<input
			type="file"
			ref="audioInput"
			accept="audio/*"
			style="display: none"
			@change="(e) => handleFileSelect(e, 'audio')"
		/>
		<input
			type="file"
			ref="documentInput"
			accept=".pdf,.doc,.docx,.xls,.xlsx,.txt,.ppt,.pptx"
			style="display: none"
			@change="(e) => handleFileSelect(e, 'document')"
		/>

		<!-- Input area -->
		<div class="input-area">
			<!-- Attachment button -->
			<div class="attach-wrapper">
				<button class="btn-attach" @click="toggleAttachMenu" type="button">
					📎
				</button>
				<div v-if="showAttachMenu" class="attach-menu">
					<button @click="triggerFileInput('photo')">📷 Photo</button>
					<button @click="triggerFileInput('audio')">🎵 Audio</button>
					<button @click="triggerFileInput('document')">📄 Document</button>
					<button @click="sendPhotoUrl()">🔗 URL</button>
				</div>
			</div>
			<input
				ref="messageInput"
				type="text"
				class="form-control"
				placeholder="Type a message..."
				v-model="newMessage"
				@keyup.enter="sendMessage"
				:disabled="sending"
			/>
			<button
				class="btn btn-primary btn-send"
				@click="sendMessage"
				:disabled="!newMessage.trim() || sending"
			>
				<span v-if="sending" class="spinner-border spinner-border-sm"></span>
				<span v-else>Send</span>
			</button>
		</div>
	</div>
</template>

<style scoped>
.chat-container {
	height: 100vh;
	height: 100dvh;
	display: flex;
	flex-direction: column;
	background: #1e272e;
}

.chat-header {
	background: #2d3436;
	color: #dfe6e9;
	padding: 10px 12px;
	display: flex;
	align-items: center;
	gap: 8px;
	border-bottom: 1px solid #3d4852;
	flex-shrink: 0;
}

.btn-back {
	background: none;
	border: none;
	color: #b2bec3;
	font-size: 0.95rem;
	cursor: pointer;
	padding: 6px 10px;
	border-radius: 4px;
}

.btn-back:hover {
	background: #3d4852;
	color: #dfe6e9;
}

.chat-info {
	flex: 1;
	display: flex;
	align-items: center;
	gap: 10px;
	min-width: 0;
}

.avatar-placeholder.small {
	width: 38px;
	height: 38px;
	border-radius: 50%;
	background: #00b894;
	color: #1e272e;
	display: flex;
	align-items: center;
	justify-content: center;
	font-weight: 600;
	font-size: 0.9rem;
	flex-shrink: 0;
}

.chat-title-section {
	min-width: 0;
}

.chat-title-section h2 {
	margin: 0;
	font-size: 1rem;
	font-weight: 500;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.chat-subtitle {
	font-size: 0.75rem;
	color: #636e72;
}

.chat-header .btn-light {
	background: #3d4852;
	color: #b2bec3;
	border: none;
	padding: 6px 10px;
}

.chat-loading,
.chat-error {
	flex: 1;
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	color: #b2bec3;
}

.spinner-border {
	color: #00b894 !important;
}

.messages-container {
	flex: 1;
	overflow-y: auto;
	-webkit-overflow-scrolling: touch;
	padding: 12px;
	background: #1e272e;
}

.no-messages {
	text-align: center;
	padding: 32px 16px;
	color: #636e72;
}

.date-separator {
	text-align: center;
	margin: 16px 0;
}

.date-separator span {
	background: #3d4852;
	padding: 4px 12px;
	border-radius: 12px;
	font-size: 0.75rem;
	color: #b2bec3;
}

.message-wrapper {
	display: flex;
	flex-direction: column;
	align-items: flex-start;
	margin-bottom: 8px;
	position: relative;
}

.message-wrapper.own-message {
	align-items: flex-end;
}

.message-bubble {
	max-width: 80%;
	background: #2d3436;
	border-radius: 12px;
	border-top-left-radius: 4px;
	padding: 8px 12px;
	position: relative;
	border: 1px solid #3d4852;
}

.own-message .message-bubble {
	background: #00b894;
	border-color: #00a085;
	border-radius: 12px;
	border-top-right-radius: 4px;
}

.own-message .message-text {
	color: #1e272e;
}

.own-message .message-time {
	color: rgba(30, 39, 46, 0.6);
}

.reply-reference {
	display: flex;
	background: rgba(0, 0, 0, 0.15);
	border-radius: 4px;
	padding: 6px 8px;
	margin-bottom: 6px;
	cursor: pointer;
}

.reply-bar {
	width: 3px;
	background: #00b894;
	border-radius: 2px;
	margin-right: 8px;
	flex-shrink: 0;
}

.reply-content {
	font-size: 0.8rem;
	min-width: 0;
}

.reply-content strong {
	color: #00b894;
}

.own-message .reply-content strong {
	color: #1e272e;
}

.reply-content p {
	margin: 0;
	color: #b2bec3;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.own-message .reply-content p {
	color: rgba(30, 39, 46, 0.7);
}

.sender-name {
	font-size: 0.75rem;
	font-weight: 600;
	color: #00b894;
	margin-bottom: 2px;
}

.message-text {
	margin: 0;
	word-wrap: break-word;
	color: #dfe6e9;
	font-size: 0.95rem;
	line-height: 1.4;
}

.message-photo {
	max-width: 100%;
	max-height: 280px;
	border-radius: 6px;
}

.message-footer {
	display: flex;
	justify-content: flex-end;
	align-items: center;
	gap: 4px;
	margin-top: 4px;
}

.message-time {
	font-size: 0.65rem;
	color: #636e72;
}

.message-status {
	font-size: 0.75rem;
	color: #636e72;
}

.message-status.status-read {
	color: #00b894;
}

.own-message .message-status {
	color: rgba(30, 39, 46, 0.5);
}

.own-message .message-status.status-read {
	color: #1e272e;
}

.message-reactions {
	display: flex;
	flex-wrap: wrap;
	gap: 4px;
	margin-top: 4px;
}

.reaction-badge {
	background: rgba(0, 184, 148, 0.2);
	padding: 2px 6px;
	border-radius: 10px;
	font-size: 0.85rem;
	cursor: pointer;
}

.message-actions {
	display: none;
	position: absolute;
	top: -4px;
	right: -70px;
	background: #2d3436;
	border-radius: 16px;
	padding: 4px 6px;
	box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
	border: 1px solid #3d4852;
}

.own-message .message-actions {
	right: auto;
	left: -70px;
}

.message-wrapper:hover .message-actions {
	display: flex;
	gap: 2px;
}

.message-actions button {
	background: none;
	border: none;
	cursor: pointer;
	font-size: 0.9rem;
	padding: 4px;
	border-radius: 4px;
}

.message-actions button:hover {
	background: #3d4852;
}

.emoji-picker {
	position: absolute;
	top: 100%;
	margin-top: 4px;
	background: #2d3436;
	border-radius: 8px;
	padding: 8px;
	box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
	border: 1px solid #3d4852;
	display: flex;
	flex-wrap: wrap;
	gap: 4px;
	z-index: 100;
}

.emoji-picker button {
	background: none;
	border: none;
	font-size: 1.2rem;
	cursor: pointer;
	padding: 6px;
	border-radius: 4px;
}

.emoji-picker button:hover {
	background: #3d4852;
}

.reply-preview {
	background: #2d3436;
	padding: 10px 14px;
	display: flex;
	justify-content: space-between;
	align-items: center;
	border-left: 3px solid #00b894;
	border-top: 1px solid #3d4852;
	flex-shrink: 0;
}

.reply-preview-content strong {
	color: #00b894;
	display: block;
	font-size: 0.8rem;
}

.reply-preview-content p {
	margin: 0;
	font-size: 0.8rem;
	color: #b2bec3;
}

.btn-cancel-reply {
	background: none;
	border: none;
	font-size: 1.1rem;
	cursor: pointer;
	color: #636e72;
	padding: 4px;
}

.btn-cancel-reply:hover {
	color: #dfe6e9;
}

.input-area {
	padding: 10px 12px;
	background: #2d3436;
	display: flex;
	gap: 8px;
	border-top: 1px solid #3d4852;
	flex-shrink: 0;
	align-items: center;
}

.attach-wrapper {
	position: relative;
}

.btn-attach {
	background: none;
	border: none;
	font-size: 1.4rem;
	cursor: pointer;
	padding: 6px 8px;
	border-radius: 8px;
	transition: background 0.2s;
}

.btn-attach:hover {
	background: #3d4852;
}

.attach-menu {
	position: absolute;
	bottom: 50px;
	left: 0;
	background: #2d3436;
	border: 1px solid #3d4852;
	border-radius: 10px;
	padding: 6px;
	display: flex;
	flex-direction: column;
	gap: 2px;
	z-index: 100;
	box-shadow: 0 4px 12px rgba(0,0,0,0.4);
	min-width: 140px;
}

.attach-menu button {
	background: none;
	border: none;
	color: #dfe6e9;
	padding: 10px 14px;
	text-align: left;
	cursor: pointer;
	border-radius: 6px;
	font-size: 0.9rem;
	white-space: nowrap;
}

.attach-menu button:hover {
	background: #3d4852;
}

.input-area .form-control {
	border-radius: 20px;
	padding: 10px 16px;
	background: #1e272e;
	border: 1px solid #3d4852;
	color: #dfe6e9;
	flex: 1;
}

.input-area .form-control:focus {
	border-color: #00b894;
	box-shadow: none;
	outline: none;
}

.input-area .form-control::placeholder {
	color: #636e72;
}

.btn-send {
	border-radius: 20px;
	padding: 10px 18px;
	background: #00b894;
	border: none;
	color: #1e272e;
	font-weight: 500;
	flex-shrink: 0;
}

.btn-send:hover:not(:disabled) {
	background: #00a085;
}

.btn-send:disabled {
	background: #3d4852;
	color: #636e72;
}

/* File message styles */
.file-message {
	display: flex;
	align-items: center;
	gap: 10px;
	padding: 8px 12px;
	background: rgba(0,0,0,0.15);
	border-radius: 8px;
	min-width: 180px;
}

.file-link {
	text-decoration: none;
	color: inherit;
}

.file-link:hover {
	background: rgba(0,0,0,0.25);
}

.file-icon {
	font-size: 1.8rem;
}

.file-info {
	display: flex;
	flex-direction: column;
	gap: 2px;
	flex: 1;
	min-width: 0;
}

.file-name {
	font-size: 0.9rem;
	font-weight: 500;
	color: #dfe6e9;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.file-hint {
	font-size: 0.75rem;
	color: #b2bec3;
}

.audio-player {
	width: 100%;
	max-width: 220px;
	height: 32px;
	margin-top: 4px;
}

@media (max-width: 500px) {
	.message-bubble {
		max-width: 85%;
	}
	.message-actions {
		position: static;
		margin-top: 4px;
		display: flex;
		background: transparent;
		box-shadow: none;
		border: none;
		padding: 0;
	}
	.own-message .message-actions {
		justify-content: flex-end;
	}
	.attach-menu {
		bottom: 55px;
	}
}
</style>
