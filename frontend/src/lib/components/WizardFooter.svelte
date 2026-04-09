<script lang="ts">
	import { Button } from "$lib/components/ui/button";
	import { canGoBack, canGoForward, nextStep, prevStep } from "$lib/stores/wizard";
	import { ChevronLeft, ChevronRight } from "lucide-svelte";
</script>

<footer class="flex items-center justify-between border-t-2 border-border px-6 py-4">
	{#if $canGoBack}
		<Button variant="outline" onclick={prevStep}>
			<ChevronLeft data-icon="inline-start" />
			Back
		</Button>
	{:else}
		<div></div>
	{/if}
	<Button disabled={!$canGoForward} onclick={nextStep}>
		Next
		<ChevronRight data-icon="inline-end" />
	</Button>
</footer>
