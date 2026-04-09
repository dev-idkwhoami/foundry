import { writable, derived } from "svelte/store";

export const projectName = writable("");
export const workingDir = writable("");
export const hasContext = writable(false);

export const targetPath = derived(
  [workingDir, projectName],
  ([$dir, $name]) => {
    if (!$dir || !$name) return "";
    // Normalize to backslashes for consistent Windows paths
    return `${$dir}\\${$name}`.replace(/\//g, "\\");
  }
);
