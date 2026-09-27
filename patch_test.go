package main

import (
	"bytes"
	"os"
	"testing"
)

func TestEmbeddedPatchParses(t *testing.T) {
	spec, err := loadPatchSpec()
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Records) == 0 {
		t.Fatal("no patch records")
	}
	for i, rec := range spec.Records {
		if len(rec.Original) == 0 || len(rec.Original) != len(rec.Patched) {
			t.Fatalf("record %d has invalid lengths", i)
		}
		if bytes.Equal(rec.Original, rec.Patched) {
			t.Fatalf("record %d does not change bytes", i)
		}
	}
}

func TestUnrelatedChangesRemainCompatible(t *testing.T) {
	spec, err := loadPatchSpec()
	if err != nil {
		t.Fatal(err)
	}
	maxEnd := 0
	for _, rec := range spec.Records {
		end := int(rec.Offset) + len(rec.Original)
		if end > maxEnd {
			maxEnd = end
		}
	}
	data := make([]byte, maxEnd+32)
	copy(data[:2], []byte("MZ"))
	for _, rec := range spec.Records {
		copy(data[int(rec.Offset):int(rec.Offset)+len(rec.Original)], rec.Original)
	}

	check, err := inspectExecutable(data, spec)
	if err != nil || check.State != stateOriginal {
		t.Fatalf("expected original-compatible state, got %+v, err=%v", check, err)
	}

	// Change a byte outside all patch records. Whole-file hash gating would reject
	// this, but patch-site compatibility checking should still allow it.
	data[len(data)-1] ^= 0x5A
	check, err = inspectExecutable(data, spec)
	if err != nil || check.State != stateOriginal {
		t.Fatalf("unrelated change should remain compatible, got %+v, err=%v", check, err)
	}
}

func TestChangedPatchSiteIsRejected(t *testing.T) {
	spec, err := loadPatchSpec()
	if err != nil {
		t.Fatal(err)
	}
	maxEnd := 0
	for _, rec := range spec.Records {
		end := int(rec.Offset) + len(rec.Original)
		if end > maxEnd {
			maxEnd = end
		}
	}
	data := make([]byte, maxEnd+1)
	for _, rec := range spec.Records {
		copy(data[int(rec.Offset):int(rec.Offset)+len(rec.Original)], rec.Original)
	}
	first := spec.Records[0]
	data[int(first.Offset)] ^= 0x7F
	check, err := inspectExecutable(data, spec)
	if err != nil {
		t.Fatal(err)
	}
	if check.State != stateUnknown || len(check.UnknownRecords) == 0 {
		t.Fatalf("changed patch site should be rejected, got %+v", check)
	}
}

// Optional integration test. It is intentionally skipped unless a locally owned
// original executable path is supplied, so the repository never needs game files.
func TestPatchAgainstOriginal(t *testing.T) {
	path := os.Getenv("HP_ORIGINAL_EXE")
	if path == "" {
		t.Skip("set HP_ORIGINAL_EXE to run the integration test")
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := loadPatchSpec()
	if err != nil {
		t.Fatal(err)
	}
	check, err := inspectExecutable(original, spec)
	if err != nil {
		t.Fatal(err)
	}
	if check.State != stateOriginal {
		t.Fatalf("input is not compatible with the guarded patch: %+v", check)
	}

	patched := append([]byte(nil), original...)
	for _, rec := range spec.Records {
		end := int(rec.Offset) + len(rec.Patched)
		copy(patched[int(rec.Offset):end], rec.Patched)
	}
	check, err = inspectExecutable(patched, spec)
	if err != nil {
		t.Fatal(err)
	}
	if check.State != statePatched {
		t.Fatalf("patched output does not match embedded patch records: %+v", check)
	}

	if expectedPath := os.Getenv("HP_EXPECTED_PATCHED_EXE"); expectedPath != "" {
		expected, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(patched, expected) {
			t.Fatal("patched output differs from HP_EXPECTED_PATCHED_EXE")
		}
	}
}
