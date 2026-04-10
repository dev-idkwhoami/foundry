<script lang="ts">
	import { Button } from "$lib/components/ui/button";
	import { X, FolderOpen, Trash2, Loader2, Unlink } from "lucide-svelte";
	import type { db } from "$lib/wailsjs/go/models";

	let { onclose }: { onclose: () => void } = $props();

	let projects = $state<db.Installation[]>([]);
	let loading = $state(true);
	let actionInProgress = $state<number | null>(null);
	let error = $state("");

	$effect(() => {
		loadProjects();
	});

	async function loadProjects() {
		try {
			const { ListProjects } = await import("$lib/wailsjs/go/main/App");
			projects = (await ListProjects()) ?? [];
		} catch (e) {
			error = e instanceof Error ? e.message : "Failed to load projects";
		} finally {
			loading = false;
		}
	}

	async function unlinkProject(project: db.Installation) {
		actionInProgress = project.id;
		error = "";
		try {
			const { HerdUnlink } = await import("$lib/wailsjs/go/main/App");
			await HerdUnlink(project.projectPath);
		} catch (e) {
			error = e instanceof Error ? e.message : "Failed to unlink";
		} finally {
			actionInProgress = null;
		}
	}

	async function forgetProject(project: db.Installation) {
		actionInProgress = project.id;
		error = "";
		try {
			const { ForgetProject } = await import("$lib/wailsjs/go/main/App");
			await ForgetProject(project.id);
			projects = projects.filter((p) => p.id !== project.id);
		} catch (e) {
			error = e instanceof Error ? e.message : "Failed to forget project";
		} finally {
			actionInProgress = null;
		}
	}

	async function openInExplorer(path: string) {
		try {
			const { OpenInExplorer } = await import("$lib/wailsjs/go/main/App");
			await OpenInExplorer(path);
		} catch {
			// ignore
		}
	}

	function formatDate(iso: string): string {
		try {
			return new Date(iso).toLocaleDateString(undefined, {
				year: "numeric",
				month: "short",
				day: "numeric",
			});
		} catch {
			return iso;
		}
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) onclose();
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === "Escape") onclose();
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
	onclick={handleBackdropClick}
>
	<div class="flex w-full max-w-xl flex-col border-2 border-border bg-background shadow-[4px_4px_0_0_rgba(0,0,0,0.2)]" style="max-height: 80vh;">
		<div class="flex items-center justify-between border-b-2 border-border px-6 py-4">
			<h2 class="text-lg font-bold tracking-tight">Projects</h2>
			<Button variant="ghost" size="icon" onclick={onclose}>
				<X class="size-4" />
			</Button>
		</div>

		<div class="project-scroll flex-1 overflow-y-auto px-6 py-5 min-h-0">
			{#if loading}
				<div class="flex items-center justify-center gap-2 py-8">
					<Loader2 class="size-5 animate-spin text-muted-foreground" />
					<p class="text-sm text-muted-foreground">Loading projects…</p>
				</div>
			{:else if projects.length === 0}
				<div class="flex flex-col items-center gap-2 py-8 text-center">
					<p class="text-sm text-muted-foreground">No tracked projects yet.</p>
					<p class="text-xs text-muted-foreground">
						Projects appear here after a successful installation.
					</p>
				</div>
			{:else}
				<div class="flex flex-col gap-3">
					{#each projects as project (project.id)}
						{@const busy = actionInProgress === project.id}
						<div class="border-2 border-border bg-card p-4">
							<div class="flex items-start justify-between gap-3">
								<div class="min-w-0 flex-1">
									<h3 class="truncate text-sm font-bold">{project.projectName}</h3>
									<button
										class="mt-0.5 flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
										onclick={() => openInExplorer(project.projectPath)}
										title="Open in Explorer"
									>
										<FolderOpen class="size-3 shrink-0" />
										<span class="truncate">{project.projectPath}</span>
									</button>
									<div class="mt-1.5 flex items-center gap-3 text-[10px] text-muted-foreground">
										<span>{project.siteName}.test</span>
										<span>·</span>
										<span>{formatDate(project.installedAt)}</span>
									</div>
								</div>
							</div>

							<div class="mt-3 flex gap-2">
								<Button
									variant="outline"
									size="sm"
									class="!rounded-none border-2 text-xs"
									disabled={busy}
									onclick={() => unlinkProject(project)}
								>
									{#if busy}
										<Loader2 class="size-3 animate-spin" />
									{:else}
										<Unlink class="size-3" />
									{/if}
									Unlink Herd
								</Button>
								<Button
									variant="outline"
									size="sm"
									class="!rounded-none border-2 text-xs text-destructive hover:bg-destructive hover:text-destructive-foreground"
									disabled={busy}
									onclick={() => forgetProject(project)}
								>
									{#if busy}
										<Loader2 class="size-3 animate-spin" />
									{:else}
										<Trash2 class="size-3" />
									{/if}
									Forget Project
								</Button>
							</div>
						</div>
					{/each}
				</div>
			{/if}

			{#if error}
				<p class="mt-3 text-sm font-medium text-destructive">{error}</p>
			{/if}
		</div>
	</div>
</div>

<style>
	.project-scroll::-webkit-scrollbar {
		width: 6px;
	}
	.project-scroll::-webkit-scrollbar-track {
		background: transparent;
	}
	.project-scroll::-webkit-scrollbar-thumb {
		background: hsl(var(--muted-foreground) / 0.2);
		border-radius: 3px;
	}
	.project-scroll::-webkit-scrollbar-thumb:hover {
		background: hsl(var(--muted-foreground) / 0.35);
	}
</style>
