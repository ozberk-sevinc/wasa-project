<script>
import { conversationAPI, messageAPI } from "@/services/api.js";
import GroupInfoPanel from "@/components/GroupInfoPanel.vue";

export default {
	name: "ChatView",
	components: {
		GroupInfoPanel,
	},
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
			showInputEmojiPicker: false,
			showAttachMenu: false,
			commonEmojis: ["👍", "❤️", "😂", "😮", "😢", "🙏", "🔥", "✅"],
			inputEmojis: [
				"😀", "😃", "😄", "😁", "😆", "😅", "🤣", "😂",
				"😊", "😇", "🙂", "😉", "😍", "🥰", "😘", "😋",
				"😎", "🤓", "🧐", "😏", "😢", "😭", "😤", "😡",
				"🤔", "🤨", "😐", "😑", "😴", "😮", "😲", "😱",
				"👍", "👎", "👏", "🙌", "🤝", "✌️", "🤞", "🤟",
				"❤️", "🧡", "💛", "💚", "💙", "💜", "🖤", "🤍",
				"💯", "🔥", "✨", "⭐", "🌟", "💫", "✅", "❌",
				"🎉", "🎊", "🎈", "🎁", "🎂", "🎵", "🎶", "🎮"
			],
			showGroupInfo: false,
			showForwardDialog: false,
			forwardingMessage: null,
			conversations: [],
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
			this.showInputEmojiPicker = false;
		},

		toggleInputEmojiPicker() {
			this.showInputEmojiPicker = !this.showInputEmojiPicker;
			this.showAttachMenu = false;
		},

		insertEmoji(emoji) {
			const input = this.$refs.messageInput;
			const startPos = input.selectionStart;
			const endPos = input.selectionEnd;
			const textBefore = this.newMessage.substring(0, startPos);
			const textAfter = this.newMessage.substring(endPos);
			this.newMessage = textBefore + emoji + textAfter;
			this.$nextTick(() => {
				const newPos = startPos + emoji.length;
				input.focus();
				input.setSelectionRange(newPos, newPos);
			});
			this.showInputEmojiPicker = false;
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

		openGroupInfo() {
			if (this.conversation?.type === "group") {
				this.showGroupInfo = true;
			}
		},

		handleGroupUpdated(updatedGroup) {
			if (this.conversation) {
				this.conversation.title = updatedGroup.name;
				this.conversation.photoUrl = updatedGroup.photoUrl;
			}
		},

		handleLeftGroup() {
			this.$router.push("/");
		},

		openForwardDialog(message) {
			this.forwardingMessage = message;
			this.showForwardDialog = true;
			this.loadConversations();
		},

		async loadConversations() {
			try {
				const response = await conversationAPI.getAll();
				this.conversations = response.data || [];
			} catch (e) {
				console.error("Failed to load conversations:", e);
			}
		},

		async forwardMessage(targetConversationId) {
			if (!this.forwardingMessage) return;

			try {
				await messageAPI.forward(
					this.conversationId,
					this.forwardingMessage.id,
					targetConversationId
				);
				alert("Message forwarded successfully!");
				this.showForwardDialog = false;
				this.forwardingMessage = null;
			} catch (e) {
				alert(e.response?.data?.message || "Failed to forward message");
			}
		},

		cancelForward() {
			this.showForwardDialog = false;
			this.forwardingMessage = null;
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
			<div 
				class="chat-info" 
				v-if="conversation"
				:class="{ 'clickable': conversation.type === 'group' }"
				@click="openGroupInfo"
				:title="conversation.type === 'group' ? 'Click for group info' : ''"
			>
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
						<button @click="openForwardDialog(message)" title="Forward">↪️</button>
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
			<!-- Emoji button -->
			<div class="emoji-wrapper">
				<button class="btn-emoji" @click="toggleInputEmojiPicker" type="button">
					😊
				</button>
				<div v-if="showInputEmojiPicker" class="input-emoji-picker">
					<button
						v-for="emoji in inputEmojis"
						:key="emoji"
						@click="insertEmoji(emoji)"
						type="button"
					>
						{{ emoji }}
					</button>
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

		<!-- Group Info Panel -->
		<GroupInfoPanel
			v-if="conversation?.type === 'group'"
			:show="showGroupInfo"
			:group-id="conversationId"
			:current-user-id="currentUser?.id"
			@close="showGroupInfo = false"
			@group-updated="handleGroupUpdated"
			@left-group="handleLeftGroup"
		/>

		<!-- Forward Message Dialog -->
		<div v-if="showForwardDialog" class="modal fade show d-block" tabindex="-1" style="background-color: rgba(0,0,0,0.5)">
			<div class="modal-dialog">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">Forward Message</h5>
						<button type="button" class="btn-close" @click="cancelForward"></button>
					</div>
					<div class="modal-body">
						<p class="text-muted mb-3">Select a conversation to forward this message to:</p>
						<div class="list-group">
							<button
								v-for="conv in conversations"
								:key="conv.id"
								class="list-group-item list-group-item-action d-flex align-items-center"
								@click="forwardMessage(conv.id)"
								:disabled="conv.id === conversationId"
							>
								<div class="avatar-placeholder small me-3">
									{{ getInitials(conv.title) }}
								</div>
								<div class="flex-grow-1">
									<div class="fw-bold">{{ conv.title }}</div>
									<small class="text-muted">{{ conv.type === 'group' ? 'Group' : 'Direct' }}</small>
								</div>
								<span v-if="conv.id === conversationId" class="badge bg-secondary">Current</span>
							</button>
						</div>
						<div v-if="conversations.length === 0" class="text-center text-muted py-3">
							No other conversations available
						</div>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-secondary" @click="cancelForward">Cancel</button>
					</div>
				</div>
			</div>
		</div>
		</div>
	</div>
</template>

<style scoped>
.chat-container {
	height: 100vh;
	height: 100dvh;
	display: flex;
	flex-direction: column;
	background: #1a1d29;
}

.chat-header {
	background: #252435;
	color: #e2e8f0;
	padding: 10px 12px;
	display: flex;
	align-items: center;
	gap: 8px;
	border-bottom: 1px solid #3d3a52;
	flex-shrink: 0;
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

.chat-info {
	flex: 1;
	display: flex;
	align-items: center;
	gap: 10px;
	min-width: 0;
}

.chat-info.clickable {
	cursor: pointer;
	border-radius: 8px;
	padding: 4px 8px;
	margin: -4px -8px;
	transition: background 0.2s;
}

.chat-info.clickable:hover {
	background: rgba(255, 255, 255, 0.1);
}

.avatar-placeholder.small {
	width: 38px;
	height: 38px;
	border-radius: 50%;
	background: #8b5cf6;
	color: #1a1d29;
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
	color: #64748b;
}

.chat-header .btn-light {
	background: #3d3a52;
	color: #cbd5e1;
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
	color: #cbd5e1;
}

.spinner-border {
	color: #8b5cf6 !important;
}

.messages-container {
	flex: 1;
	overflow-y: auto;
	-webkit-overflow-scrolling: touch;
	padding: 12px;
	background: #1a1d29;
}

.no-messages {
	text-align: center;
	padding: 32px 16px;
	color: #64748b;
}

.date-separator {
	text-align: center;
	margin: 16px 0;
}

.date-separator span {
	background: #3d3a52;
	padding: 4px 12px;
	border-radius: 12px;
	font-size: 0.75rem;
	color: #cbd5e1;
}

.message-wrapper {
	display: flex;
	flex-direction: column;
	align-items: flex-start;
	margin-bottom: 8px;
	position: relative;
	width: 100%;
}

.message-wrapper.own-message {
	align-items: flex-end;
}

.message-bubble {
	max-width: 80%;
	background: #252435;
	border-radius: 12px;
	border-top-left-radius: 4px;
	padding: 8px 12px;
	position: relative;
	border: 1px solid #3d3a52;
}

.own-message .message-bubble {
	background: #8b5cf6;
	border-color: #7c3aed;
	border-radius: 12px;
	border-top-right-radius: 4px;
}

.own-message .message-text {
	color: #1a1d29;
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
	background: #8b5cf6;
	border-radius: 2px;
	margin-right: 8px;
	flex-shrink: 0;
}

.reply-content {
	font-size: 0.8rem;
	min-width: 0;
}

.reply-content strong {
	color: #8b5cf6;
}

.own-message .reply-content strong {
	color: #1a1d29;
}

.reply-content p {
	margin: 0;
	color: #cbd5e1;
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
	color: #8b5cf6;
	margin-bottom: 2px;
}

.message-text {
	margin: 0;
	word-wrap: break-word;
	color: #e2e8f0;
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
	color: #64748b;
}

.message-status {
	font-size: 0.75rem;
	color: #64748b;
}

.message-status.status-read {
	color: #8b5cf6;
}

.own-message .message-status {
	color: rgba(30, 39, 46, 0.5);
}

.own-message .message-status.status-read {
	color: #1a1d29;
}

.message-reactions {
	display: flex;
	flex-wrap: wrap;
	gap: 4px;
	margin-top: 4px;
}

.reaction-badge {
	background: rgba(139, 92, 246, 0.2);
	border: 1px solid rgba(139, 92, 246, 0.3);
	padding: 3px 8px;
	border-radius: 12px;
	font-size: 0.9rem;
	cursor: pointer;
	transition: all 0.2s;
	display: inline-flex;
	align-items: center;
	gap: 2px;
}

.reaction-badge:hover {
	background: rgba(139, 92, 246, 0.35);
	border-color: rgba(139, 92, 246, 0.5);
	transform: scale(1.1);
}

.own-message .reaction-badge {
	background: rgba(26, 29, 41, 0.3);
	border-color: rgba(26, 29, 41, 0.5);
}

.own-message .reaction-badge:hover {
	background: rgba(26, 29, 41, 0.5);
	border-color: rgba(26, 29, 41, 0.7);
}

.message-actions {
	display: none;
	position: absolute;
	top: 50%;
	transform: translateY(-50%);
	right: 8px;
	background: #252435;
	border-radius: 16px;
	padding: 6px 8px;
	box-shadow: 0 2px 12px rgba(0, 0, 0, 0.5);
	border: 1px solid #3d3a52;
	z-index: 10;
}

.own-message .message-actions {
	right: auto;
	left: 8px;
}

.message-wrapper:hover .message-actions {
	display: flex;
	gap: 2px;
}

.message-actions button {
	background: none;
	border: none;
	cursor: pointer;
	font-size: 1.1rem;
	padding: 6px 8px;
	border-radius: 6px;
	transition: all 0.2s;
}

.message-actions button:hover {
	background: #3d3a52;
	transform: scale(1.1);
}

.emoji-picker {
	position: absolute;
	top: 50%;
	transform: translateY(-50%);
	right: 180px;
	background: #252435;
	border-radius: 12px;
	padding: 8px;
	box-shadow: 0 4px 16px rgba(0, 0, 0, 0.5);
	border: 1px solid #3d3a52;
	display: flex;
	flex-wrap: wrap;
	gap: 4px;
	z-index: 101;
	max-width: 200px;
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
	background: #3d3a52;
}

.own-message .emoji-picker {
	right: auto;
	left: 180px;
}

.reply-preview {
	background: #252435;
	padding: 10px 14px;
	display: flex;
	justify-content: space-between;
	align-items: center;
	border-left: 3px solid #8b5cf6;
	border-top: 1px solid #3d3a52;
	flex-shrink: 0;
}

.reply-preview-content strong {
	color: #8b5cf6;
	display: block;
	font-size: 0.8rem;
}

.reply-preview-content p {
	margin: 0;
	font-size: 0.8rem;
	color: #cbd5e1;
}

.btn-cancel-reply {
	background: none;
	border: none;
	font-size: 1.1rem;
	cursor: pointer;
	color: #64748b;
	padding: 4px;
}

.btn-cancel-reply:hover {
	color: #e2e8f0;
}

.input-area {
	padding: 10px 12px;
	background: #252435;
	display: flex;
	gap: 8px;
	border-top: 1px solid #3d3a52;
	flex-shrink: 0;
	align-items: center;
}

.attach-wrapper,
.emoji-wrapper {
	position: relative;
}

.btn-attach,
.btn-emoji {
	background: none;
	border: none;
	font-size: 1.4rem;
	cursor: pointer;
	padding: 6px 8px;
	border-radius: 8px;
	transition: background 0.2s;
}

.btn-attach:hover,
.btn-emoji:hover {
	background: #3d3a52;
}

.attach-menu {
	position: absolute;
	bottom: 50px;
	left: 0;
	background: #252435;
	border: 1px solid #3d3a52;
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
	color: #e2e8f0;
	padding: 10px 14px;
	text-align: left;
	cursor: pointer;
	border-radius: 6px;
	font-size: 0.9rem;
	white-space: nowrap;
}

.attach-menu button:hover {
	background: #3d3a52;
}

.input-emoji-picker {
	position: absolute;
	bottom: 50px;
	left: 0;
	background: #252435;
	border: 1px solid #3d3a52;
	border-radius: 10px;
	padding: 12px;
	display: grid;
	grid-template-columns: repeat(8, 1fr);
	gap: 6px;
	z-index: 100;
	box-shadow: 0 4px 12px rgba(0,0,0,0.4);
	max-width: 320px;
	max-height: 300px;
	overflow-y: auto;
}

.input-emoji-picker button {
	background: none;
	border: none;
	font-size: 1.5rem;
	cursor: pointer;
	padding: 6px;
	border-radius: 6px;
	transition: background 0.2s;
	line-height: 1;
}

.input-emoji-picker button:hover {
	background: #3d3a52;
	transform: scale(1.2);
}

.input-area .form-control {
	border-radius: 20px;
	padding: 10px 16px;
	background: #1a1d29;
	border: 1px solid #3d3a52;
	color: #e2e8f0;
	flex: 1;
}

.input-area .form-control:focus {
	border-color: #8b5cf6;
	box-shadow: none;
	outline: none;
}

.input-area .form-control::placeholder {
	color: #64748b;
}

.btn-send {
	border-radius: 20px;
	padding: 10px 18px;
	background: #8b5cf6;
	border: none;
	color: #1a1d29;
	font-weight: 500;
	flex-shrink: 0;
}

.btn-send:hover:not(:disabled) {
	background: #7c3aed;
}

.btn-send:disabled {
	background: #3d3a52;
	color: #64748b;
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
	color: #e2e8f0;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.file-hint {
	font-size: 0.75rem;
	color: #cbd5e1;
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
