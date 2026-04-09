<script lang="ts">
	import { Button } from "$lib/components/ui/button";
	import { Badge } from "$lib/components/ui/badge";
	import { Separator } from "$lib/components/ui/separator";
	import * as Card from "$lib/components/ui/card";

	import { Check, FileText, Wrench, Settings, Rocket, FolderOpen } from "lucide-svelte";

	import { selectedFeatures, configValues, featureList } from "$lib/stores/features";
	import { projectName, workingDir, targetPath } from "$lib/stores/project";
	import { goToStep } from "$lib/stores/wizard";

	const allPatches = $derived(
		$selectedFeatures.flatMap((f) =>
			(f.patches ?? []).map((p) => ({ feature: f.name, file: p.file }))
		)
	);

	const manualStepCount = $derived(
		$selectedFeatures.reduce(
			(sum, f) => sum + (f.patches ?? []).filter((p) => p.mode === "manual").length,
			0
		)
	);

	const configEntries = $derived(
		$selectedFeatures
			.filter((f) => f.config && f.config.length > 0)
			.map((f) => ({
				feature: f.name,
				fields: f.config.map((c) => ({
					label: c.label,
					value: $configValues[`${f.id}.${c.key}`] || c.default || "—",
				})),
			}))
	);
</script>

<div class="mx-auto flex w-full max-w-2xl flex-col gap-6 p-6">
	<div class="flex items-center gap-3">
		<Rocket class="size-5 text-foreground" />
		<h2 class="text-lg font-bold">Review & Install</h2>
	</div>

	<p class="text-sm text-muted-foreground">
		Review your selections before installing. This will scaffold your Laravel project and apply all selected features.
	</p>

	<!-- Project Info -->
	<Card.Root class="border-2 rounded-none">
		<Card.Header>
			<Card.Title class="flex items-center gap-2 text-sm font-bold">
				<FolderOpen class="size-4" />
				Project Info
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
				<dt class="font-medium text-muted-foreground">Name</dt>
				<dd class="font-mono font-bold">{$projectName}</dd>

				<dt class="font-medium text-muted-foreground">Directory</dt>
				<dd class="font-mono text-xs break-all">{$targetPath}</dd>
			</dl>
		</Card.Content>
	</Card.Root>

	<!-- Selected Features -->
	<Card.Root class="border-2 rounded-none">
		<Card.Header>
			<Card.Title class="flex items-center gap-2 text-sm font-bold">
				<Check class="size-4" />
				Selected Features
				<Badge variant="secondary" class="ml-auto rounded-none font-mono text-xs">
					{$selectedFeatures.length}
				</Badge>
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<ul class="flex flex-col gap-1.5">
				{#each $selectedFeatures as feature (feature.id)}
					<li class="flex items-center gap-2 text-sm">
						<Check class="size-3.5 text-green-600 dark:text-green-400 shrink-0" />
						<span class="font-medium">{feature.name}</span>
						{#if feature.description}
							<span class="text-xs text-muted-foreground truncate">
								— {feature.description}
							</span>
						{/if}
					</li>
				{/each}
			</ul>
		</Card.Content>
	</Card.Root>

	<!-- Patches -->
	{#if allPatches.length > 0}
		<Card.Root class="border-2 rounded-none">
			<Card.Header>
				<Card.Title class="flex items-center gap-2 text-sm font-bold">
					<FileText class="size-4" />
					Auto Patches
					<Badge variant="secondary" class="ml-auto rounded-none font-mono text-xs">
						{allPatches.length}
					</Badge>
				</Card.Title>
				<Card.Description class="text-xs">
					These file patches will be applied automatically during installation.
				</Card.Description>
			</Card.Header>
			<Card.Content>
				<ul class="flex flex-col gap-1">
					{#each allPatches as patch, i (i)}
						<li class="flex items-center justify-between gap-2 text-sm">
							<span class="font-mono text-xs truncate">{patch.file}</span>
							<Badge variant="outline" class="rounded-none text-xs shrink-0">
								{patch.feature}
							</Badge>
						</li>
					{/each}
				</ul>
			</Card.Content>
		</Card.Root>
	{/if}

	<!-- Configuration -->
	{#if configEntries.length > 0}
		<Card.Root class="border-2 rounded-none">
			<Card.Header>
				<Card.Title class="flex items-center gap-2 text-sm font-bold">
					<Settings class="size-4" />
					Configuration
				</Card.Title>
			</Card.Header>
			<Card.Content class="flex flex-col gap-4">
				{#each configEntries as entry, i (i)}
					{#if i > 0}
						<Separator />
					{/if}
					<div>
						<p class="mb-2 text-xs font-bold uppercase tracking-wide text-muted-foreground">
							{entry.feature}
						</p>
						<dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
							{#each entry.fields as field}
								<dt class="font-medium text-muted-foreground">{field.label}</dt>
								<dd class="font-mono">{field.value}</dd>
							{/each}
						</dl>
					</div>
				{/each}
			</Card.Content>
		</Card.Root>
	{/if}

	<!-- Manual Steps Notice -->
	{#if manualStepCount > 0}
		<Card.Root class="border-2 border-amber-500/50 rounded-none bg-amber-500/5">
			<Card.Content class="flex items-center gap-3 pt-6">
				<Wrench class="size-5 text-amber-600 dark:text-amber-400 shrink-0" />
				<div>
					<p class="text-sm font-bold">
						{manualStepCount} manual step{manualStepCount > 1 ? "s" : ""} required after install
					</p>
					<p class="text-xs text-muted-foreground">
						You'll see these instructions after installation completes.
					</p>
				</div>
			</Card.Content>
		</Card.Root>
	{/if}

	<!-- Install CTA -->
	<div class="mt-2">
		<Button
			size="lg"
			class="w-full border-2 rounded-none py-6 text-base font-bold"
			onclick={() => goToStep("install")}
		>
			<Rocket class="mr-2 size-5" />
			Install Project
		</Button>
	</div>
</div>
