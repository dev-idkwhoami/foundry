<script lang="ts">
	import { Input } from "$lib/components/ui/input";
	import * as Alert from "$lib/components/ui/alert";
	import { projectName, workingDir, targetPath } from "$lib/stores/project";
	import { prerequisitesMet } from "$lib/stores/install";
	import {
		FolderOpen,
		AlertTriangle,
		CheckCircle,
		Loader2,
		Package,
		GitBranch,
		Server,
		Paintbrush,
		Settings,
	} from "lucide-svelte";

	type EnvState =
		| { status: "loading" }
		| { status: "ok"; version: string }
		| { status: "warn"; version: string; message: string }
		| { status: "missing"; message: string };

	let git = $state<EnvState>({ status: "loading" });
	let herd = $state<EnvState>({ status: "loading" });
	let flux = $state<{ status: "loading" | "active" | "missing" }>({ status: "loading" });
	let targetDirWarning = $state("");

	// Update prerequisitesMet whenever git/herd status changes
	$effect(() => {
		$prerequisitesMet = git.status === "ok" && herd.status === "ok";
	});

	// Check if target directory already exists and is non-empty
	$effect(() => {
		const path = $targetPath;
		if (!path) {
			targetDirWarning = "";
			return;
		}
		checkTargetDir(path);
	});

	async function checkTargetDir(path: string) {
		try {
			const { CheckTargetDirectory } = await import("$lib/wailsjs/go/main/App");
			const result = await CheckTargetDirectory(path);
			targetDirWarning = result === "not-empty"
				? "This directory already exists and is not empty. Installation may overwrite existing files."
				: "";
		} catch {
			targetDirWarning = "";
		}
	}

	export function refreshEnv() {
		checkFlux();
	}

	$effect(() => {
		checkGit();
		checkHerd();
		checkFlux();
	});

	async function checkGit() {
		try {
			const { GetGitVersion } = await import("$lib/wailsjs/go/main/App");
			const version = await GetGitVersion();
			if (!version) {
				git = { status: "missing", message: "Git not found in PATH. Install from git-scm.com" };
			} else {
				git = { status: "ok", version };
			}
		} catch {
			git = { status: "missing", message: "Git not found in PATH. Install from git-scm.com" };
		}
	}

	async function checkHerd() {
		try {
			const { GetHerdVersion } = await import("$lib/wailsjs/go/main/App");
			const version = await GetHerdVersion();
			if (!version) {
				herd = { status: "missing", message: "Laravel Herd not found. Install from herd.laravel.com" };
			} else {
				herd = { status: "ok", version };
			}
		} catch {
			herd = { status: "missing", message: "Laravel Herd not found. Install from herd.laravel.com" };
		}
	}

	async function checkFlux() {
		try {
			const app = await import("$lib/wailsjs/go/main/App");
			const [key, username] = await Promise.all([app.GetFluxLicenseKey(), app.GetFluxUsername()]);
			flux = { status: key && username ? "active" : "missing" };
		} catch {
			flux = { status: "missing" };
		}
	}
</script>

<div class="flex flex-1 flex-col items-center justify-center gap-6">
	<div class="text-center">
		<h2 class="text-2xl font-bold">Project Setup</h2>
		<p class="mt-2 text-sm text-muted-foreground">
			Name your project and verify your environment.
		</p>
	</div>

	<div class="flex w-full max-w-lg flex-col gap-4">
		<!-- Project Name -->
		<div class="border-2 border-border bg-card p-4">
			<label for="project-name" class="mb-2 block text-xs font-medium text-muted-foreground uppercase">
				Project Name
			</label>
			<Input
				id="project-name"
				placeholder="my-laravel-app"
				bind:value={$projectName}
				autofocus
			/>
		</div>

		<!-- Working Directory -->
		<div class="border-2 border-border bg-card p-4">
			<p class="mb-2 text-xs font-medium text-muted-foreground uppercase">
				Working Directory
			</p>
			<div class="flex items-center gap-3 text-sm">
				<FolderOpen class="h-4 w-4 shrink-0 text-muted-foreground" />
				<span class="truncate">{$workingDir || "Not selected"}</span>
			</div>
		</div>

		<!-- Target Path -->
		{#if $targetPath}
			<div class="border-2 border-border bg-card p-4">
				<p class="mb-2 text-xs font-medium text-muted-foreground uppercase">
					Target Path
				</p>
				<code class="block truncate text-sm text-foreground">{$targetPath}</code>
			</div>

			{#if targetDirWarning}
				<div class="flex items-start gap-3 border-2 border-amber-500/50 bg-amber-500/5 p-4">
					<AlertTriangle class="h-4 w-4 shrink-0 text-amber-600 dark:text-amber-400 mt-0.5" />
					<span class="text-sm text-amber-600 dark:text-amber-400">{targetDirWarning}</span>
				</div>
			{/if}
		{/if}

		<!-- Environment Checks -->
		<p class="mt-2 text-xs font-medium text-muted-foreground uppercase">Environment</p>

		<div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
			<!-- Git -->
			<div class="border-2 border-border bg-card p-4">
				<div class="mb-2 flex items-center gap-2">
					<GitBranch class="h-3.5 w-3.5 text-muted-foreground" />
					<span class="text-xs font-medium text-muted-foreground uppercase">Git</span>
				</div>
				{#if git.status === "loading"}
					<div class="flex items-center gap-2 text-xs text-muted-foreground">
						<Loader2 class="h-3.5 w-3.5 animate-spin" />
						Checking...
					</div>
				{:else if git.status === "ok"}
					<div class="flex items-center gap-2 text-sm font-medium text-green-600 dark:text-green-400">
						<CheckCircle class="h-3.5 w-3.5" />
						{git.version}
					</div>
				{:else}
					<div class="flex items-center gap-2 text-sm font-medium text-destructive">
						<AlertTriangle class="h-3.5 w-3.5" />
						Missing
					</div>
					<span class="mt-1 block text-xs text-muted-foreground">{git.message}</span>
				{/if}
			</div>

			<!-- Herd -->
			<div class="border-2 border-border bg-card p-4">
				<div class="mb-2 flex items-center gap-2">
					<Server class="h-3.5 w-3.5 text-muted-foreground" />
					<span class="text-xs font-medium text-muted-foreground uppercase">Herd</span>
				</div>
				{#if herd.status === "loading"}
					<div class="flex items-center gap-2 text-xs text-muted-foreground">
						<Loader2 class="h-3.5 w-3.5 animate-spin" />
						Checking...
					</div>
				{:else if herd.status === "ok"}
					<div class="flex items-center gap-2 text-sm font-medium text-green-600 dark:text-green-400">
						<CheckCircle class="h-3.5 w-3.5" />
						{herd.version}
					</div>
				{:else}
					<div class="flex items-center gap-2 text-sm font-medium text-destructive">
						<AlertTriangle class="h-3.5 w-3.5" />
						Missing
					</div>
					<span class="mt-1 block text-xs text-muted-foreground">{herd.message}</span>
				{/if}
			</div>

			<!-- Flux UI Pro -->
			<div class="border-2 border-border bg-card p-4">
				<div class="mb-2 flex items-center gap-2">
					<Paintbrush class="h-3.5 w-3.5 text-muted-foreground" />
					<span class="text-xs font-medium text-muted-foreground uppercase">Flux Pro</span>
				</div>
				{#if flux.status === "loading"}
					<div class="flex items-center gap-2 text-xs text-muted-foreground">
						<Loader2 class="h-3.5 w-3.5 animate-spin" />
						Checking...
					</div>
				{:else if flux.status === "active"}
					<div class="flex items-center gap-2 text-sm font-medium text-green-600 dark:text-green-400">
						<CheckCircle class="h-3.5 w-3.5" />
						Licensed
					</div>
				{:else}
					<div class="flex items-center gap-2 text-sm font-medium text-amber-600 dark:text-amber-400">
						<Settings class="h-3.5 w-3.5" />
						Not configured
					</div>
					<span class="mt-1 block text-xs text-muted-foreground">Add key in Settings</span>
				{/if}
			</div>
		</div>
	</div>
</div>
