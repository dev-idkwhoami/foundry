<script lang="ts">
	import * as Accordion from "$lib/components/ui/accordion";
	import { Input } from "$lib/components/ui/input";
	import { Badge } from "$lib/components/ui/badge";

	import { Settings, Eye } from "lucide-svelte";
	import {
		selectedFeatures,
		configValues,
		initConfigDefaults,
	} from "$lib/stores/features";

	const TRANSFORMS = ["lower", "title", "plural", "snake", "camel", "dot"] as const;
	type Transform = (typeof TRANSFORMS)[number];

	let openItems = $state<string[]>([]);
	let debugMode = $state(false);
	let previews = $state<Record<string, Record<Transform, string>>>({});
	let debounceTimers: Record<string, ReturnType<typeof setTimeout>> = {};
	let resolveToken: ((value: string, transforms: string[]) => Promise<string>) | null = $state(null);

	const configurableFeatures = $derived(
		$selectedFeatures.filter((f) => f.config && f.config.length > 0)
	);

	$effect(() => {
		initConfigDefaults();
		openItems = configurableFeatures.map((f) => f.id);
	});

	$effect(() => {
		loadDebugAndResolveToken();
	});

	async function loadDebugAndResolveToken() {
		try {
			const mod = await import("$lib/wailsjs/go/main/App");
			debugMode = await mod.IsDebug();
			if (debugMode) {
				resolveToken = mod.ResolveToken;
			}
		} catch {
			resolveToken = null;
		}
	}

	function updateConfig(featureId: string, key: string, value: string) {
		const storeKey = `${featureId}.${key}`;
		configValues.update((v) => ({ ...v, [storeKey]: value }));
		if (debugMode) schedulePreviews(storeKey, value);
	}

	function schedulePreviews(storeKey: string, value: string) {
		if (debounceTimers[storeKey]) {
			clearTimeout(debounceTimers[storeKey]);
		}

		if (!value.trim()) {
			previews = { ...previews };
			delete previews[storeKey];
			return;
		}

		debounceTimers[storeKey] = setTimeout(() => {
			fetchPreviews(storeKey, value);
		}, 300);
	}

	async function fetchPreviews(storeKey: string, value: string) {
		const results: Record<Transform, string> = {} as Record<Transform, string>;

		if (resolveToken) {
			const promises = TRANSFORMS.map(async (t) => {
				try {
					const result = await resolveToken!(value, [t]);
					results[t] = result;
				} catch {
					results[t] = value;
				}
			});
			await Promise.all(promises);
		} else {
			for (const t of TRANSFORMS) {
				results[t] = clientTransform(value, t);
			}
		}

		previews = { ...previews, [storeKey]: results };
	}

	function clientTransform(value: string, transform: Transform): string {
		switch (transform) {
			case "lower":
				return value.toLowerCase();
			case "title":
				return value.charAt(0).toUpperCase() + value.slice(1);
			case "plural":
				return value.endsWith("s") ? value : value + "s";
			case "snake":
				return value.replace(/([a-z])([A-Z])/g, "$1_$2").replace(/[\s-]+/g, "_").toLowerCase();
			case "camel":
				return value.replace(/[\s_-]+(.)/g, (_, c) => c.toUpperCase()).replace(/^(.)/, (_, c) => c.toLowerCase());
			case "dot":
				return value.replace(/([a-z])([A-Z])/g, "$1.$2").replace(/[\s_-]+/g, ".").toLowerCase();
		}
	}

	function getConfigValue(featureId: string, key: string): string {
		return $configValues[`${featureId}.${key}`] ?? "";
	}
</script>

<div class="mx-auto flex w-full max-w-2xl flex-col gap-6 p-6">
	<div class="flex items-center gap-3">
		<Settings class="size-5 text-foreground" />
		<h2 class="text-lg font-bold">Feature Configuration</h2>
	</div>

	{#if configurableFeatures.length === 0}
		<div class="border-2 border-border bg-muted/30 p-6 text-center">
			<p class="text-sm font-medium text-muted-foreground">
				No features require configuration
			</p>
			<p class="mt-2 text-xs text-muted-foreground/70">
				All selected features use their defaults. You can proceed to the next step.
			</p>
		</div>
	{:else}
		<p class="text-sm text-muted-foreground">
			Configure the selected features below. The live preview shows how values will be transformed in generated code.
		</p>

		<Accordion.Root type="multiple" bind:value={openItems}>
			{#each configurableFeatures as feature (feature.id)}
				<Accordion.Item value={feature.id} class="border-2 border-border mb-2">
					<Accordion.Trigger class="px-4 font-bold">
						<span class="flex items-center gap-2">
							{feature.name}
							<Badge variant="outline" class="text-xs font-normal">
								{feature.config.length} field{feature.config.length > 1 ? "s" : ""}
							</Badge>
						</span>
					</Accordion.Trigger>
					<Accordion.Content class="px-4">
						<div class="flex flex-col gap-4">
							{#each feature.config as field (field.key)}
								{@const storeKey = `${feature.id}.${field.key}`}
								{@const currentValue = getConfigValue(feature.id, field.key)}
								{@const fieldPreviews = previews[storeKey]}

								<div class="flex flex-col gap-1.5">
									<label for={storeKey} class="text-sm font-medium">
										{field.label}
									</label>
									{#if field.type === "select" && field.options?.length > 0}
										<select
											id={storeKey}
											class="h-8 w-full border-2 border-input bg-background text-foreground px-2.5 py-1 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
											value={currentValue}
											onchange={(e) => {
												const target = e.target as HTMLSelectElement;
												updateConfig(feature.id, field.key, target.value);
											}}
										>
											{#each field.options as opt (opt.value)}
												<option value={opt.value} selected={currentValue === opt.value}>
													{opt.label}
												</option>
											{/each}
										</select>
									{:else}
										<Input
											id={storeKey}
											type="text"
											value={currentValue}
											placeholder={field.placeholder || field.default}
											oninput={(e: Event) => {
												const target = e.target as HTMLInputElement;
												updateConfig(feature.id, field.key, target.value);
											}}
											class="border-2 rounded-none"
										/>
									{/if}

									{#if debugMode && currentValue.trim() && fieldPreviews}
										<div class="mt-1 border-2 border-border/50 bg-muted/40 p-3">
											<div class="mb-1.5 flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
												<Eye class="size-3" />
												Transformer Preview
											</div>
											<div class="grid grid-cols-2 gap-x-4 gap-y-1">
												{#each TRANSFORMS as transform (transform)}
													<div class="flex items-center justify-between gap-2 text-xs">
														<span class="font-mono text-muted-foreground">
															{transform}
														</span>
														<span class="font-mono font-medium text-foreground">
															{fieldPreviews[transform] ?? "..."}
														</span>
													</div>
												{/each}
											</div>
										</div>
									{/if}
								</div>
							{/each}
						</div>
					</Accordion.Content>
				</Accordion.Item>
			{/each}
		</Accordion.Root>
	{/if}
</div>
