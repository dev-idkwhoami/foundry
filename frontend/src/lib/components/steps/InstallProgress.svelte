<script lang="ts">
	import { onMount, onDestroy } from "svelte";
	import { get } from "svelte/store";

	import { EventsOn, EventsOff } from "$lib/wailsjs/runtime/runtime";
	import { Install, Quit, OpenInExplorer } from "$lib/wailsjs/go/main/App";

	import { projectName, workingDir, targetPath } from "$lib/stores/project";
	import { selectedIds, configValues } from "$lib/stores/features";
	import { goToStep } from "$lib/stores/wizard";
	import { manualSteps as manualStepsStore } from "$lib/stores/install";
	import type { ManualPatch } from "$lib/types/installer";

	import { Button } from "$lib/components/ui/button";
	import * as Card from "$lib/components/ui/card";
	import { Badge } from "$lib/components/ui/badge";

	import {
		Loader2,
		Check,
		X,
		Circle,
		Terminal,
		Rocket,
		AlertTriangle,
		ArrowRight,
		FolderOpen,
	} from "lucide-svelte";

	// --- Stage definitions ---

	interface Stage {
		id: string;
		label: string;
		status: "pending" | "running" | "done" | "error";
	}

	const STAGE_DEFS: { id: string; label: string }[] = [
		{ id: "pre-clone", label: "Pre-clone hooks" },
		{ id: "clone", label: "Cloning repository" },
		{ id: "post-clone", label: "Post-clone hooks" },
		{ id: "pre-herd", label: "Pre-herd hooks" },
		{ id: "herd", label: "Setting up Herd" },
		{ id: "post-herd", label: "Post-herd hooks" },
		{ id: "patching", label: "Applying patches" },
		{ id: "pre-install", label: "Pre-install hooks" },
		{ id: "post-install", label: "Post-install commands" },
		{ id: "hooks:post-install", label: "Post-install hooks" },
		{ id: "pre-cleanup", label: "Pre-cleanup hooks" },
		{ id: "cleanup", label: "Cleaning up" },
		{ id: "post-cleanup", label: "Post-cleanup hooks" },
	];

	// --- Reactive state ---

	let stages: Stage[] = $state(
		STAGE_DEFS.map((s) => ({ ...s, status: "pending" as const }))
	);

	let logs: { message: string; level: string }[] = $state([]);
	let errorMessage: string = $state("");
	let errorStage: string = $state("");
	let manualSteps: ManualPatch[] = $state([]);
	let finished: boolean = $state(false);
	let failed: boolean = $state(false);

	let logContainer: HTMLDivElement | undefined = $state(undefined);

	// --- Auto-scroll log panel ---

	$effect(() => {
		// Access length to create dependency
		const _len = logs.length;
		if (logContainer) {
			logContainer.scrollTop = logContainer.scrollHeight;
		}
	});

	// --- Derived ---

	const currentStageLabel = $derived(
		stages.find((s) => s.status === "running")?.label ?? ""
	);

	const hasManualSteps = $derived(manualSteps.length > 0);

	// --- Helpers ---

	function updateStageStatus(
		stageId: string,
		status: "pending" | "running" | "done" | "error"
	) {
		stages = stages.map((s) =>
			s.id === stageId ? { ...s, status } : s
		);
	}

	// --- Lifecycle ---

	onMount(() => {
		// Set up event listeners
		EventsOn("install:log", (data: { message: string; level: string }) => {
			logs = [...logs, data];
		});

		EventsOn(
			"install:progress",
			(data: { stage: string; status: "running" | "done" | "error" }) => {
				updateStageStatus(data.stage, data.status);
			}
		);

		EventsOn("install:error", (data: { stage: string; message: string }) => {
			errorMessage = data.message;
			errorStage = data.stage;
			failed = true;
			updateStageStatus(data.stage, "error");
		});

		EventsOn("install:complete", () => {
			finished = true;
		});

		EventsOn(
			"install:result",
			(data: {
				success: boolean;
				manualSteps: ManualPatch[];
				errorMessage: string;
				errorStage: string;
			}) => {
				if (data.success) {
					finished = true;
					manualSteps = data.manualSteps ?? [];
					manualStepsStore.set(manualSteps);
				} else {
					failed = true;
					errorMessage = data.errorMessage;
					errorStage = data.errorStage;
				}
			}
		);

		// Kick off the install
		Install({
			projectName: get(projectName),
			workingDir: get(workingDir),
			selectedIds: Array.from(get(selectedIds)),
			configValues: get(configValues),
			tempClonePath: "",
		});
	});

	onDestroy(() => {
		EventsOff("install:log");
		EventsOff("install:progress");
		EventsOff("install:error");
		EventsOff("install:complete");
		EventsOff("install:result");
	});
</script>

<div class="flex w-full flex-1 flex-col gap-6 p-6 min-h-0">
	<!-- Header -->
	<div class="flex items-center gap-3">
		{#if finished}
			<Rocket class="size-5 text-green-600 dark:text-green-400" />
			<h2 class="text-lg font-bold">Installation Complete</h2>
		{:else if failed}
			<AlertTriangle class="size-5 text-red-600 dark:text-red-400" />
			<h2 class="text-lg font-bold">Installation Failed</h2>
		{:else}
			<Terminal class="size-5 text-foreground" />
			<h2 class="text-lg font-bold">Installing Project</h2>
		{/if}
	</div>

	{#if !finished && !failed}
		<p class="text-sm text-muted-foreground">
			{currentStageLabel ? currentStageLabel + "..." : "Preparing installation..."}
		</p>
	{/if}

	<div class="grid grid-cols-[220px_1fr] gap-6 flex-1 min-h-0">
		<!-- Stage sidebar -->
		<Card.Root class="border-2 border-border rounded-none">
			<Card.Header class="pb-3">
				<Card.Title class="text-xs font-bold uppercase tracking-wide text-muted-foreground">
					Stages
				</Card.Title>
			</Card.Header>
			<Card.Content class="flex flex-col gap-2">
				{#each stages as stage (stage.id)}
					<div class="flex items-center gap-2.5">
						{#if stage.status === "pending"}
							<Circle class="size-4 text-muted-foreground/40 shrink-0" />
						{:else if stage.status === "running"}
							<Loader2 class="size-4 text-foreground animate-spin shrink-0" />
						{:else if stage.status === "done"}
							<Check class="size-4 text-green-600 dark:text-green-400 shrink-0" />
						{:else if stage.status === "error"}
							<X class="size-4 text-red-600 dark:text-red-400 shrink-0" />
						{/if}
						<span
							class="text-sm"
							class:font-bold={stage.status === "running"}
							class:text-muted-foreground={stage.status === "pending"}
							class:text-red-600={stage.status === "error"}
							class:dark:text-red-400={stage.status === "error"}
						>
							{stage.label}
						</span>
					</div>
				{/each}
			</Card.Content>
		</Card.Root>

		<!-- Log panel -->
		<Card.Root class="border-2 border-border rounded-none flex flex-col min-h-0">
			<Card.Header class="pb-3 shrink-0">
				<Card.Title class="flex items-center gap-2 text-xs font-bold uppercase tracking-wide text-muted-foreground">
					<Terminal class="size-3.5" />
					Output
				</Card.Title>
			</Card.Header>
			<Card.Content class="flex-1 min-h-0 flex flex-col">
				<div
					bind:this={logContainer}
					class="log-scroll flex-1 min-h-0 overflow-auto bg-zinc-950 text-zinc-300 font-mono text-xs p-3 border border-zinc-800 rounded-sm"
				>
					{#each logs as log, i (i)}
						<div class="leading-5">
							{#if log.level === "error"}
								<span class="text-red-400">{log.message}</span>
							{:else if log.level === "warn"}
								<span class="text-amber-400">{log.message}</span>
							{:else if log.level === "success"}
								<span class="text-green-400">{log.message}</span>
							{:else}
								<span>{log.message}</span>
							{/if}
						</div>
					{/each}
					{#if logs.length === 0}
						<span class="text-zinc-500 italic">Waiting for output...</span>
					{/if}
				</div>
			</Card.Content>
		</Card.Root>
	</div>

	<!-- Error state -->
	{#if failed}
		<Card.Root class="border-2 border-red-500/50 rounded-none bg-red-500/5">
			<Card.Header class="pb-2">
				<Card.Title class="flex items-center gap-2 text-red-600 dark:text-red-400">
					<AlertTriangle class="size-5 shrink-0" />
					Critical Error
				</Card.Title>
			</Card.Header>
			<Card.Content class="flex flex-col gap-3">
				<p class="text-sm font-bold">
					Failed at: {stages.find((s) => s.id === errorStage)?.label ?? errorStage}
				</p>
				<div class="max-h-32 overflow-auto bg-zinc-950 border border-zinc-800 rounded-sm p-3">
					<pre class="text-xs text-red-400 font-mono whitespace-pre-wrap">{errorMessage}</pre>
				</div>
				{#if errorStage === "patching"}
					<p class="text-xs text-muted-foreground">
						A patch failed to apply. This usually means a feature's patch conflicts with another feature or the base repository has changed. Check the output log above for details on which patch file failed.
					</p>
				{:else if errorStage === "clone"}
					<p class="text-xs text-muted-foreground">
						Could not clone the repository. Check your network connection and verify the repository URL in Settings.
					</p>
				{:else if errorStage === "herd"}
					<p class="text-xs text-muted-foreground">
						Herd site setup failed. Ensure Laravel Herd is running and the site name is not already in use.
					</p>
				{/if}
			</Card.Content>
		</Card.Root>
		<Button
			size="lg"
			variant="outline"
			class="w-full border-2 rounded-none py-6 text-base font-bold"
			onclick={() => goToStep("review")}
		>
			<X class="mr-2 size-5" />
			Close
		</Button>
	{/if}

	<!-- Success state -->
	{#if finished}
		<Card.Root class="border-2 border-green-500/50 rounded-none bg-green-500/5">
			<Card.Content class="flex items-center gap-3 pt-6">
				<Check class="size-5 text-green-600 dark:text-green-400 shrink-0" />
				<div>
					<p class="text-sm font-bold text-green-600 dark:text-green-400">
						Project installed successfully!
					</p>
					{#if hasManualSteps}
						<p class="text-xs text-muted-foreground">
							There are manual steps required to complete the setup.
						</p>
					{/if}
				</div>
			</Card.Content>
		</Card.Root>

		{#if hasManualSteps}
			<Button
				size="lg"
				class="w-full border-2 rounded-none py-6 text-base font-bold"
				onclick={() => goToStep("manual")}
			>
				<ArrowRight class="mr-2 size-5" />
				Continue to Manual Steps
			</Button>
		{:else}
			<div class="flex gap-3">
				<Button
					size="lg"
					variant="outline"
					class="flex-1 border-2 rounded-none py-6 text-base font-bold"
					onclick={() => {
						OpenInExplorer(get(targetPath));
						Quit();
					}}
				>
					<FolderOpen class="mr-2 size-5" />
					Open in Explorer
				</Button>
				<Button
					size="lg"
					variant="outline"
					class="flex-1 border-2 rounded-none py-6 text-base font-bold"
					onclick={() => Quit()}
				>
					<X class="mr-2 size-5" />
					Close
				</Button>
			</div>
		{/if}
	{/if}
</div>

<style>
	:global(.log-scroll::-webkit-scrollbar) {
		width: 6px;
		height: 6px;
	}
	:global(.log-scroll::-webkit-scrollbar-track) {
		background: transparent;
	}
	:global(.log-scroll::-webkit-scrollbar-thumb) {
		background: rgba(255, 255, 255, 0.15);
		border-radius: 3px;
	}
	:global(.log-scroll::-webkit-scrollbar-thumb:hover) {
		background: rgba(255, 255, 255, 0.25);
	}
	:global(.log-scroll::-webkit-scrollbar-corner) {
		background: transparent;
	}
</style>
