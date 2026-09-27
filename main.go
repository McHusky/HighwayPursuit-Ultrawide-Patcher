package main

import (
	"bytes"
	"embed"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	appName     = "Highway Pursuit Modern Display Patcher"
	appVersion  = "1.1.0"
	gameExeName = "HighwayPursuit.exe"
	patchMagic  = "HPUPATCH2"
	patchAsset  = "assets/highway_pursuit_v6_guarded.patch"
)

//go:embed assets/highway_pursuit_v6_guarded.patch
var patchFS embed.FS

type patchRecord struct {
	Offset   uint32
	Original []byte
	Patched  []byte
}

type patchSpec struct {
	Records []patchRecord
}

type executableState int

const (
	stateUnknown executableState = iota
	stateOriginal
	statePatched
	stateMixed
)

type inspection struct {
	State           executableState
	OriginalMatches int
	PatchedMatches  int
	UnknownRecords  []int
}

func main() {
	spec, err := loadPatchSpec()
	if err != nil {
		showError(appName, "The embedded patch data could not be read.\n\n"+err.Error())
		return
	}

	restore := false
	var explicitPath string
	for _, arg := range os.Args[1:] {
		switch strings.ToLower(arg) {
		case "--restore", "/restore", "-restore":
			restore = true
		case "--help", "-h", "/?":
			showInfo(appName, usageText())
			return
		default:
			if explicitPath == "" {
				explicitPath = arg
			}
		}
	}

	target, err := findTarget(explicitPath)
	if err != nil {
		showError(appName, err.Error()+"\n\n"+usageText())
		return
	}

	if restore {
		if err := restoreOriginal(target, spec); err != nil {
			showError(appName, "Restore failed.\n\n"+err.Error())
			return
		}
		showInfo(appName, "Original HighwayPursuit.exe restored successfully.")
		return
	}

	if err := patchGame(target, spec); err != nil {
		showError(appName, "Patch failed.\n\n"+err.Error())
		return
	}
}

func usageText() string {
	return "Usage:\n\n" +
		"1. Put this patcher next to HighwayPursuit.exe and double-click it, OR\n" +
		"2. Drag HighwayPursuit.exe onto the patcher.\n\n" +
		"Restore: run the patcher with --restore.\n\n" +
		"No Python, .NET runtime, Visual C++ runtime, or installer is required."
}

func findTarget(explicit string) (string, error) {
	if explicit != "" {
		p, err := filepath.Abs(explicit)
		if err != nil {
			return "", err
		}
		info, err := os.Stat(p)
		if err != nil {
			return "", fmt.Errorf("cannot open %q: %w", p, err)
		}
		if info.IsDir() {
			p = filepath.Join(p, gameExeName)
		}
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("%s was not found at %q", gameExeName, p)
		}
		return p, nil
	}

	exe, err := os.Executable()
	if err == nil {
		p := filepath.Join(filepath.Dir(exe), gameExeName)
		if _, statErr := os.Stat(p); statErr == nil {
			return p, nil
		}
	}

	cwd, err := os.Getwd()
	if err == nil {
		p := filepath.Join(cwd, gameExeName)
		if _, statErr := os.Stat(p); statErr == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("%s was not found", gameExeName)
}

func loadPatchSpec() (patchSpec, error) {
	var spec patchSpec
	blob, err := patchFS.ReadFile(patchAsset)
	if err != nil {
		return spec, err
	}
	min := len(patchMagic) + 4
	if len(blob) < min || string(blob[:len(patchMagic)]) != patchMagic {
		return spec, errors.New("invalid patch header")
	}
	pos := len(patchMagic)
	count := int(binary.LittleEndian.Uint32(blob[pos : pos+4]))
	pos += 4
	if count <= 0 {
		return spec, errors.New("patch contains no records")
	}

	spec.Records = make([]patchRecord, 0, count)
	var lastEnd uint64
	for i := 0; i < count; i++ {
		if pos+8 > len(blob) {
			return patchSpec{}, fmt.Errorf("truncated patch record %d", i)
		}
		off := binary.LittleEndian.Uint32(blob[pos : pos+4])
		length := binary.LittleEndian.Uint32(blob[pos+4 : pos+8])
		pos += 8
		if length == 0 {
			return patchSpec{}, fmt.Errorf("invalid patch record %d", i)
		}
		need := uint64(length) * 2
		if uint64(pos)+need > uint64(len(blob)) {
			return patchSpec{}, fmt.Errorf("truncated patch record %d", i)
		}
		end := uint64(off) + uint64(length)
		if i > 0 && uint64(off) < lastEnd {
			return patchSpec{}, fmt.Errorf("overlapping patch record %d", i)
		}
		original := append([]byte(nil), blob[pos:pos+int(length)]...)
		pos += int(length)
		patched := append([]byte(nil), blob[pos:pos+int(length)]...)
		pos += int(length)
		if bytes.Equal(original, patched) {
			return patchSpec{}, fmt.Errorf("patch record %d does not change any bytes", i)
		}
		spec.Records = append(spec.Records, patchRecord{Offset: off, Original: original, Patched: patched})
		lastEnd = end
	}
	if pos != len(blob) {
		return patchSpec{}, errors.New("unexpected trailing patch data")
	}
	return spec, nil
}

func inspectExecutable(data []byte, spec patchSpec) (inspection, error) {
	result := inspection{}
	for i, rec := range spec.Records {
		end := uint64(rec.Offset) + uint64(len(rec.Original))
		if end > uint64(len(data)) {
			return inspection{}, fmt.Errorf("patch record %d is outside the executable", i)
		}
		current := data[int(rec.Offset):int(end)]
		switch {
		case bytes.Equal(current, rec.Original):
			result.OriginalMatches++
		case bytes.Equal(current, rec.Patched):
			result.PatchedMatches++
		default:
			result.UnknownRecords = append(result.UnknownRecords, i)
		}
	}

	switch {
	case len(result.UnknownRecords) > 0:
		result.State = stateUnknown
	case result.OriginalMatches == len(spec.Records):
		result.State = stateOriginal
	case result.PatchedMatches == len(spec.Records):
		result.State = statePatched
	default:
		result.State = stateMixed
	}
	return result, nil
}

func patchGame(target string, spec patchSpec) error {
	data, mode, err := readFileWithMode(target)
	if err != nil {
		return err
	}
	if len(data) < 2 || string(data[:2]) != "MZ" {
		return errors.New("the selected file is not a Windows executable")
	}

	check, err := inspectExecutable(data, spec)
	if err != nil {
		return err
	}
	switch check.State {
	case statePatched:
		showInfo(appName, "This HighwayPursuit.exe is already patched.\n\nVersion: "+appVersion)
		return nil
	case stateMixed:
		return fmt.Errorf(
			"this executable appears to be partially patched or based on a different patch revision\n\n%d of %d patch locations contain patched bytes and %d still contain original bytes.\n\nNo files were changed.",
			check.PatchedMatches, len(spec.Records), check.OriginalMatches,
		)
	case stateUnknown:
		return fmt.Errorf(
			"this HighwayPursuit.exe is not compatible with this patch revision\n\n%d of %d required patch locations differ from both the expected original and patched bytes%s\n\nThe patcher does not require a whole-file SHA-256 match, so unrelated game updates are allowed. However, changed bytes at a required patch location are rejected for safety.\n\nNo files were changed.",
			len(check.UnknownRecords), len(spec.Records), formatRecordList(check.UnknownRecords, spec),
		)
	case stateOriginal:
		// Safe to continue.
	default:
		return errors.New("could not determine executable compatibility")
	}

	if !askYesNo(appName,
		"Compatible HighwayPursuit.exe found.\n\n"+
			"This patch adds:\n"+
			"- 1920x1080, 2160x1440, 3840x2160 and 5120x1440\n"+
			"- Borderless fullscreen\n"+
			"- DPI-aware positioning\n"+
			"- Centered ultrawide HUD\n\n"+
			"Compatibility is checked only at the required patch locations, not by a whole-file SHA-256 hash.\n"+
			"An exact backup of your current executable will be kept automatically.\n\nApply the patch now?") {
		return nil
	}

	backup, err := ensureBackup(target, data, mode)
	if err != nil {
		return fmt.Errorf("could not create backup: %w", err)
	}

	patched := append([]byte(nil), data...)
	for _, rec := range spec.Records {
		end := int(rec.Offset) + len(rec.Patched)
		copy(patched[int(rec.Offset):end], rec.Patched)
	}
	preWriteCheck, err := inspectExecutable(patched, spec)
	if err != nil || preWriteCheck.State != statePatched {
		return errors.New("internal verification failed before writing; no files were changed")
	}

	if err := atomicReplace(target, patched, mode); err != nil {
		_ = restoreFromBackup(target, backup, mode)
		return fmt.Errorf("could not write patched executable: %w\n\nThe original backup was restored from:\n%s", err, backup)
	}

	finalData, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(finalData, patched) {
		_ = restoreFromBackup(target, backup, mode)
		return errors.New("verification after writing failed; the original backup was restored")
	}

	showInfo(appName,
		"Patch applied successfully.\n\n"+
			"Backup:\n"+backup+"\n\n"+
			"For XSplit Broadcaster, use Window Capture rather than Game Capture. "+
			"Game Capture can cause stutter with Highway Pursuit's Direct3D 8 renderer.")
	return nil
}

func formatRecordList(records []int, spec patchSpec) string {
	if len(records) == 0 {
		return ""
	}
	limit := len(records)
	if limit > 6 {
		limit = 6
	}
	parts := make([]string, 0, limit)
	for _, idx := range records[:limit] {
		if idx >= 0 && idx < len(spec.Records) {
			parts = append(parts, fmt.Sprintf("0x%X", spec.Records[idx].Offset))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	suffix := "\nFirst differing offsets: " + strings.Join(parts, ", ")
	if len(records) > limit {
		suffix += fmt.Sprintf(" (+%d more)", len(records)-limit)
	}
	return suffix
}

func readFileWithMode(path string) ([]byte, os.FileMode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, 0, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	return data, info.Mode(), nil
}

func ensureBackup(target string, original []byte, mode os.FileMode) (string, error) {
	dir := filepath.Dir(target)
	base := strings.TrimSuffix(filepath.Base(target), filepath.Ext(target))
	candidates := []string{filepath.Join(dir, base+".original.exe")}
	for i := 1; i <= 99; i++ {
		candidates = append(candidates, filepath.Join(dir, fmt.Sprintf("%s.original.%d.exe", base, i)))
	}

	for _, p := range candidates {
		existing, err := os.ReadFile(p)
		if err == nil {
			if bytes.Equal(existing, original) {
				return p, nil
			}
			continue
		}
		if !os.IsNotExist(err) {
			continue
		}
		if err := writeNewFile(p, original, mode); err != nil {
			continue
		}
		verify, err := os.ReadFile(p)
		if err != nil || !bytes.Equal(verify, original) {
			_ = os.Remove(p)
			continue
		}
		return p, nil
	}
	return "", errors.New("could not find a safe backup filename")
}

func writeNewFile(path string, data []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode.Perm())
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	ok = true
	return nil
}

func atomicReplace(target string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(target)
	temp, err := os.CreateTemp(dir, ".hpursuit-patch-*.tmp")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	keep := false
	defer func() {
		_ = temp.Close()
		if !keep {
			_ = os.Remove(tempName)
		}
	}()

	if err := temp.Chmod(mode.Perm()); err != nil {
		return err
	}
	if _, err := temp.Write(data); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}

	// os.Rename cannot replace an existing file on Windows, so remove the
	// original only after the complete temporary file is safely on disk.
	if err := os.Remove(target); err != nil {
		return err
	}
	if err := os.Rename(tempName, target); err != nil {
		return err
	}
	keep = true
	return nil
}

func restoreFromBackup(target, backup string, mode os.FileMode) error {
	data, err := os.ReadFile(backup)
	if err != nil {
		return err
	}
	return os.WriteFile(target, data, mode.Perm())
}

func reconstructedOriginal(current []byte, spec patchSpec) ([]byte, error) {
	original := append([]byte(nil), current...)
	for i, rec := range spec.Records {
		end := uint64(rec.Offset) + uint64(len(rec.Original))
		if end > uint64(len(original)) {
			return nil, fmt.Errorf("patch record %d is outside the executable", i)
		}
		copy(original[int(rec.Offset):int(end)], rec.Original)
	}
	return original, nil
}

func restoreOriginal(target string, spec patchSpec) error {
	current, mode, err := readFileWithMode(target)
	if err != nil {
		return err
	}
	check, err := inspectExecutable(current, spec)
	if err == nil && check.State == stateOriginal {
		return errors.New("the selected HighwayPursuit.exe already contains the original bytes at all patch locations")
	}

	expectedOriginal, err := reconstructedOriginal(current, spec)
	if err != nil {
		return err
	}

	dir := filepath.Dir(target)
	base := strings.TrimSuffix(filepath.Base(target), filepath.Ext(target))
	matches, _ := filepath.Glob(filepath.Join(dir, base+".original*.exe"))
	sort.Strings(matches)

	var backup string
	var backupData []byte
	for _, p := range matches {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if bytes.Equal(b, expectedOriginal) {
			backup = p
			backupData = b
			break
		}
	}
	if backup == "" {
		return errors.New("no exact backup matching this executable was found next to the game")
	}

	if !askYesNo(appName, "Restore the matching original HighwayPursuit.exe from:\n\n"+backup+"?") {
		return nil
	}

	if err := atomicReplace(target, backupData, mode); err != nil {
		_ = os.WriteFile(target, backupData, mode.Perm())
		return err
	}
	finalData, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(finalData, backupData) {
		return errors.New("restore verification failed")
	}
	return nil
}
