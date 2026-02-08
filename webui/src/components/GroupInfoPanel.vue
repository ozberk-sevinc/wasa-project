<template>
	<div v-if="show" class="modal fade show d-block" tabindex="-1" style="background-color: rgba(0,0,0,0.5)">
		<div class="modal-dialog modal-dialog-scrollable">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title">Group Info</h5>
					<button type="button" class="btn-close" @click="close"></button>
				</div>
				<div class="modal-body">
					<div v-if="loading" class="text-center py-4">
						<div class="spinner-border text-primary" role="status">
							<span class="visually-hidden">Loading...</span>
						</div>
					</div>

					<div v-else-if="error" class="alert alert-danger">{{ error }}</div>

					<div v-else-if="group">
						<!-- Group Photo -->
						<div class="text-center mb-4">
							<div class="position-relative d-inline-block">
								<img
									:src="group.photoUrl || 'https://via.placeholder.com/120?text=Group'"
									class="rounded-circle"
									style="width: 120px; height: 120px; object-fit: cover"
									alt="Group photo"
								/>
							</div>
						</div>

						<!-- Group Name -->
						<div class="mb-4">
							<div class="d-flex align-items-center justify-content-between">
								<div class="flex-grow-1">
									<label class="form-label fw-bold mb-1">Group Name</label>
									<div v-if="!editingName" class="d-flex align-items-center">
										<h5 class="mb-0 me-2">{{ group.name }}</h5>
										<button
											v-if="isCreator"
											class="btn btn-sm btn-outline-secondary"
											@click="startEditName"
											title="Edit group name"
										>
											✏️
										</button>
									</div>
									<div v-else class="input-group">
										<input
											v-model="newGroupName"
											type="text"
											class="form-control"
											placeholder="Enter new group name"
											@keyup.enter="saveGroupName"
										/>
										<button class="btn btn-success" @click="saveGroupName">Save</button>
										<button class="btn btn-secondary" @click="cancelEditName">Cancel</button>
									</div>
								</div>
							</div>
						</div>

						<!-- Members Section -->
						<div class="mb-4">
							<div class="d-flex justify-content-between align-items-center mb-3">
								<h6 class="mb-0">Members ({{ group.members.length }})</h6>
								<button class="btn btn-sm btn-primary" @click="showAddMember = true">
									➕ Add Member
								</button>
							</div>

							<div class="list-group">
								<div
									v-for="member in group.members"
									:key="member.id"
									class="list-group-item d-flex align-items-center"
								>
									<img
										:src="member.photoUrl || 'https://via.placeholder.com/40?text=U'"
										class="rounded-circle me-3"
										style="width: 40px; height: 40px; object-fit: cover"
										alt="Member photo"
									/>
									<div class="flex-grow-1">
										<div class="fw-bold">{{ member.displayName || member.name }}</div>
										<small class="text-muted">@{{ member.name }}</small>
									</div>
									<span v-if="member.id === group.createdBy" class="badge bg-info">Creator</span>
								</div>
							</div>
						</div>

						<!-- Leave Group Button -->
						<div class="d-grid">
							<button class="btn btn-danger" @click="confirmLeaveGroup">
								🚪 Leave Group
							</button>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- Add Member Modal -->
		<div v-if="showAddMember" class="modal fade show d-block" tabindex="-1" style="background-color: rgba(0,0,0,0.7); z-index: 1060">
			<div class="modal-dialog">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">Add Member</h5>
						<button type="button" class="btn-close" @click="showAddMember = false"></button>
					</div>
					<div class="modal-body">
						<input
							v-model="memberSearchQuery"
							type="text"
							class="form-control mb-3"
							placeholder="Search users..."
							@input="searchUsers"
						/>

						<div v-if="searchingUsers" class="text-center py-3">
							<div class="spinner-border spinner-border-sm" role="status"></div>
						</div>

						<div v-else-if="searchResults.length > 0" class="list-group">
							<button
								v-for="user in searchResults"
								:key="user.id"
								class="list-group-item list-group-item-action d-flex align-items-center"
								@click="addMember(user.id)"
								:disabled="isAlreadyMember(user.id)"
							>
								<img
									:src="user.photoUrl || 'https://via.placeholder.com/40?text=U'"
									class="rounded-circle me-3"
									style="width: 40px; height: 40px; object-fit: cover"
									alt="User photo"
								/>
								<div class="flex-grow-1">
									<div class="fw-bold">{{ user.displayName || user.name }}</div>
									<small class="text-muted">@{{ user.name }}</small>
								</div>
								<span v-if="isAlreadyMember(user.id)" class="badge bg-secondary">Already member</span>
							</button>
						</div>

						<div v-else-if="memberSearchQuery" class="text-muted text-center py-3">
							No users found
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import { groupAPI, userAPI } from "@/services/api.js";

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
	},
};
</script>

<style scoped>
.modal.show {
	display: block;
}
</style>
