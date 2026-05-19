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

func TestGroupDeclarationsBySourceFilename(t *testing.T) {
	decls := []model.Declaration{
		{Name: "A", SourceFilename: "/tmp/pkg/first.go"},
		{Name: "B", SourceFilename: "/tmp/pkg/second.go"},
		{Name: "C", SourceFilename: "/tmp/pkg/first.go"},
	}

	grouped, err := groupDeclarationsBySourceFilename(decls)
	if err != nil {
		t.Fatalf("groupDeclarationsBySourceFilename() error = %v", err)
	}
	if len(grouped) != 2 {
		t.Fatalf("got %d groups, want 2", len(grouped))
	}
	if len(grouped["/tmp/pkg/first.go"]) != 2 {
		t.Fatalf("got %d declarations for first.go, want 2", len(grouped["/tmp/pkg/first.go"]))
	}
	if len(grouped["/tmp/pkg/second.go"]) != 1 {
		t.Fatalf("got %d declarations for second.go, want 1", len(grouped["/tmp/pkg/second.go"]))
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
