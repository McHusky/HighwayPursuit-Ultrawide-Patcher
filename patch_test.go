package main

import (
	"bytes"
	"os"
	"testing"
)

func mustLoadVariants(t *testing.T) []patchVariant {
	t.Helper()
	variants, err := loadPatchVariants()
	if err != nil {
		t.Fatal(err)
	}
	if len(variants) < 2 {
		t.Fatalf("expected at least two supported variants, got %d", len(variants))
	}
	return variants
}

func TestEmbeddedPatchesParse(t *testing.T) {
	variants := mustLoadVariants(t)
	for _, variant := range variants {
		t.Run(variant.Definition.ID, func(t *testing.T) {
			if len(variant.Spec.Records) == 0 {
				t.Fatal("no patch records")
			}
			for i, rec := range variant.Spec.Records {
				if len(rec.Original) == 0 || len(rec.Original) != len(rec.Patched) {
					t.Fatalf("record %d has invalid lengths", i)
				}
				if bytes.Equal(rec.Original, rec.Patched) {
					t.Fatalf("record %d does not change bytes", i)
				}
			}
		})
	}
}

func syntheticOriginal(spec patchSpec) []byte {
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
	return data
}

func TestUnrelatedChangesRemainCompatible(t *testing.T) {
	for _, variant := range mustLoadVariants(t) {
		t.Run(variant.Definition.ID, func(t *testing.T) {
			data := syntheticOriginal(variant.Spec)
			check, err := inspectExecutable(data, variant.Spec)
			if err != nil || check.State != stateOriginal {
				t.Fatalf("expected original-compatible state, got %+v, err=%v", check, err)
			}

			// Change a byte outside all guarded records. Whole-file hash gating would
			// reject this, but patch-site compatibility checking should still allow it.
			data[len(data)-1] ^= 0x5A
			check, err = inspectExecutable(data, variant.Spec)
			if err != nil || check.State != stateOriginal {
				t.Fatalf("unrelated change should remain compatible, got %+v, err=%v", check, err)
			}
		})
	}
}

func TestChangedPatchSiteIsRejected(t *testing.T) {
	for _, variant := range mustLoadVariants(t) {
		t.Run(variant.Definition.ID, func(t *testing.T) {
			data := syntheticOriginal(variant.Spec)
			first := variant.Spec.Records[0]
			data[int(first.Offset)] ^= 0x7F
			check, err := inspectExecutable(data, variant.Spec)
			if err != nil {
				t.Fatal(err)
			}
			if check.State != stateUnknown || len(check.UnknownRecords) == 0 {
				t.Fatalf("changed patch site should be rejected, got %+v", check)
			}
		})
	}
}

func TestVariantSelectionOnSyntheticInputs(t *testing.T) {
	variants := mustLoadVariants(t)
	for _, want := range variants {
		data := syntheticOriginal(want.Spec)
		checks := inspectVariants(data, variants)
		matches := checksWithState(checks, stateOriginal)
		if len(matches) != 1 {
			t.Fatalf("%s: expected exactly one original match, got %d", want.Definition.ID, len(matches))
		}
		if matches[0].Variant.Definition.ID != want.Definition.ID {
			t.Fatalf("%s: matched %s", want.Definition.ID, matches[0].Variant.Definition.ID)
		}
	}
}

func applySpec(data []byte, spec patchSpec) []byte {
	patched := append([]byte(nil), data...)
	for _, rec := range spec.Records {
		end := int(rec.Offset) + len(rec.Patched)
		copy(patched[int(rec.Offset):end], rec.Patched)
	}
	return patched
}

// Optional integration test. It is intentionally skipped unless a locally owned
// original executable path is supplied, so the repository never needs game files.
// The patcher automatically selects whichever embedded layout matches the file.
func TestPatchAgainstOriginal(t *testing.T) {
	path := os.Getenv("HP_ORIGINAL_EXE")
	if path == "" {
		t.Skip("set HP_ORIGINAL_EXE to run the integration test")
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	variants := mustLoadVariants(t)
	checks := inspectVariants(original, variants)
	matches := checksWithState(checks, stateOriginal)
	if len(matches) != 1 {
		t.Fatalf("input did not match exactly one supported original layout: %+v", checks)
	}
	selected := matches[0].Variant
	patched := applySpec(original, selected.Spec)
	check, err := inspectExecutable(patched, selected.Spec)
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
