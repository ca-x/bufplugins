package generator

import "testing"

func TestOptionsValidateRejectsEmptyRuntimeImport(t *testing.T) {
	opts := DefaultOptions()
	opts.RuntimeImport = "  "

	if err := opts.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want error")
	}
}
