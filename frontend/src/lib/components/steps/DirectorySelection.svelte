<script lang="ts">
	import { Button } from "$lib/components/ui/button";
	import * as Card from "$lib/components/ui/card";
	import { workingDir } from "$lib/stores/project";
	import { goToStep } from "$lib/stores/wizard";
	import { FolderOpen, FolderSearch, Clock } from "lucide-svelte";

	let recentDirs = $state<string[]>([]);

	// Will be replaced with Wails bindings in Phase 2
	async function loadRecent() {
		try {
			const { GetRecentDirectories } = await import("$lib/wailsjs/go/main/App");
			recentDirs = (await GetRecentDirectories()) ?? [];
		} catch {
			recentDirs = [];
		}
	}

	async function browse() {
		try {
			const { SelectDirectory } = await import("$lib/wailsjs/go/main/App");
			const dir = await SelectDirectory();
			if (dir) {
				$workingDir = dir;
				goToStep("project-setup");
			}
		} catch {
			// Fallback for dev without Wails
			$workingDir = "C:/Projects";
			goToStep("project-setup");
		}
	}

	function selectDir(dir: string) {
		$workingDir = dir;
		goToStep("project-setup");
	}

	$effect(() => {
		loadRecent();
	});
</script>

<div class="flex flex-1 flex-col items-center justify-center gap-6">
	<div class="text-center">
		<h2 class="text-2xl font-bold">Where do you want to create your project?</h2>
		<p class="mt-2 text-sm text-muted-foreground">
			Select a working directory for your new Laravel project.
		</p>
	</div>

	<div class="flex w-full max-w-md flex-col gap-3">
		{#if recentDirs.length > 0}
			<p class="flex items-center gap-2 text-xs font-medium text-muted-foreground uppercase">
				<Clock class="h-3 w-3" />
				Recent
			</p>
			{#each recentDirs as dir}
				<button
					class="flex items-center gap-3 border-2 border-border bg-card px-4 py-3 text-left text-sm transition-colors hover:bg-muted"
					onclick={() => selectDir(dir)}
				>
					<FolderOpen class="h-4 w-4 shrink-0 text-muted-foreground" />
					<span class="truncate">{dir}</span>
				</button>
			{/each}
			<div class="my-1"></div>
		{/if}

		<Button variant="outline" class="w-full justify-start gap-3" onclick={browse}>
			<FolderSearch class="h-4 w-4" />
			Browse for directory...
		</Button>
	</div>
</div>
