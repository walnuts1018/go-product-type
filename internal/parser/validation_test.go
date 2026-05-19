package parser

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestCollectDeclarationsRejectsAnnotatedNonEmptyStruct(t *testing.T) {
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "sample.go", `package sample
// +adtgen:product A B
type AB struct {
	Field string
}
`, goparser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	_, err = CollectDeclarations(fset, []*ast.File{file})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "sample.go:3:6") {
		t.Fatalf("expected position-qualified error, got %q", err)
	}
	if !strings.Contains(err.Error(), "annotated declaration AB must be an empty struct") {
		t.Fatalf("unexpected error: %q", err)
	}
}

func TestCollectDeclarationsCapturesPositionAndTypeParameters(t *testing.T) {
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "sample.go", `package sample
// +adtgen:product A[T] B[U]
type Pair[T comparable, U interface{ String() string }] struct{}
`, goparser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	decls, err := CollectDeclarations(fset, []*ast.File{file})
	if err != nil {
		t.Fatal(err)
	}
	if len(decls) != 1 {
		t.Fatalf("got %d declarations, want 1", len(decls))
	}

	decl := decls[0]
	if decl.Position.Filename != "sample.go" || decl.Position.Line != 3 || decl.Position.Column != 6 {
		t.Fatalf("got position %+v, want sample.go:3:6", decl.Position)
	}
	if len(decl.TypeParameters) != 2 {
		t.Fatalf("got %d type parameters, want 2", len(decl.TypeParameters))
	}
	if decl.TypeParameters[0] != "T comparable" {
		t.Fatalf("got first type parameter %+v", decl.TypeParameters[0])
	}
	if decl.TypeParameters[1] != "U interface{ String() string }" {
		t.Fatalf("got second type parameter %+v", decl.TypeParameters[1])
	}
}

func TestCollectDeclarationsExpandsGroupedTypeParameters(t *testing.T) {
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "sample.go", `package sample
// +adtgen:product A[T] B[U]
type Pair[T, U comparable] struct{}
`, goparser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	decls, err := CollectDeclarations(fset, []*ast.File{file})
	if err != nil {
		t.Fatal(err)
	}
	if len(decls) != 1 {
		t.Fatalf("got %d declarations, want 1", len(decls))
	}

	want := []string{"T comparable", "U comparable"}
	if len(decls[0].TypeParameters) != len(want) {
		t.Fatalf("got %d type parameters, want %d", len(decls[0].TypeParameters), len(want))
	}
	for i, got := range decls[0].TypeParameters {
		if got != want[i] {
			t.Fatalf("got type parameter %d = %q, want %q", i, got, want[i])
		}
	}
}
