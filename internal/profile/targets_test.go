package profile

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestTargetLookup(t *testing.T) {
	t.Parallel()

	registry := Registry{}

	for _, name := range []string{"mp4", "MP4", "  mp3  "} {
		found, ok := registry.Target(name)
		if !ok {
			t.Errorf("Target(%q) not found", name)
			continue
		}
		if found.Name != strings.ToLower(strings.TrimSpace(name)) {
			t.Errorf("Target(%q).Name = %q", name, found.Name)
		}
		if found.DefaultPreset == "" {
			t.Errorf("Target(%q) has no default preset", name)
		}
	}

	if _, ok := registry.Target("avi"); ok {
		t.Error("Target(\"avi\") was found, but avi is not an output format")
	}
}

func TestEveryTargetCanBePlanned(t *testing.T) {
	t.Parallel()

	// A target the registry advertises but cannot plan for would surface as a
	// confusing "unsupported preset" error rather than an honest refusal.
	for _, target := range (Registry{}).Targets() {
		if target.DefaultPreset == "" {
			t.Errorf("target %q advertises no default preset", target.Name)
		}
		if defaultPreset(target.Name) != target.DefaultPreset {
			t.Errorf("defaultPreset(%q) = %q, want %q", target.Name, defaultPreset(target.Name), target.DefaultPreset)
		}
		if len(target.BatchExtensions) == 0 {
			t.Errorf("target %q has no batch input extensions", target.Name)
		}
	}
}

func TestBatchExtensionsNeverIncludeTheirOwnTarget(t *testing.T) {
	t.Parallel()

	// Including ".mp4" among the inputs for --to mp4 would make a batch try to
	// convert every output onto itself.
	for _, target := range (Registry{}).Targets() {
		own := "." + target.Name
		if slices.Contains(target.BatchExtensions, own) {
			t.Errorf("target %q lists %q as a batch input", target.Name, own)
		}
		for _, extension := range target.BatchExtensions {
			if !strings.HasPrefix(extension, ".") {
				t.Errorf("target %q has batch extension %q without a leading dot", target.Name, extension)
			}
			if extension != strings.ToLower(extension) {
				t.Errorf("target %q has batch extension %q that is not lowercase", target.Name, extension)
			}
		}
	}
}

func TestTargetNamesMatchTargets(t *testing.T) {
	t.Parallel()

	registry := Registry{}
	names := registry.TargetNames()
	targets := registry.Targets()

	if len(names) != len(targets) {
		t.Fatalf("TargetNames() has %d entries, Targets() has %d", len(names), len(targets))
	}
	for i, target := range targets {
		if names[i] != target.Name {
			t.Errorf("name %d = %q, want %q", i, names[i], target.Name)
		}
	}
}

func TestTargetsIsACopy(t *testing.T) {
	t.Parallel()

	// Callers outside the package must not be able to edit the registry.
	first := (Registry{}).Targets()
	first[0].Name = "mutated"

	if (Registry{}).Targets()[0].Name == "mutated" {
		t.Error("Targets() handed out the backing array, so a caller can rewrite the registry")
	}
}

func TestEveryAdvertisedFormatHasATarget(t *testing.T) {
	t.Parallel()

	// `mediaconv formats` must not advertise a conversion that --to rejects.
	registry := Registry{}
	for _, format := range registry.Formats() {
		if _, ok := registry.Target(format.Target); !ok {
			t.Errorf("formats lists %s -> %s, but %q is not a target", format.Source, format.Target, format.Target)
		}
		if format.Source == "" || format.Profile == "" {
			t.Errorf("format %+v is missing a source or profile", format)
		}
	}
}

func TestBatchExtensionsCoverTheAdvertisedSources(t *testing.T) {
	t.Parallel()

	// Every source `formats` advertises should be picked up by a batch for that
	// target, except the target's own extension, which is excluded on purpose.
	registry := Registry{}
	byTarget := make(map[string][]string)
	for _, format := range registry.Formats() {
		byTarget[format.Target] = append(byTarget[format.Target], "."+format.Source)
	}

	for _, target := range registry.Targets() {
		for _, source := range byTarget[target.Name] {
			if source == "."+target.Name {
				continue
			}
			if !slices.Contains(target.BatchExtensions, source) {
				t.Errorf("target %q advertises source %q but a batch would skip it", target.Name, source)
			}
		}
	}
}

func TestSupportedVideoSourceMatchesExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path    string
		formats []string
		want    string
	}{
		{path: filepath.Join("a", "clip.webm"), formats: []string{"webm"}, want: "webm"},
		{path: filepath.Join("a", "clip.mkv"), formats: []string{"matroska", "webm"}, want: "mkv"},
		{path: filepath.Join("a", "clip.mov"), formats: []string{"mov", "mp4"}, want: "mov"},
	}

	for _, test := range tests {
		got, ok := supportedVideoSource(test.path, test.formats)
		if !ok {
			t.Errorf("supportedVideoSource(%q) not supported", test.path)
			continue
		}
		if got != test.want {
			t.Errorf("supportedVideoSource(%q) = %q, want %q", test.path, got, test.want)
		}
	}
}
