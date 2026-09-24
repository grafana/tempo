package tempopb

// IsDryRun reports whether the redaction mode is a dry-run: evaluate and count matches without
// rewriting any blocks. Centralizes the mode comparison used across the backend-scheduler
// redaction lifecycle (compaction gating, quiescence, rescan, metrics).
func (m RedactionMode) IsDryRun() bool {
	return m == RedactionMode_REDACTION_MODE_DRY_RUN
}
