<script lang="ts">
	import { currentStepId, navSteps, STEPS } from "$lib/stores/wizard";
	import { cn } from "$lib/utils";

	function stepIndex(id: string): number {
		return STEPS.findIndex((s) => s.id === id);
	}
</script>

<nav class="flex items-center gap-2 border-b-2 border-border px-6 py-4">
	{#each navSteps as step, i}
		{@const currentIdx = stepIndex($currentStepId)}
		{@const thisIdx = stepIndex(step.id)}
		{@const isActive = $currentStepId === step.id}
		{@const isCompleted = currentIdx > thisIdx}

		{#if i > 0}
			<div
				class={cn(
					"h-0.5 w-8 transition-colors",
					isCompleted ? "bg-primary" : "bg-border"
				)}
			></div>
		{/if}

		<div class="flex items-center gap-2">
			<span
				class={cn(
					"flex h-7 w-7 items-center justify-center border-2 text-xs font-bold transition-colors",
					isActive && "border-primary bg-primary text-primary-foreground",
					isCompleted && "border-primary bg-primary text-primary-foreground",
					!isActive && !isCompleted && "border-border text-muted-foreground"
				)}
			>
				{i + 1}
			</span>
			<span
				class={cn(
					"text-sm font-medium transition-colors",
					isActive ? "text-foreground" : "text-muted-foreground"
				)}
			>
				{step.label}
			</span>
		</div>
	{/each}
</nav>
