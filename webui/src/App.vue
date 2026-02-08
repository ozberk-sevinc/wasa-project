<script setup>
import { RouterView } from "vue-router";
</script>

<script>
export default {
	name: "App",
};
</script>

<template>
	<div id="app">
		<RouterView />
	</div>
</template>

<style>
/* Global styles */
* {
	box-sizing: border-box;
	margin: 0;
	padding: 0;
}

body {
	font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
	background: #1a1d29;
	min-height: 100vh;
	min-height: 100dvh;
	color: #e2e8f0;
}

#app {
	min-height: 100vh;
	min-height: 100dvh;
}

/* Mobile touch improvements */
button, a {
	-webkit-tap-highlight-color: transparent;
}

input, button {
	font-family: inherit;
}

/* Scrollbar styling */
::-webkit-scrollbar {
	width: 6px;
	height: 6px;
}

::-webkit-scrollbar-track {
	background: #1a1d29;
}

::-webkit-scrollbar-thumb {
	background: #3d3a52;
	border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
	background: #6366f1;
}
</style>
