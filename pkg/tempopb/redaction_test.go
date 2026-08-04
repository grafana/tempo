package tempopb

import "testing"

func TestRedactionModeIsDryRun(t *testing.T) {
	if !RedactionMode_REDACTION_MODE_DRY_RUN.IsDryRun() {
		t.Error("DRY_RUN mode must report IsDryRun() == true")
	}
	if RedactionMode_REDACTION_MODE_APPLY.IsDryRun() {
		t.Error("APPLY mode must report IsDryRun() == false")
	}
}
