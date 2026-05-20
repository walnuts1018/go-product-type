package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/walnuts1018/go-adtgen/internal/model"
)

const productExpressionAB = "A B"

func TestCollectDeclarationsFindsProductAnnotation(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", `package sample
// +adtgen:product=A,B
type AB struct{}
`, parser.ParseComments)
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
	if decls[0].Name != "AB" {
		t.Fatalf("got %q, want AB", decls[0].Name)
	}
	if decls[0].Expression != productExpressionAB {
		t.Fatalf("got %q, want %q", decls[0].Expression, productExpressionAB)
	}
	if decls[0].Kind != model.DeclarationKindProduct {
		t.Fatalf("got kind %q, want %q", decls[0].Kind, model.DeclarationKindProduct)
	}
	if decls[0].SourceFilename != "sample.go" {
		t.Fatalf("got source filename %q, want %q", decls[0].SourceFilename, "sample.go")
	}
}

func TestCollectDeclarationsFindsSumAnnotation(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", `package sample
// +adtgen:sum=Hoge,Fuga
type HogeOrFuga interface{}
`, parser.ParseComments)
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
	if decls[0].Kind != model.DeclarationKindSum {
		t.Fatalf("got kind %q, want %q", decls[0].Kind, model.DeclarationKindSum)
	}
	if decls[0].Name != "HogeOrFuga" {
		t.Fatalf("got %q, want %q", decls[0].Name, "HogeOrFuga")
	}
	if decls[0].Expression != "Hoge Fuga" {
		t.Fatalf("got %q, want %q", decls[0].Expression, "Hoge Fuga")
	}
}

func TestCollectDeclarationsCapturesSumInterfaceMethods(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", `package sample
// +adtgen:sum=Hoge,Fuga
type HogeOrFuga interface{
	String() string
	WriteTo(io.Writer) (int64, error)
}
`, parser.ParseComments)
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
	if len(decls[0].InterfaceMethods) != 2 {
		t.Fatalf("got %d interface methods, want 2", len(decls[0].InterfaceMethods))
	}
	if decls[0].InterfaceMethods[0].Name != "String" {
		t.Fatalf("got first method name %q, want %q", decls[0].InterfaceMethods[0].Name, "String")
	}
	if decls[0].InterfaceMethods[0].Signature != "func() string" {
		t.Fatalf("got first method signature %q, want %q", decls[0].InterfaceMethods[0].Signature, "func() string")
	}
	if decls[0].InterfaceMethods[1].Name != "WriteTo" {
		t.Fatalf("got second method name %q, want %q", decls[0].InterfaceMethods[1].Name, "WriteTo")
	}
	if decls[0].InterfaceMethods[1].Signature != "func(io.Writer) (int64, error)" {
		t.Fatalf("got second method signature %q, want %q", decls[0].InterfaceMethods[1].Signature, "func(io.Writer) (int64, error)")
	}
}

func TestCollectDeclarationsFindsTypeSpecAnnotationInGroupedTypeDeclaration(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", `package sample
type (
	// +adtgen:product=A,B
	AB struct{}
)
`, parser.ParseComments)
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
	if decls[0].Name != "AB" {
		t.Fatalf("got %q, want AB", decls[0].Name)
	}
	if decls[0].Expression != productExpressionAB {
		t.Fatalf("got %q, want %q", decls[0].Expression, productExpressionAB)
	}
}

func TestCollectDeclarationsIgnoresUnannotatedDeclarationsInGroupedTypeDeclaration(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", `package sample
type (
	// +adtgen:product=A,B
	AB struct{}
	CD struct{}
)
`, parser.ParseComments)
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
	if decls[0].Name != "AB" {
		t.Fatalf("got %q, want AB", decls[0].Name)
	}
}

func TestCollectDeclarationsRejectsAnnotatedTypeAliasDeclarations(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", `package sample
// +adtgen:product=A,B
type AB = struct{}
`, parser.ParseComments)
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
	if !strings.Contains(err.Error(), "annotated declaration AB must not be a type alias") {
		t.Fatalf("unexpected error: %q", err)
	}
}

func TestCollectDeclarationsIgnoresSimilarDirectivePrefixes(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", `package sample
// +adtgen:productivity=A,B
type AB struct{}
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	decls, err := CollectDeclarations(fset, []*ast.File{file})
	if err != nil {
		t.Fatal(err)
	}
	if len(decls) != 0 {
		t.Fatalf("got %d declarations, want 0", len(decls))
	}
}

func TestCollectDeclarationsIgnoresOldDirectiveSyntax(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", `package sample
//adtgen:product=A,B
type AB struct{}
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	decls, err := CollectDeclarations(fset, []*ast.File{file})
	if err != nil {
		t.Fatal(err)
	}
	if len(decls) != 0 {
		t.Fatalf("got %d declarations, want 0", len(decls))
	}
}

func TestCollectDeclarationsParsesOptions(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", "package sample\n// +adtgen:sum=Hoge,Fuga;options=no-setter\ntype HogeOrFuga interface{}\n", parser.ParseComments)
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
	if decls[0].Expression != "Hoge Fuga" {
		t.Fatalf("got %q, want %q", decls[0].Expression, "Hoge Fuga")
	}
	if !decls[0].Options.NoSetter {
		t.Fatal("decls[0].Options.NoSetter = false, want true")
	}
}

func TestCollectDeclarationsRejectsUnknownOption(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", "package sample\n// +adtgen:sum=Hoge,Fuga;options=unknown\ntype HogeOrFuga struct{}\n", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	_, err = CollectDeclarations(fset, []*ast.File{file})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown option") {
		t.Fatalf("unexpected error: %q", err)
	}
}

func TestCollectDeclarationsRejectsNoSetterForProduct(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", "package sample\n// +adtgen:product=A,B;options=no-setter\ntype AB struct{}\n", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	_, err = CollectDeclarations(fset, []*ast.File{file})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no-setter option is only supported for sum declarations") {
		t.Fatalf("unexpected error: %q", err)
	}
}

func TestCollectFilesCapturesPassthroughCodeAndAnnotatedDeclarations(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", `package sample

import "fmt"

const label = "x"

type Inline struct{ Name string }

func UseInline(x Inline) string { return fmt.Sprint(x.Name) }

// +adtgen:product=Inline,External
type Combined struct{}
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	files, err := CollectFiles(fset, []*ast.File{file})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if len(files[0].Declarations) != 1 {
		t.Fatalf("got %d declarations, want 1", len(files[0].Declarations))
	}
	if files[0].Declarations[0].Name != "Combined" {
		t.Fatalf("got declaration %q, want %q", files[0].Declarations[0].Name, "Combined")
	}
	if len(files[0].PassthroughImports) != 1 {
		t.Fatalf("got %d passthrough imports, want 1", len(files[0].PassthroughImports))
	}
	if files[0].PassthroughImports[0].Path != "fmt" {
		t.Fatalf("got import path %q, want %q", files[0].PassthroughImports[0].Path, "fmt")
	}
	got := strings.Join(files[0].PassthroughDecls, "\n")
	for _, want := range []string{
		"const label = \"x\"",
		"type Inline struct{",
		"func UseInline(x Inline) string",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("passthrough declarations missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "type Combined struct{}") {
		t.Fatalf("passthrough declarations unexpectedly contain annotated declaration:\n%s", got)
	}
}

func TestCollectFilesKeepsUnannotatedSpecsInGroupedTypeDeclaration(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", `package sample

type (
	Kept struct{ Name string }
	// +adtgen:product=A,B
	Generated struct{}
)
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	files, err := CollectFiles(fset, []*ast.File{file})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if len(files[0].PassthroughDecls) != 1 {
		t.Fatalf("got %d passthrough decls, want 1", len(files[0].PassthroughDecls))
	}
	if !strings.Contains(files[0].PassthroughDecls[0], "Kept struct") {
		t.Fatalf("passthrough decl missing kept type:\n%s", files[0].PassthroughDecls[0])
	}
	if strings.Contains(files[0].PassthroughDecls[0], "Generated struct") {
		t.Fatalf("passthrough decl unexpectedly contains annotated type:\n%s", files[0].PassthroughDecls[0])
	}
}
