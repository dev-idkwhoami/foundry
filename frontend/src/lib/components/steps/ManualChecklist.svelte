<script lang="ts">
	import { get } from "svelte/store";

	import { manualSteps } from "$lib/stores/install";
	import { targetPath } from "$lib/stores/project";
	import { OpenInExplorer, Quit } from "$lib/wailsjs/go/main/App";

	import { Button } from "$lib/components/ui/button";
	import * as Card from "$lib/components/ui/card";
	import { Badge } from "$lib/components/ui/badge";
	import { Checkbox } from "$lib/components/ui/checkbox";

	import {
		Check,
		ClipboardList,
		ClipboardCopy,
		FolderOpen,
		X,
	} from "lucide-svelte";

	// --- Reactive state ---

	// Track completion per step using a keyed map so we don't need an $effect to sync array length
	let checkedMap: Record<string, boolean> = $state({});

	function stepKey(step: { featureId: string; file: string; instruction: string }): string {
		return step.featureId + "::" + (step.file || step.instruction);
	}

	function isChecked(step: { featureId: string; file: string; instruction: string }): boolean {
		return checkedMap[stepKey(step)] ?? false;
	}

	function toggleChecked(step: { featureId: string; file: string; instruction: string }, value: boolean) {
		checkedMap[stepKey(step)] = value;
	}

	// --- Derived ---

	const allChecked = $derived(
		$manualSteps.length > 0 &&
		$manualSteps.every((step) => checkedMap[stepKey(step)] === true)
	);

	const checkedCount = $derived(
		$manualSteps.filter((step) => checkedMap[stepKey(step)] === true).length
	);

	// --- Handlers ---

	function handleOpenProject() {
		OpenInExplorer(get(targetPath));
	}

	function handleDone() {
		OpenInExplorer(get(targetPath));
		Quit();
	}

	let copiedKey = $state<string | null>(null);

	async function handleCopy(text: string, key: string) {
		await navigator.clipboard.writeText(text);
		copiedKey = key;
		setTimeout(() => {
			if (copiedKey === key) copiedKey = null;
		}, 2000);
	}

	function handleClose() {
		Quit();
	}
</script>

<div class="mx-auto flex w-full max-w-3xl flex-1 flex-col gap-6 p-6 min-h-0">
	<!-- Header -->
	<div class="flex items-center gap-3">
		<ClipboardList class="size-5 text-foreground" />
		<div>
			<h2 class="text-lg font-bold">Manual Steps</h2>
			{#if $manualSteps.length > 0}
				<p class="text-sm text-muted-foreground">
					{checkedCount} of {$manualSteps.length} step{$manualSteps.length === 1 ? "" : "s"} completed
				</p>
			{/if}
		</div>
	</div>

	{#if $manualSteps.length === 0}
		<!-- Empty state -->
		<div class="flex flex-1 flex-col items-center justify-center gap-4">
			<div class="flex size-16 items-center justify-center rounded-full bg-green-500/10">
				<Check class="size-8 text-green-600 dark:text-green-400" />
			</div>
			<div class="text-center">
				<p class="text-lg font-bold">No manual steps required</p>
				<p class="text-sm text-muted-foreground">
					Everything was configured automatically. You're all set!
				</p>
			</div>
		</div>

		<div class="flex gap-3">
			<Button
				size="lg"
				variant="outline"
				class="flex-1 border-2 rounded-none py-6 text-base font-bold"
				onclick={handleDone}
			>
				<FolderOpen class="mr-2 size-5" />
				Open in Explorer
			</Button>
			<Button
				size="lg"
				variant="outline"
				class="flex-1 border-2 rounded-none py-6 text-base font-bold"
				onclick={handleClose}
			>
				<X class="mr-2 size-5" />
				Close
			</Button>
		</div>
	{:else}
		<!-- Open project button -->
		<Button
			variant="outline"
			class="w-fit border-2 rounded-none font-bold"
			onclick={handleOpenProject}
		>
			<FolderOpen class="mr-2 size-4" />
			Open Project in Explorer
		</Button>

		<!-- Steps list -->
		<div class="flex flex-1 flex-col gap-4 overflow-auto min-h-0">
			{#each $manualSteps as step, i (step.featureId + "-" + step.file)}
				<Card.Root
					class="border-2 border-border rounded-none {isChecked(step)
						? 'opacity-60'
						: ''}"
				>
					<Card.Header class="pb-3">
						<div class="flex items-start gap-3">
							<Checkbox
								checked={isChecked(step)}
								onCheckedChange={(v) => toggleChecked(step, v === true)}
								class="mt-0.5"
							/>
							<div class="flex flex-1 flex-col gap-2">
								<div class="flex items-center gap-2">
									<Badge variant="outline" class="rounded-none border-2 text-xs font-bold">
										{step.featureName}
									</Badge>
								</div>
								<p
									class="text-sm leading-relaxed {isChecked(step)
										? 'line-through text-muted-foreground'
										: 'text-foreground'}"
								>
									{step.instruction}
								</p>
								{#if step.copy}
									{@const key = stepKey(step)}
									<button
										class="mt-1.5 flex items-center gap-2 border-2 border-border bg-zinc-950 px-3 py-1.5 text-left font-mono text-xs text-zinc-300 transition-colors hover:border-foreground/50"
										onclick={() => handleCopy(step.copy, key)}
										title="Copy to clipboard"
									>
										<code class="flex-1 select-all">{step.copy}</code>
										{#if copiedKey === key}
											<Check class="size-3 shrink-0 text-green-400" />
										{:else}
											<ClipboardCopy class="size-3 shrink-0 text-muted-foreground" />
										{/if}
									</button>
								{/if}
							</div>
						</div>
					</Card.Header>
				</Card.Root>
			{/each}
		</div>

		<!-- Completion bar -->
		{#if allChecked}
			<Card.Root class="border-2 border-green-500/50 rounded-none bg-green-500/5">
				<Card.Content class="flex items-center gap-3 pt-6">
					<Check class="size-5 text-green-600 dark:text-green-400 shrink-0" />
					<p class="text-sm font-bold text-green-600 dark:text-green-400">
						All manual steps completed!
					</p>
				</Card.Content>
			</Card.Root>
		{/if}

		<!-- Action buttons -->
		<div class="flex gap-3">
			<Button
				size="lg"
				class="flex-1 border-2 rounded-none py-6 text-base font-bold"
				disabled={!allChecked}
				onclick={handleDone}
			>
				<Check class="mr-2 size-5" />
				Done
			</Button>
			<Button
				size="lg"
				variant="outline"
				class="flex-1 border-2 rounded-none py-6 text-base font-bold"
				onclick={handleClose}
			>
				<X class="mr-2 size-5" />
				Close
			</Button>
		</div>
	{/if}
</div>
