<script lang="ts">
	import { onMount, onDestroy } from "svelte";
	import { get } from "svelte/store";
	import { Loader2, AlertTriangle } from "lucide-svelte";
	import { registry } from "$lib/stores/features";
	import { goToStep, resolveFirstStep } from "$lib/stores/wizard";
	import { hasContext, projectName, workingDir } from "$lib/stores/project";

	type LoadState =
		| { status: "loading"; message: string }
		| { status: "error"; message: string };

	let loadState = $state<LoadState>({ status: "loading", message: "Cloning repository..." });

	let cleanups: (() => void)[] = [];

	async function handleReady(payload: string) {
		if (payload) {
			loadState = { status: "error", message: payload };
			return;
		}

		loadState = { status: "loading", message: "Loading feature registry..." };

		try {
			const { GetFeatureRegistry } = await import("$lib/wailsjs/go/main/App");
			const reg = await GetFeatureRegistry();
			registry.set(reg);
		} catch {
			// FeatureSelection has its own retry — proceed anyway
		}

		const ctx = {
			hasContext: get(hasContext),
			projectName: get(projectName),
		};
		goToStep(resolveFirstStep(ctx));
	}

	function handleCloneError(payload: string) {
		loadState = { status: "error", message: `Clone failed: ${payload}` };
	}

	function handleRegistryError(payload: string) {
		loadState = { status: "error", message: `Registry build failed: ${payload}` };
	}

	onMount(async () => {
		try {
			const [{ EventsOn }, app] = await Promise.all([
				import("$lib/wailsjs/runtime/runtime"),
				import("$lib/wailsjs/go/main/App"),
			]);

			// Check if startup already completed before we mounted (race condition).
			const result = await app.GetStartupResult();
			if (result.done) {
				if (result.error) {
					loadState = { status: "error", message: result.error };
				} else {
					await handleReady("");
				}
				return;
			}

			// Not done yet — listen for events.
			cleanups.push(EventsOn("ready", handleReady));
			cleanups.push(EventsOn("clone:error", handleCloneError));
			cleanups.push(EventsOn("registry:error", handleRegistryError));
		} catch {
			// Dev mode without Wails runtime — do nothing, +page.svelte handles fallback
		}
	});

	onDestroy(() => {
		for (const cleanup of cleanups) {
			cleanup();
		}
	});
</script>

{#if loadState.status === "loading"}
	<div class="flex flex-1 flex-col items-center justify-center gap-4">
		<Loader2 class="h-8 w-8 animate-spin text-primary" />
		<p class="text-sm text-muted-foreground">{loadState.message}</p>
	</div>
{:else}
	<div class="flex flex-1 flex-col items-center justify-center gap-4">
		<div class="flex w-full max-w-md flex-col items-center gap-4 border-2 border-border bg-card p-8">
			<AlertTriangle class="h-8 w-8 text-destructive" />
			<h2 class="text-lg font-bold">Startup Error</h2>
			<p class="text-center text-sm text-muted-foreground">{loadState.message}</p>
			<p class="text-center text-xs text-muted-foreground">
				Check your repository settings using the <strong>Settings</strong> icon in the title bar.
			</p>
		</div>
	</div>
{/if}
