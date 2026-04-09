<script lang="ts">
	import { Input } from "$lib/components/ui/input";
	import { Button } from "$lib/components/ui/button";
	import { Separator } from "$lib/components/ui/separator";
	import { X, Eye, EyeOff, Save } from "lucide-svelte";

	let { onclose }: { onclose: () => void } = $props();

	let fluxUsername = $state("");
	let fluxKey = $state("");
	let repository = $state("");
	let showKey = $state(false);
	let saving = $state(false);
	let loading = $state(true);
	let error = $state("");

	$effect(() => {
		loadSettings();
	});

	async function loadSettings() {
		try {
			const app = await import("$lib/wailsjs/go/main/App");
			const [cfg, key, username] = await Promise.all([
				app.GetConfig(),
				app.GetFluxLicenseKey(),
				app.GetFluxUsername(),
			]);
			repository = cfg.repository;
			fluxKey = key;
			fluxUsername = username;
		} catch (err) {
			error = String(err);
		} finally {
			loading = false;
		}
	}

	async function save() {
		saving = true;
		error = "";
		try {
			const app = await import("$lib/wailsjs/go/main/App");
			await app.SetFluxUsername(fluxUsername);
			await app.SetFluxLicenseKey(fluxKey);
			await app.SetRepository(repository);
			onclose();
		} catch (err) {
			error = String(err);
			saving = false;
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
	<div class="w-full max-w-lg border-2 border-border bg-background shadow-[4px_4px_0_0_rgba(0,0,0,0.2)]">
		<div class="flex items-center justify-between border-b-2 border-border px-6 py-4">
			<h2 class="text-lg font-bold tracking-tight">Settings</h2>
			<Button variant="ghost" size="icon" onclick={onclose}>
				<X class="size-4" />
			</Button>
		</div>

		<div class="px-6 py-5">
			{#if loading}
				<p class="text-sm text-muted-foreground">Loading settings…</p>
			{:else}
				<div class="space-y-5">
					<div class="space-y-2">
						<label for="flux-username" class="text-sm font-bold">
							Flux UI Pro Username
						</label>
						<Input
							id="flux-username"
							type="text"
							bind:value={fluxUsername}
							placeholder="Email or username"
							class="!rounded-none border-2"
						/>
						<p class="text-xs text-muted-foreground">
							Your email or username for composer.fluxui.dev.
						</p>
					</div>

					<div class="space-y-2">
						<label for="flux-key" class="text-sm font-bold">
							Flux UI Pro License Key
						</label>
						<div class="flex gap-2">
							<Input
								id="flux-key"
								type={showKey ? "text" : "password"}
								bind:value={fluxKey}
								placeholder="Enter your license key"
								class="flex-1 !rounded-none border-2"
							/>
							<Button
								variant="outline"
								size="icon"
								class="shrink-0 !rounded-none border-2"
								onclick={() => (showKey = !showKey)}
							>
								{#if showKey}
									<EyeOff class="size-4" />
								{:else}
									<Eye class="size-4" />
								{/if}
							</Button>
						</div>
						<p class="text-xs text-muted-foreground">
							Required for installing Flux UI Pro components.
						</p>
					</div>

					<Separator class="!bg-border" />

					<div class="space-y-2">
						<label for="repository" class="text-sm font-bold">
							Repository URL
						</label>
						<Input
							id="repository"
							type="text"
							bind:value={repository}
							placeholder="https://github.com/user/repo"
							class="!rounded-none border-2"
						/>
						<p class="text-xs text-muted-foreground">
							Git repository used as the base for new projects.
						</p>
					</div>

					{#if error}
						<p class="text-sm font-medium text-destructive">{error}</p>
					{/if}

					<Button
						class="w-full !rounded-none border-2 font-bold"
						disabled={saving}
						onclick={save}
					>
						<Save class="size-4" />
						{saving ? "Saving…" : "Save & Close"}
					</Button>
				</div>
			{/if}
		</div>
	</div>
</div>
