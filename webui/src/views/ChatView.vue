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
			pendingPhotoUrl: null,
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
			// Auto-refresh (fallback when WebSocket disconnected)
			refreshInterval: null,
			// WebSocket for real-time updates
			ws: null,
			wsConnected: false,
			wsReconnectTimer: null,
			// Context menu
			contextMenuMessageId: null,
			contextMenuTimer: null,
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
	watch: {
		conversationId() {
			// Clear existing interval and restart with new conversation
			if (this.refreshInterval) {
				clearInterval(this.refreshInterval);
			}
			
			// Disconnect and reconnect WebSocket for new conversation
			this.disconnectWebSocket();
			this.connectWebSocket();
			
			this.loadConversation(); // Show spinner when switching conversations
			this.refreshInterval = setInterval(() => {
				this.loadConversation(true); // Silent refresh every 5 seconds
			}, 5000);
		},
	},
	mounted() {
		const userData = localStorage.getItem("wasatext_user");
		if (userData) {
			this.currentUser = JSON.parse(userData);
			console.log("Logged in as:", this.currentUser.name, "ID:", this.currentUser.id);
		} else {
			console.warn("No user data found in localStorage");
		}
		
		console.log("Loading conversation ID:", this.conversationId);
		this.loadConversation(); // Initial load with spinner
		
				// Connect WebSocket for real-time messaging
				this.connectWebSocket();
				// Expose WebSocket globally for child components (GroupInfoPanel)
				Object.defineProperty(window, 'WS_GLOBAL', {
					configurable: true,
					get: () => this.ws
				});
		
		// Auto-refresh messages every 5 seconds as fallback (disabled when WebSocket is active)
		this.refreshInterval = setInterval(() => {
			this.loadConversation(true); // Silent refresh, no spinner or scroll jump
		}, 5000);
	},
	beforeUnmount() {
		// Clean up WebSocket connection
		this.disconnectWebSocket();
		
		// Clean up interval when component is destroyed
		if (this.refreshInterval) {
			clearInterval(this.refreshInterval);
		}
	},
	methods: {
		async loadConversation(silent = false) {
			// Only show loading spinner on initial load, not on auto-refresh
			if (!silent) {
				this.loading = true;
				console.log("🔄 Loading conversation, setting loading=true");
			}
			this.error = null;
			
			console.log("📡 Fetching conversation ID:", this.conversationId);
			const startTime = Date.now();
			
			try {
				const response = await conversationAPI.getById(this.conversationId);
				const loadTime = Date.now() - startTime;
				console.log(`✅ Conversation loaded in ${loadTime}ms`);
				
				const oldMessageCount = this.messages.length;
				this.conversation = response.data;
				this.messages = response.data.messages || [];
				console.log(`📨 Loaded ${this.messages.length} messages`);
				// Only auto-scroll if: initial load, or new messages arrived and we're already at bottom
				if (!silent || this.messages.length > oldMessageCount) {
					this.$nextTick(() => {
						const container = this.$refs.messagesContainer;
						if (container) {
							// Check if user is already scrolled to bottom (within 100px)
							const isAtBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 100;
							if (isAtBottom || !silent) {
								this.scrollToBottom();
							}
						}
					});
				}
			} catch (e) {
				if (!silent) {
					console.error("❌ Error loading conversation:", e);
					console.error("Response data:", e.response?.data);
					console.error("Status:", e.response?.status);
					this.error = e.response?.data?.message || "Failed to load conversation";
				}
			} finally {
				if (!silent) {
					this.loading = false;
					console.log("✅ Loading complete, setting loading=false");
				}
			}
		},

		connectWebSocket() {
			const token = localStorage.getItem("wasatext_token");
			if (!token) {
				console.warn("No token for WebSocket connection");
				return;
			}

			const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
			const wsUrl = `${protocol}//${window.location.host}/api/ws?token=${encodeURIComponent(token)}`;
			
			console.log("🔌 Connecting to WebSocket...");
			this.ws = new WebSocket(wsUrl);

			this.ws.onopen = () => {
				console.log("✅ WebSocket connected - real-time messaging enabled");
				this.wsConnected = true;
				// Stop polling when WebSocket is active
				if (this.refreshInterval) {
					clearInterval(this.refreshInterval);
					this.refreshInterval = null;
				}
			};

			this.ws.onmessage = (event) => {
				try {
					const data = JSON.parse(event.data);
					
					if (data.type === "new_message") {
						const message = data.payload;
						// Only add if it's for the current conversation
						if (message.conversationId === this.conversationId) {
							// Check if message already exists
							const exists = this.messages.some(m => m.id === message.id);
							if (!exists) {
								console.log("📨 Real-time message received:", message.id);
								this.messages.push(message);
								this.$nextTick(() => {
									const container = this.$refs.messagesContainer;
									if (container) {
										const isAtBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 100;
										if (isAtBottom) {
											this.scrollToBottom();
										}
									}
								});
							}
						}
					} else if (data.type === "messages_read") {
						// Another user has read messages in this conversation
						const payload = data.payload;
						if (payload.conversationId === this.conversationId) {
							console.log("✓✓ Messages read by user:", payload.readByUserId);
							// Update only the messages that are now fully read by everyone
							const fullyReadIds = payload.fullyReadMessageIds || [];
							this.messages.forEach(msg => {
								if (fullyReadIds.includes(msg.id) && msg.status !== "read") {
									msg.status = "read";
									console.log("📬 Message", msg.id, "now fully read by all participants");
								}
							});
						}
					} else if (data.type === "profile_updated") {
						// Another user updated their profile name or photo
						const payload = data.payload;
						// Update name and photo in all messages from this user
						this.messages.forEach(msg => {
							if (msg.sender?.id === payload.userId) {
								if (payload.name) msg.sender.name = payload.name;
								if (payload.photoUrl !== undefined) msg.sender.photoUrl = payload.photoUrl;
							}
							// Also update reactions from this user
							if (msg.reactions) {
								msg.reactions.forEach(r => {
									if (r.user?.id === payload.userId) {
										if (payload.name) r.user.name = payload.name;
										if (payload.photoUrl !== undefined) r.user.photoUrl = payload.photoUrl;
									}
								});
							}
						});
						// Update conversation participants
						if (this.conversation?.participants) {
							this.conversation.participants.forEach(p => {
								if (p.id === payload.userId) {
									if (payload.name) p.name = payload.name;
									if (payload.photoUrl !== undefined) p.photoUrl = payload.photoUrl;
								}
							});
						}
						// Update conversation title/photo for direct chats
						if (this.conversation?.type === "direct") {
							const otherParticipant = this.conversation.participants?.find(p => p.id !== this.currentUser?.id);
							if (otherParticipant?.id === payload.userId) {
								if (payload.name) this.conversation.title = payload.name;
								if (payload.photoUrl !== undefined) this.conversation.photoUrl = payload.photoUrl;
							}
						}
					} else if (data.type === "reaction_added") {
						const payload = data.payload;
						if (payload.conversationId === this.conversationId) {
							const msg = this.messages.find(m => m.id === payload.messageId);
							if (msg) {
								msg.reactions = msg.reactions || [];
								// Remove existing reaction from same user (they can only have one)
								const reactUserId = payload.reaction?.user?.id;
								if (reactUserId) {
									msg.reactions = msg.reactions.filter(r => r.user?.id !== reactUserId);
								}
								msg.reactions.push(payload.reaction);
							}
						}
					} else if (data.type === "reaction_removed") {
						const payload = data.payload;
						if (payload.conversationId === this.conversationId) {
							const msg = this.messages.find(m => m.id === payload.messageId);
							if (msg && msg.reactions) {
								msg.reactions = msg.reactions.filter(r => r.id !== payload.reactionId);
							}
						}
					} else if (data.type === "group_updated") {
						const payload = data.payload;
						if (this.conversation && this.conversationId === payload.groupId) {
							if (payload.name) this.conversation.title = payload.name;
							if (payload.photoUrl !== undefined) this.conversation.photoUrl = payload.photoUrl;
						}
					}
				} catch (e) {
					console.error("Error parsing WebSocket message:", e);
				}
			};

			this.ws.onerror = (error) => {
				console.error("❌ WebSocket error:", error);
				this.wsConnected = false;
			};

			this.ws.onclose = () => {
				console.log("🔌 WebSocket disconnected - falling back to polling");
				this.wsConnected = false;
				
				// Start polling fallback if not already running
				if (!this.refreshInterval) {
					this.refreshInterval = setInterval(() => {
						this.loadConversation(true);
					}, 5000);
				}
				
				// Attempt to reconnect after 3 seconds
				if (this.wsReconnectTimer) {
					clearTimeout(this.wsReconnectTimer);
				}
				this.wsReconnectTimer = setTimeout(() => {
					if (this.$route.name === "ChatView") {
						console.log("🔄 Attempting WebSocket reconnect...");
						this.connectWebSocket();
					}
				}, 3000);
			};
		},

		disconnectWebSocket() {
			if (this.ws) {
				this.ws.close();
				this.ws = null;
				this.wsConnected = false;
			}
			if (this.wsReconnectTimer) {
				clearTimeout(this.wsReconnectTimer);
				this.wsReconnectTimer = null;
			}
		},

		async sendMessage() {
			if ((!this.newMessage.trim() && !this.pendingPhotoUrl) || this.sending) return;

			this.sending = true;
			try {
				let payload = {};
				
				// If we have a pending photo
				if (this.pendingPhotoUrl) {
					if (this.newMessage.trim()) {
						// Send as text with photo attachment
						payload = {
							contentType: "text",
							text: this.newMessage.trim(),
							photoUrl: this.pendingPhotoUrl,
						};
					} else {
						// Send as photo only
						payload = {
							contentType: "photo",
							photoUrl: this.pendingPhotoUrl,
						};
					}
				} else {
					// Text only
					payload = {
						contentType: "text",
						text: this.newMessage.trim(),
					};
				}
				
				if (this.replyingTo) {
					payload.replyToMessageId = this.replyingTo.id;
				}

				const response = await messageAPI.send(this.conversationId, payload);
				// Don't manually add message - let WebSocket handle it for consistency
				// this.messages.push(response.data);
				this.newMessage = "";
				this.pendingPhotoUrl = null;
				this.replyingTo = null;
				// WebSocket will add the message and scroll automatically
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

			this.sending = true;
			try {
				if (type === "photo") {
					// Upload photo to server and store URL for later
					const uploadResponse = await messageAPI.uploadPhoto(this.conversationId, file);
					this.pendingPhotoUrl = uploadResponse.data.photoUrl;
					// Don't send immediately - wait for user to add text or press send
					this.$refs.messageInput?.focus();
				} else if (type === "audio") {
					// Audio not supported - only text and photo
					alert("Audio messages are not supported. Only text and photo messages are allowed.");
				} else {
					// Document/file not supported - only text and photo
					alert("Document messages are not supported. Only text and photo messages are allowed.");
				}
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
			return "📁";
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
				// Update message reactions - replace existing user reaction if any
				const msg = this.messages.find((m) => m.id === messageId);
				if (msg) {
					msg.reactions = msg.reactions || [];
					// Remove any existing reaction from current user
					msg.reactions = msg.reactions.filter((r) => r.user?.id !== this.currentUser?.id);
					// Add the new reaction
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

		canRemoveReaction(reaction) {
			// User can only remove their own reactions
			return reaction.user?.id === this.currentUser?.id;
		},

		groupReactionsByEmoji(reactions) {
			if (!reactions || reactions.length === 0) return [];
			
			const grouped = {};
			reactions.forEach(reaction => {
				if (!grouped[reaction.emoji]) {
					grouped[reaction.emoji] = {
						emoji: reaction.emoji,
						count: 0,
						users: [],
						currentUserReaction: null
					};
				}
				grouped[reaction.emoji].count++;
				grouped[reaction.emoji].users.push(reaction.user?.name || 'Unknown');
				if (reaction.user?.id === this.currentUser?.id) {
					grouped[reaction.emoji].currentUserReaction = reaction;
				}
			});
			
			return Object.values(grouped);
		},

		getUserReactionForMessage(message) {
			// Find ANY reaction from the current user on this message
			if (!message.reactions) return null;
			return message.reactions.find(r => r.user?.id === this.currentUser?.id);
		},

		async toggleReaction(messageId, emoji, currentUserReaction) {
			// Find the message
			const msg = this.messages.find(m => m.id === messageId);
			if (!msg) return;

			// Check if user has ANY reaction on this message
			const existingReaction = this.getUserReactionForMessage(msg);

			if (existingReaction && existingReaction.emoji === emoji) {
				// User already reacted with this exact emoji, remove it
				await this.removeReaction(messageId, existingReaction.id);
			} else {
				// User hasn't reacted with this emoji (or hasn't reacted at all)
				// addReaction will replace any existing reaction
				await this.addReaction(messageId, emoji);
			}
		},

		setReply(message) {
			this.replyingTo = message;
			this.$refs.messageInput?.focus();
		},

		cancelReply() {
			this.replyingTo = null;
		},

		cancelPendingPhoto() {
			this.pendingPhotoUrl = null;
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
		getPhotoUrl(url) {
			if (!url) return null;
			// If already absolute, return as is
			if (url.startsWith('http://') || url.startsWith('https://')) {
				return url;
			}
			// Convert relative URL to absolute by prepending API base URL
			return __API_URL__ + url;
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

		showContextMenu(messageId, event) {
			event.preventDefault();
			// Clear any existing timer
			if (this.contextMenuTimer) {
				clearTimeout(this.contextMenuTimer);
			}
			// Close emoji picker if open
			this.showEmojiPicker = null;
			// Show context menu for this message
			this.contextMenuMessageId = messageId;
			// Auto-hide after 2 seconds
			this.contextMenuTimer = setTimeout(() => {
				this.contextMenuMessageId = null;
			}, 2000);
		},

		hideContextMenu() {
			if (this.contextMenuTimer) {
				clearTimeout(this.contextMenuTimer);
			}
			this.contextMenuMessageId = null;
		},

		keepContextMenuAlive() {
			// When user hovers over the menu, keep it visible
			if (this.contextMenuTimer) {
				clearTimeout(this.contextMenuTimer);
			}
			// Reset the timer
			this.contextMenuTimer = setTimeout(() => {
				this.contextMenuMessageId = null;
			}, 2000);
		},

		getInitials(name) {
			if (!name) return "?";
			const words = name.trim().split(/\s+/);
			if (words.length === 1) {
				return words[0].substring(0, 2).toUpperCase();
			}
			return (words[0][0] + words[1][0]).toUpperCase();
		},

		scrollToMessage(messageId) {
			// Find the message element and scroll to it
			const messageElement = document.querySelector(`[data-message-id="${messageId}"]`);
			if (messageElement) {
				messageElement.scrollIntoView({ behavior: "smooth", block: "center" });
				// Highlight briefly
				messageElement.style.backgroundColor = "rgba(139, 92, 246, 0.2)";
				setTimeout(() => {
					messageElement.style.backgroundColor = "";
				}, 1500);
			}
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
        v-if="conversation" 
        class="chat-info"
        :class="{ 'clickable': conversation.type === 'group' }"
        :title="conversation.type === 'group' ? 'Click for group info' : ''"
        @click="openGroupInfo"
      >
        <div class="chat-avatar">
          <img 
            v-if="conversation.photoUrl" 
            :src="getPhotoUrl(conversation.photoUrl)" 
            :alt="conversation.title"
            class="avatar-img-small"
          >
          <div v-else class="avatar-placeholder small">
            {{ getInitials(conversation.title) }}
          </div>
        </div>
        <div class="chat-title-section">
          <h2>{{ conversation.title }}</h2>
          <span v-if="conversation.type === 'group'" class="chat-subtitle">
            {{ conversation.participants?.length }} members
          </span>
        </div>
      </div>
      <button class="btn btn-light btn-sm" @click="loadConversation">🔄</button>
    </header>

    <!-- Loading -->
    <div v-if="loading" class="chat-loading">
      <div class="spinner-border text-primary" />
      <p>Loading messages...</p>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="chat-error">
      <p>{{ error }}</p>
      <button class="btn btn-primary" @click="loadConversation">Retry</button>
    </div>

    <!-- Messages -->
    <div v-else ref="messagesContainer" class="messages-container">
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
          :data-message-id="message.id"
        >
          <!-- Sender avatar (for messages from others) -->
          <div v-if="!isOwnMessage(message)" class="message-avatar">
            <img 
              v-if="message.sender?.photoUrl" 
              :src="getPhotoUrl(message.sender.photoUrl)" 
              :alt="message.sender?.name"
              class="avatar-img-small"
            >
            <div v-else class="avatar-placeholder-small">
              {{ getInitials(message.sender?.name) }}
            </div>
          </div>

          <div 
            class="message-bubble"
            @contextmenu="showContextMenu(message.id, $event)"
          >
            <!-- Reply reference -->
            <div
              v-if="message.repliedToMessageId"
              class="reply-reference"
              @click="scrollToMessage(message.repliedToMessageId)"
            >
              <div class="reply-bar" />
              <div class="reply-content">
                <strong>{{ getRepliedMessage(message.repliedToMessageId)?.sender?.name || "Unknown" }}</strong>
                <p>
                  <span v-if="getRepliedMessage(message.repliedToMessageId)?.photoUrl" class="reply-photo-icon">📷 </span>
                  {{ getRepliedMessage(message.repliedToMessageId)?.text || (getRepliedMessage(message.repliedToMessageId)?.photoUrl ? "Photo" : "Message") }}
                </p>
              </div>
            </div>

            <!-- Forwarded marker -->
            <div v-if="message.isForwarded" class="forwarded-marker">
              <span class="forwarded-icon">↪️</span>
              <span class="forwarded-text">Forwarded</span>
            </div>

            <!-- Sender name (for group messages, if not own message) -->
            <div 
              v-if="conversation?.type === 'group' && !isOwnMessage(message)" 
              class="sender-name"
            >
              {{ message.sender?.name }}
            </div>

            <!-- Content -->
            <div class="message-content">
              <!-- Photo -->
              <img 
                v-if="message.photoUrl"
                :src="getPhotoUrl(message.photoUrl)"
                class="message-photo"
                alt="Photo"
              >
              <!-- Text -->
              <p v-if="message.text" class="message-text">{{ message.text }}</p>
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
                v-for="group in groupReactionsByEmoji(message.reactions)"
                :key="group.emoji"
                class="reaction-badge"
                :class="{ 'own-reaction': group.currentUserReaction }"
                :title="group.users.join(', ')"
                @click="toggleReaction(message.id, group.emoji, group.currentUserReaction)"
              >
                {{ group.emoji }}
                <span v-if="group.count > 1" class="reaction-count">{{ group.count }}</span>
              </span>
            </div>
          </div>

          <!-- Message actions (show on right-click) -->
          <div 
            v-if="contextMenuMessageId === message.id"
            class="message-actions context-menu"
            @mouseenter="keepContextMenuAlive"
            @mouseleave="hideContextMenu"
          >
            <button title="Reply" @click="setReply(message); hideContextMenu()">
              <span class="action-icon">↩️</span>
              <span class="action-label">Reply</span>
            </button>
            <button title="React" @click="showEmojiPicker = message.id; hideContextMenu()">
              <span class="action-icon">😊</span>
              <span class="action-label">React</span>
            </button>
            <button title="Forward" @click="openForwardDialog(message); hideContextMenu()">
              <span class="action-icon">↪️</span>
              <span class="action-label">Forward</span>
            </button>
            <button
              v-if="isOwnMessage(message)"
              title="Delete"
              class="delete-btn"
              @click="deleteMessage(message.id); hideContextMenu()"
            >
              <span class="action-icon">🗑️</span>
              <span class="action-label">Delete</span>
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

    <!-- Photo preview -->
    <div v-if="pendingPhotoUrl" class="photo-preview">
      <div class="photo-preview-content">
        <img :src="getPhotoUrl(pendingPhotoUrl)" alt="Preview" class="photo-preview-image">
        <span class="photo-preview-label">📷 Photo attached</span>
      </div>
      <button class="btn-cancel-photo" @click="cancelPendingPhoto">✕</button>
    </div>

    <!-- Hidden file inputs -->
    <input
      ref="photoInput"
      type="file"
      accept="image/*"
      style="display: none"
      @change="(e) => handleFileSelect(e, 'photo')"
    >

    <!-- Input area -->
    <div class="input-area">
      <!-- Attachment button -->
      <div class="attach-wrapper">
        <button class="btn-attach" type="button" @click="toggleAttachMenu">
          📎
        </button>
        <div v-if="showAttachMenu" class="attach-menu">
          <button @click="triggerFileInput('photo')">📷 Photo</button>
          <button @click="sendPhotoUrl()">🔗 URL</button>
        </div>
      </div>
      <!-- Emoji button -->
      <div class="emoji-wrapper">
        <button class="btn-emoji" type="button" @click="toggleInputEmojiPicker">
          😊
        </button>
        <div v-if="showInputEmojiPicker" class="input-emoji-picker">
          <button
            v-for="emoji in inputEmojis"
            :key="emoji"
            type="button"
            @click="insertEmoji(emoji)"
          >
            {{ emoji }}
          </button>
        </div>
      </div>
      <input
        ref="messageInput"
        v-model="newMessage"
        type="text"
        class="form-control"
        placeholder="Type a message..."
        :disabled="sending"
        @keyup.enter="sendMessage"
      >
      <button
        class="btn btn-primary btn-send"
        :disabled="!newMessage.trim() || sending"
        @click="sendMessage"
      >
        <span v-if="sending" class="spinner-border spinner-border-sm" />
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
              <button type="button" class="btn-close" @click="cancelForward" />
            </div>
            <div class="modal-body">
              <p class="text-muted mb-3">Select a conversation to forward this message to:</p>
              <div class="list-group">
                <button
                  v-for="conv in conversations"
                  :key="conv.id"
                  class="list-group-item list-group-item-action d-flex align-items-center"
                  :disabled="conv.id === conversationId"
                  @click="forwardMessage(conv.id)"
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

.chat-avatar {
	width: 38px;
	height: 38px;
	flex-shrink: 0;
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
	flex-direction: row;
	align-items: flex-start;
	margin-bottom: 8px;
	position: relative;
	width: 100%;
	gap: 8px;
}

.message-wrapper.own-message {
	flex-direction: row-reverse;
	justify-content: flex-start;
}

.message-avatar {
	flex-shrink: 0;
	width: 32px;
	height: 32px;
	margin-top: 0;
}

.avatar-img-small {
	width: 32px;
	height: 32px;
	border-radius: 50%;
	object-fit: cover;
}

.avatar-placeholder-small {
	width: 32px;
	height: 32px;
	border-radius: 50%;
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	display: flex;
	align-items: center;
	justify-content: center;
	color: white;
	font-size: 12px;
	font-weight: 600;
	text-transform: uppercase;
}

.message-bubble {
	max-width: 80%;
	background: #252435;
	border-radius: 12px;
	border-top-left-radius: 4px;
	padding: 8px 12px;
	position: relative;
	border: 1px solid #3d3a52;
	cursor: context-menu;
	transition: border-color 0.2s;
}

.message-bubble:hover {
	border-color: #4c4861;
}

.own-message .message-bubble {
	background: #8b5cf6;
	border-color: #7c3aed;
	border-radius: 12px;
	border-top-right-radius: 4px;
}

.own-message .message-bubble:hover {
	border-color: #9f7cf7;
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

.reply-photo-icon {
	opacity: 0.8;
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

.forwarded-marker {
	display: flex;
	align-items: center;
	gap: 4px;
	font-size: 0.7rem;
	color: #64748b;
	margin-bottom: 4px;
	font-style: italic;
}

.forwarded-icon {
	font-size: 0.8rem;
}

.own-message .forwarded-marker {
	color: rgba(30, 39, 46, 0.6);
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

.reaction-count {
	font-size: 0.75rem;
	font-weight: 600;
	margin-left: 2px;
	opacity: 0.9;
}

.reaction-badge:hover {
	background: rgba(139, 92, 246, 0.35);
	border-color: rgba(139, 92, 246, 0.5);
	transform: scale(1.1);
}

.reaction-badge.own-reaction {
	background: rgba(59, 130, 246, 0.3);
	border-color: rgba(59, 130, 246, 0.5);
	font-weight: 600;
}

.reaction-badge.own-reaction:hover {
	background: rgba(59, 130, 246, 0.45);
	border-color: rgba(59, 130, 246, 0.7);
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
	position: relative;
	top: 0;
	background: #1f1d2e;
	border-radius: 12px;
	padding: 4px;
	box-shadow: 0 4px 20px rgba(0, 0, 0, 0.7);
	border: 1px solid #3d3a52;
	z-index: 100;
	display: flex;
	flex-direction: column;
	gap: 2px;
	animation: contextMenuPop 0.15s ease-out;
	min-width: 120px;
	flex-shrink: 0;
	align-self: flex-start;
	margin-top: 8px;
}

@keyframes contextMenuPop {
	0% {
		opacity: 0;
		transform: scale(0.9);
	}
	100% {
		opacity: 1;
		transform: scale(1);
	}
}

.message-actions button {
	background: transparent;
	border: none;
	cursor: pointer;
	padding: 10px 12px;
	border-radius: 8px;
	transition: all 0.2s;
	display: flex;
	align-items: center;
	gap: 10px;
	width: 100%;
	text-align: left;
	color: #e2e8f0;
	font-size: 0.9rem;
}

.message-actions button:hover {
	background: #3d3a52;
}

.message-actions button.delete-btn:hover {
	background: rgba(239, 68, 68, 0.2);
	color: #fca5a5;
}

.message-actions .action-icon {
	font-size: 1.2rem;
	flex-shrink: 0;
}

.message-actions .action-label {
	font-size: 0.9rem;
	font-weight: 500;
}

.emoji-picker {
	position: absolute;
	top: 100%;
	margin-top: 4px;
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

.message-wrapper:not(.own-message) .emoji-picker {
	left: 0;
}

.message-wrapper.own-message .emoji-picker {
	right: 0;
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

.photo-preview {
	background: #252435;
	padding: 10px 14px;
	display: flex;
	justify-content: space-between;
	align-items: center;
	border-left: 3px solid #3b82f6;
	border-top: 1px solid #3d3a52;
	flex-shrink: 0;
}

.photo-preview-content {
	display: flex;
	align-items: center;
	gap: 10px;
}

.photo-preview-image {
	width: 40px;
	height: 40px;
	object-fit: cover;
	border-radius: 4px;
}

.photo-preview-label {
	font-size: 0.85rem;
	color: #3b82f6;
}

.btn-cancel-photo {
	background: none;
	border: none;
	font-size: 1.1rem;
	cursor: pointer;
	color: #64748b;
	padding: 4px;
}

.btn-cancel-photo:hover {
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
