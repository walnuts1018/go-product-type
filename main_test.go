package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/walnuts1018/go-adtgen/internal/model"
)

func TestOutputPathFromSourceFilename(t *testing.T) {
	filename := filepath.Join("tmp", "example", "generate_types.go")
	got, err := outputPathFromSourceFilename(filename)
	if err != nil {
		t.Fatalf("outputPathFromSourceFilename() error = %v", err)
	}
	want := filepath.Join("tmp", "example", "generate_types_adtgen.go")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestOutputPathFromSourceFilenameRejectsNonGoFiles(t *testing.T) {
	filename := filepath.Join("tmp", "example", "generate_types.txt")
	_, err := outputPathFromSourceFilename(filename)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSortedSourceFilenames(t *testing.T) {
	files := []model.SourceFile{
		{SourceFilename: "/tmp/pkg/second.go", Declarations: []model.Declaration{{Name: "B"}}},
		{SourceFilename: "/tmp/pkg/ignored.go"},
		{SourceFilename: "/tmp/pkg/first.go", Declarations: []model.Declaration{{Name: "A"}}},
	}

	got := sortedSourceFilenames(files)
	if len(got) != 2 {
		t.Fatalf("got %d filenames, want 2", len(got))
	}
	if got[0] != "/tmp/pkg/first.go" {
		t.Fatalf("got first filename %q, want %q", got[0], "/tmp/pkg/first.go")
	}
	if got[1] != "/tmp/pkg/second.go" {
		t.Fatalf("got second filename %q, want %q", got[1], "/tmp/pkg/second.go")
	}
}

func TestRemoveLegacyGeneratedFileRemovesLegacyOutput(t *testing.T) {
	dir := t.TempDir()
	sourceFilename := filepath.Join(dir, "first.go")
	legacyFilename := filepath.Join(dir, "zz_generated.adtgen.go")
	if err := os.WriteFile(legacyFilename, []byte("legacy"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	if err := removeLegacyGeneratedFile(sourceFilename); err != nil {
		t.Fatalf("removeLegacyGeneratedFile() error = %v", err)
	}

	if _, err := os.Stat(legacyFilename); !os.IsNotExist(err) {
		t.Fatalf("legacy file still exists, stat error = %v", err)
	}
}
