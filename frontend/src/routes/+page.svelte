<script lang="ts">
	import WizardNav from "$lib/components/WizardNav.svelte";
	import WizardFooter from "$lib/components/WizardFooter.svelte";
	import StartupScreen from "$lib/components/steps/StartupScreen.svelte";
	import DirectorySelection from "$lib/components/steps/DirectorySelection.svelte";
	import ProjectSetup from "$lib/components/steps/ProjectSetup.svelte";
	import FeatureSelection from "$lib/components/steps/FeatureSelection.svelte";
	import FeatureConfig from "$lib/components/steps/FeatureConfig.svelte";
	import ReviewSummary from "$lib/components/steps/ReviewSummary.svelte";
	import InstallProgress from "$lib/components/steps/InstallProgress.svelte";
	import ManualChecklist from "$lib/components/steps/ManualChecklist.svelte";
	import SettingsPage from "$lib/components/SettingsPage.svelte";
	import ProjectManager from "$lib/components/ProjectManager.svelte";
	import { currentStepId, goToStep, resolveFirstStep } from "$lib/stores/wizard";
	import { projectName, workingDir, hasContext } from "$lib/stores/project";
	import { FolderTree, Settings, Archive, X } from "lucide-svelte";
	import { onMount } from "svelte";

	let showSettings = $state(false);
	let showProjects = $state(false);
	let projectSetupRef: ReturnType<typeof ProjectSetup> | undefined = $state(undefined);

	function quit() {
		import("$lib/wailsjs/go/main/App").then(({ Quit }) => Quit()).catch(() => window.close());
	}

	const showNav = $derived(
		["project-setup", "features", "config", "review"].includes($currentStepId)
	);

	const showFooter = $derived(
		["project-setup", "features", "config"].includes($currentStepId)
	);

	onMount(async () => {
		try {
			const { GetStartupContext } = await import("$lib/wailsjs/go/main/App");
			const ctx = await GetStartupContext();

			$hasContext = ctx.hasContext;
			if (ctx.projectName) $projectName = ctx.projectName;
			if (ctx.workingDir) $workingDir = ctx.workingDir;

			// Stay on "loading" — StartupScreen handles the transition after clone completes
		} catch {
			// Dev mode without Wails — skip to dir-select
			setTimeout(() => goToStep("dir-select"), 1000);
		}
	});
</script>

<div class="flex h-screen flex-col bg-background">
	<!-- Title bar (drag region) -->
	<header
		class="flex items-center gap-3 border-b-2 border-border px-6 py-3"
		style="--wails-draggable: drag"
	>
		<FolderTree color="#00e6d6" class="h-6 w-6" />
		<h1 class="text-lg font-bold tracking-tight">Foundry</h1>
		<div class="ml-auto flex items-center gap-1.5" style="--wails-draggable: no-drag">
			<button
				onclick={() => (showProjects = true)}
				class="flex h-7 w-7 items-center justify-center border-2 border-border bg-background text-foreground transition-colors hover:bg-muted"
				title="Manage Projects"
			>
				<Archive class="h-4 w-4" />
			</button>
			<button
				onclick={() => (showSettings = true)}
				class="flex h-7 w-7 items-center justify-center border-2 border-border bg-background text-foreground transition-colors hover:bg-muted"
			>
				<Settings class="h-4 w-4" />
			</button>
			<button
				onclick={quit}
				class="flex h-7 w-7 items-center justify-center border-2 border-border bg-background text-foreground transition-colors hover:bg-destructive hover:text-destructive-foreground"
			>
				<X class="h-4 w-4" />
			</button>
		</div>
	</header>

	<!-- Step nav -->
	{#if showNav}
		<WizardNav />
	{/if}

	<!-- Step content -->
	<main class="flex flex-1 flex-col overflow-y-auto p-6">
		{#if $currentStepId === "loading"}
			<StartupScreen />
		{:else if $currentStepId === "dir-select"}
			<DirectorySelection />
		{:else if $currentStepId === "project-setup"}
			<ProjectSetup bind:this={projectSetupRef} />
		{:else if $currentStepId === "features"}
			<FeatureSelection />
		{:else if $currentStepId === "config"}
			<FeatureConfig />
		{:else if $currentStepId === "review"}
			<ReviewSummary />
		{:else if $currentStepId === "install"}
			<InstallProgress />
		{:else if $currentStepId === "manual"}
			<ManualChecklist />
		{/if}
	</main>

	<!-- Footer nav -->
	{#if showFooter}
		<WizardFooter />
	{/if}
</div>

{#if showSettings}
	<SettingsPage onclose={() => { showSettings = false; projectSetupRef?.refreshEnv(); }} />
{/if}

{#if showProjects}
	<ProjectManager onclose={() => { showProjects = false; }} />
{/if}
