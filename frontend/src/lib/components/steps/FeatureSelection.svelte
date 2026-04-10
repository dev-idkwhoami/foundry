<script lang="ts">
	import { Checkbox } from "$lib/components/ui/checkbox";
	import { Badge } from "$lib/components/ui/badge";
	import * as Tooltip from "$lib/components/ui/tooltip";
	import { Loader2, Lock, AlertTriangle, Wrench, Search } from "lucide-svelte";
	import Input from "$lib/components/ui/input/input.svelte";
	import {
		registry,
		featureList,
		selectedIds,
		disabledIds,
		lockedIds,
		incompatibleIds,
		dynamicDisabled,
		dynamicDisabledReasons,
		checking,
		toggleFeature,
		deselectFeature,
		addDynamicDisable,
	} from "$lib/stores/features";
	import { get } from "svelte/store";

	type LoadState =
		| { status: "loading" }
		| { status: "ready" }
		| { status: "error"; message: string };

	let loadState = $state<LoadState>({ status: "loading" });
	let searchQuery = $state("");

	const filteredFeatures = $derived.by(() => {
		const q = searchQuery.toLowerCase().trim();
		if (!q) return $featureList;
		return $featureList.filter(
			(f) => f.name.toLowerCase().includes(q) || f.description?.toLowerCase().includes(q)
		);
	});

	$effect(() => {
		loadRegistry();
	});

	async function loadRegistry() {
		if (get(registry)) {
			loadState = { status: "ready" };
			return;
		}
		try {
			const { GetFeatureRegistry } = await import("$lib/wailsjs/go/main/App");
			const reg = await GetFeatureRegistry();
			if (!reg || !reg.Features || reg.Features.length === 0) {
				// Registry not built yet — retry after a short delay
				setTimeout(loadRegistry, 500);
				return;
			}
			registry.set(reg);
			loadState = { status: "ready" };
		} catch (e) {
			loadState = {
				status: "error",
				message: e instanceof Error ? e.message : "Failed to load feature registry",
			};
		}
	}

	async function handleToggle(featureId: string) {
		const wasSelected = get(selectedIds).has(featureId);
		const autoSelected = toggleFeature(featureId);

		if (wasSelected) return;

		checking.update((s) => {
			const next = new Set(s);
			next.add(featureId);
			return next;
		});

		try {
			const { CheckPatchCompatibility } = await import("$lib/wailsjs/go/main/App");
			const currentSelected = [...get(selectedIds)];
			const result = await CheckPatchCompatibility(featureId, currentSelected);

			if (!result.compatible) {
				deselectFeature(featureId);
				addDynamicDisable(featureId, result.reason);
			}
		} catch (e) {
			deselectFeature(featureId);
			addDynamicDisable(
				featureId,
				e instanceof Error ? e.message : "Compatibility check failed"
			);
		} finally {
			checking.update((s) => {
				const next = new Set(s);
				next.delete(featureId);
				return next;
			});
		}
	}

	function isSelected(id: string, selected: Set<string>): boolean {
		return selected.has(id);
	}

	function isDisabled(id: string, disabled: Set<string>, locked: Set<string>): boolean {
		return disabled.has(id) || locked.has(id);
	}

	function getDisableReason(
		id: string,
		incompat: Set<string>,
		dynDisabled: Set<string>,
		dynReasons: Record<string, string>
	): string | null {
		if (incompat.has(id)) return "Incompatible with a selected feature";
		if (dynDisabled.has(id)) return dynReasons[id] ?? "Incompatible";
		return null;
	}
</script>

{#if loadState.status === "loading"}
	<div class="flex flex-1 items-center justify-center gap-2">
		<Loader2 class="text-muted-foreground size-5 animate-spin" />
		<p class="text-muted-foreground">Loading features...</p>
	</div>
{:else if loadState.status === "error"}
	<div class="flex flex-1 items-center justify-center">
		<div class="border-destructive bg-destructive/10 flex items-center gap-3 border-2 p-4">
			<AlertTriangle class="text-destructive size-5 shrink-0" />
			<div>
				<p class="font-bold">Failed to load features</p>
				<p class="text-muted-foreground text-sm">{loadState.message}</p>
			</div>
		</div>
	</div>
{:else}
	<div class="mx-auto flex w-full max-w-4xl flex-1 min-h-0 flex-col gap-6 p-6">
		<div>
			<h2 class="text-2xl font-black">Select Features</h2>
			<p class="text-muted-foreground mt-1">
				Choose the features to install. Dependencies are auto-selected.
			</p>
		</div>

		<div class="relative">
			<Search class="text-muted-foreground absolute left-3 top-1/2 size-4 -translate-y-1/2" />
			<Input
				type="text"
				placeholder="Search features..."
				bind:value={searchQuery}
				class="pl-9"
			/>
		</div>

		{#if $featureList.length === 0}
			<div class="text-muted-foreground border-border flex items-center justify-center border-2 p-8">
				<p>No features available in the registry.</p>
			</div>
		{:else}
			<div class="feature-scroll -mr-3 flex-1 overflow-y-auto pr-3 min-h-0">
				{#if filteredFeatures.length === 0}
					<div class="text-muted-foreground border-border flex items-center justify-center border-2 p-8">
						<p>No features match "{searchQuery}"</p>
					</div>
				{:else}
					<div class="grid grid-cols-1 gap-3 md:grid-cols-2">
						{#each filteredFeatures as feature (feature.id)}
							{@const selected = isSelected(feature.id, $selectedIds)}
							{@const locked = $lockedIds.has(feature.id)}
							{@const disabled = isDisabled(feature.id, $disabledIds, $lockedIds)}
							{@const isChecking = $checking.has(feature.id)}
							{@const disableReason = getDisableReason(
								feature.id,
								$incompatibleIds,
								$dynamicDisabled,
								$dynamicDisabledReasons
							)}
							{@const isIncompatOrDynamic = $disabledIds.has(feature.id)}

							<button
								type="button"
								class="border-border bg-card flex items-start gap-3 border-2 p-4 text-left transition-colors
									{selected ? 'border-primary bg-primary/5' : ''}
									{isIncompatOrDynamic ? 'border-muted opacity-50' : ''}
									{!disabled && !isChecking ? 'hover:border-foreground/50 cursor-pointer' : ''}
									{disabled || isChecking ? 'cursor-not-allowed' : ''}"
								onclick={() => {
									if (disabled || isChecking) return;
									handleToggle(feature.id);
								}}
								disabled={disabled || isChecking}
							>
								<div class="pt-0.5">
									{#if isChecking}
										<Loader2 class="text-muted-foreground size-4 animate-spin" />
									{:else if locked && selected}
										<Tooltip.Root>
											<Tooltip.Trigger>
												<Lock class="text-muted-foreground size-4" />
											</Tooltip.Trigger>
											<Tooltip.Content>
												Required by another selected feature
											</Tooltip.Content>
										</Tooltip.Root>
									{:else if disableReason}
										<Tooltip.Root>
											<Tooltip.Trigger>
												<AlertTriangle class="text-destructive size-4" />
											</Tooltip.Trigger>
											<Tooltip.Content>
												{disableReason}
											</Tooltip.Content>
										</Tooltip.Root>
									{:else}
										<Checkbox
											checked={selected}
											disabled={disabled}
											onCheckedChange={() => handleToggle(feature.id)}
											onclick={(e: MouseEvent) => e.stopPropagation()}
										/>
									{/if}
								</div>

								<div class="flex min-w-0 flex-1 flex-col gap-1.5">
									<span class="text-sm font-bold">{feature.name}</span>
									{#if feature.description}
										<span class="text-muted-foreground text-xs leading-relaxed">
											{feature.description}
										</span>
									{/if}

									<div class="flex flex-wrap gap-1.5">
										{#if feature.requires?.length > 0}
											<Badge variant="secondary" class="text-[10px]">
												{feature.requires.length} dep{feature.requires.length > 1 ? "s" : ""}
											</Badge>
										{/if}
										{#if feature.patches?.length > 0}
											<Badge variant="outline" class="text-[10px]">
												<Wrench class="size-2.5" />
												{feature.patches.length} patch{feature.patches.length > 1 ? "es" : ""}
											</Badge>
										{/if}
										{#if feature.config?.length > 0}
											<Badge variant="outline" class="text-[10px]">
												{feature.config.length} config
											</Badge>
										{/if}
									</div>
								</div>
							</button>
						{/each}
					</div>
				{/if}
			</div>
		{/if}
	</div>
{/if}

<style>
	.feature-scroll::-webkit-scrollbar {
		width: 6px;
	}
	.feature-scroll::-webkit-scrollbar-track {
		background: transparent;
	}
	.feature-scroll::-webkit-scrollbar-thumb {
		background: hsl(var(--muted-foreground) / 0.2);
		border-radius: 3px;
	}
	.feature-scroll::-webkit-scrollbar-thumb:hover {
		background: hsl(var(--muted-foreground) / 0.35);
	}
</style>
