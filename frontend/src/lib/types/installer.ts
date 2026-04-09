/** ManualPatch represents a patch the user needs to apply manually.
 *  This type mirrors the Go struct installer.ManualPatch but is maintained
 *  manually because Wails only auto-generates types used in bound method signatures. */
export interface ManualPatch {
	featureName: string;
	featureId: string;
	file: string;
	instruction: string;
}
