import { writable, derived, get } from "svelte/store";
import { hasContext, projectName } from "./project";
import { selectedIds } from "./features";
import { prerequisitesMet } from "./install";

export type StepId =
  | "loading"
  | "dir-select"
  | "project-setup"
  | "features"
  | "config"
  | "review"
  | "install"
  | "manual";

export const STEPS: { id: StepId; label: string; showInNav: boolean }[] = [
  { id: "loading", label: "Loading", showInNav: false },
  { id: "dir-select", label: "Directory", showInNav: false },
  { id: "project-setup", label: "Project Setup", showInNav: true },
  { id: "features", label: "Features", showInNav: true },
  { id: "config", label: "Configuration", showInNav: true },
  { id: "review", label: "Review", showInNav: true },
  { id: "install", label: "Install", showInNav: false },
  { id: "manual", label: "Manual Steps", showInNav: false },
];

export const currentStepId = writable<StepId>("loading");

export const navSteps = STEPS.filter((s) => s.showInNav);

/** Determines the first real step based on startup context */
export function resolveFirstStep(ctx: {
  hasContext: boolean;
  projectName: string;
}): StepId {
  if (!ctx.hasContext) return "dir-select";
  if (ctx.projectName) return "features";
  return "project-setup";
}

export const canGoBack = derived(currentStepId, ($id) => {
  const navIds: StepId[] = ["project-setup", "features", "config", "review"];
  const idx = navIds.indexOf($id);
  return idx > 0;
});

export const canGoForward = derived(
  [currentStepId, projectName, selectedIds, prerequisitesMet],
  ([$id, $name, $selected, $prereqs]) => {
    const navIds: StepId[] = ["project-setup", "features", "config", "review"];
    const idx = navIds.indexOf($id);
    if (idx < 0 || idx >= navIds.length - 1) return false;

    if ($id === "project-setup" && (!$name || !$prereqs)) return false;
    if ($id === "features" && $selected.size === 0) return false;

    return true;
  }
);

const navOrder: StepId[] = [
  "project-setup",
  "features",
  "config",
  "review",
];

export function nextStep() {
  const $id = get(currentStepId);
  const idx = navOrder.indexOf($id);
  if (idx >= 0 && idx < navOrder.length - 1) {
    currentStepId.set(navOrder[idx + 1]);
  }
}

export function prevStep() {
  const $id = get(currentStepId);
  const idx = navOrder.indexOf($id);
  if (idx > 0) {
    currentStepId.set(navOrder[idx - 1]);
  }
}

export function goToStep(step: StepId) {
  currentStepId.set(step);
}
