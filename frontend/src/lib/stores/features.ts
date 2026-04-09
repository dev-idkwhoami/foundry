import { writable, derived, get } from "svelte/store";
import type { features } from "$lib/wailsjs/go/models";

// --- Registry (loaded once from backend) ---

export const registry = writable<features.Registry | null>(null);

export const featureList = derived(registry, ($r) =>
  $r?.Features ?? []
);

export const dependencyMap = derived(registry, ($r) =>
  $r?.DependencyMap ?? {}
);

export const incompatMap = derived(registry, ($r) =>
  $r?.IncompatMap ?? {}
);

// --- Selection state ---

/** Set of currently selected feature IDs */
export const selectedIds = writable<Set<string>>(new Set());

/** Set of feature IDs dynamically disabled by patch compat checks */
export const dynamicDisabled = writable<Set<string>>(new Set());

/** Map of feature ID → reason string for dynamic disables */
export const dynamicDisabledReasons = writable<Record<string, string>>({});

/** Features that are currently checking patch compat */
export const checking = writable<Set<string>>(new Set());

// --- Derived constraints ---

/** IDs that are statically incompatible with any selected feature */
export const incompatibleIds = derived(
  [selectedIds, incompatMap],
  ([$selected, $incompat]) => {
    const result = new Set<string>();
    for (const id of $selected) {
      for (const inc of $incompat[id] ?? []) {
        if (!$selected.has(inc)) {
          result.add(inc);
        }
      }
    }
    return result;
  }
);

/** IDs that are required by a selected feature and cannot be deselected */
export const lockedIds = derived(
  [selectedIds, dependencyMap],
  ([$selected, $deps]) => {
    const result = new Set<string>();
    for (const id of $selected) {
      for (const dep of $deps[id] ?? []) {
        if ($selected.has(dep)) {
          result.add(dep);
        }
      }
    }
    return result;
  }
);

/** The full set of disabled IDs (static incompat + dynamic) */
export const disabledIds = derived(
  [incompatibleIds, dynamicDisabled],
  ([$incompat, $dynamic]) => {
    const result = new Set<string>($incompat);
    for (const id of $dynamic) {
      result.add(id);
    }
    return result;
  }
);

// --- Selected features (full objects, in order) ---

export const selectedFeatures = derived(
  [featureList, selectedIds],
  ([$features, $selected]) =>
    $features.filter((f) => $selected.has(f.id))
);

// --- Config values ---

/** Config values keyed by `featureId.configKey` */
export const configValues = writable<Record<string, string>>({});

/** Initialize config defaults for all selected features */
export function initConfigDefaults() {
  const features = get(featureList);
  const selected = get(selectedIds);
  const current = get(configValues);
  const updated = { ...current };

  for (const f of features) {
    if (!selected.has(f.id)) continue;
    for (const c of f.config ?? []) {
      const key = `${f.id}.${c.key}`;
      if (!(key in updated)) {
        updated[key] = c.default ?? "";
      }
    }
  }

  configValues.set(updated);
}

// --- Actions ---

/**
 * Toggle a feature on or off, handling auto-select of dependencies.
 * Returns the list of feature IDs that were auto-selected as dependencies.
 */
export function toggleFeature(id: string): string[] {
  const current = get(selectedIds);
  const next = new Set(current);
  const autoSelected: string[] = [];

  if (current.has(id)) {
    // Deselect — only if not locked
    if (get(lockedIds).has(id)) return [];
    next.delete(id);
  } else {
    // Select — also select all transitive dependencies
    next.add(id);
    const deps = get(dependencyMap);
    const toCheck = [id];
    const visited = new Set<string>();

    while (toCheck.length > 0) {
      const curr = toCheck.pop()!;
      if (visited.has(curr)) continue;
      visited.add(curr);

      for (const dep of deps[curr] ?? []) {
        if (!next.has(dep)) {
          next.add(dep);
          autoSelected.push(dep);
        }
        toCheck.push(dep);
      }
    }
  }

  selectedIds.set(next);
  return autoSelected;
}

/** Remove a feature from selection (used when dynamic compat check fails) */
export function deselectFeature(id: string) {
  const current = get(selectedIds);
  const next = new Set(current);
  next.delete(id);
  selectedIds.set(next);
}

/** Mark a feature as dynamically disabled */
export function addDynamicDisable(id: string, reason: string) {
  dynamicDisabled.update((s) => {
    const next = new Set(s);
    next.add(id);
    return next;
  });
  dynamicDisabledReasons.update((r) => ({ ...r, [id]: reason }));
}

/** Clear a dynamic disable (e.g. when the conflicting feature is deselected) */
export function removeDynamicDisable(id: string) {
  dynamicDisabled.update((s) => {
    const next = new Set(s);
    next.delete(id);
    return next;
  });
  dynamicDisabledReasons.update((r) => {
    const next = { ...r };
    delete next[id];
    return next;
  });
}

/** Reset all selection state */
export function resetSelection() {
  selectedIds.set(new Set());
  dynamicDisabled.set(new Set());
  dynamicDisabledReasons.set({});
  configValues.set({});
  checking.set(new Set());
}
