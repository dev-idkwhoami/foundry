import { writable } from "svelte/store";
import type { ManualPatch } from "$lib/types/installer";

/** Manual patches collected during installation, consumed by ManualChecklist. */
export const manualSteps = writable<ManualPatch[]>([]);

/** Whether critical prerequisites (Git, Herd) are met. Blocks wizard advancement. */
export const prerequisitesMet = writable(false);
