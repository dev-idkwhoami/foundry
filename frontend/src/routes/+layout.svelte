<script lang="ts">
	import "../app.css";
	import { onMount } from "svelte";

	let { children } = $props();

	onMount(() => {
		const mq = window.matchMedia("(prefers-color-scheme: dark)");

		function apply(dark: boolean) {
			document.documentElement.classList.toggle("dark", dark);
		}

		apply(mq.matches);
		mq.addEventListener("change", (e) => apply(e.matches));
	});
</script>

{@render children()}
