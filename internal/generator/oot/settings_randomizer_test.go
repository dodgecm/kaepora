//+build docker

package oot_test

import (
	"kaepora/internal/generator/factory"
	"kaepora/internal/generator/oot"
	"testing"
)

func TestOOTSettingsRandomizer(t *testing.T) {
	t.Parallel()

	f := factory.New(nil)
	g, err := f.NewGenerator(oot.SettingsRandomizerName + ":5.2.13")
	if err != nil {
		t.Fatal(err)
	}

	out, err := g.Generate("s3.json", "DEADBEEF")
	if err != nil {
		t.Fatal(err)
	}

	// Patches are not reproducible so we are limited to length checks.
	if len(out.SeedPatch) == 0 {
		t.Fatal("got an empty patch")
	}

	if len(out.SpoilerLog) == 0 {
		t.Fatal("got an empty spoiler log")
	}

	if len(out.SeedPatch) < 250*1024 {
		t.Errorf("generated patch seems too small (%d bytes)", len(out.SeedPatch))
	}
	if len(out.SeedPatch) > 350*1024 {
		t.Errorf("generated patch seems too large (%d bytes)", len(out.SeedPatch))
	}
}
