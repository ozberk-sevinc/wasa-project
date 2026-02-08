import { createRouter, createWebHistory } from "vue-router";
import LoginView from "../views/LoginView.vue";
import ConversationsView from "../views/ConversationsView.vue";
import ChatView from "../views/ChatView.vue";
import ProfileView from "../views/ProfileView.vue";

const router = createRouter({
	history: createWebHistory(import.meta.env.BASE_URL),
	routes: [
		{ path: "/login", name: "login", component: LoginView },
		{ path: "/", name: "conversations", component: ConversationsView, meta: { requiresAuth: true } },
		{ path: "/chat/:id", name: "chat", component: ChatView, meta: { requiresAuth: true } },
		{ path: "/profile", name: "profile", component: ProfileView, meta: { requiresAuth: true } },
	],
});

// Auth guard
router.beforeEach((to, from, next) => {
	const token = localStorage.getItem("wasatext_token");
	
	if (to.meta.requiresAuth && !token) {
		next("/login");
	} else if (to.path === "/login" && token) {
		next("/");
	} else {
		next();
	}
});

export default router;
