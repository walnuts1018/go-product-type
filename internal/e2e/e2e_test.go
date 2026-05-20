package e2e

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoGenerateProducesExpectedOutput(t *testing.T) {
	dir := prepareFixture(t, "base")
	outputPath := filepath.Join(dir, "generate_types_adtgen.go")

	cmd := exec.Command("go", "generate", ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go generate failed: %v\n%s", err, out)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("os.ReadFile(got) error = %v", err)
	}
	want, err := os.ReadFile(filepath.Join(dir, "expected.txt"))
	if err != nil {
		t.Fatalf("os.ReadFile(want) error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("generated output mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestGeneratedFixtureBuilds(t *testing.T) {
	dir := prepareFixture(t, "base")
	cmd := exec.Command("go", "test", ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture build failed: %v\n%s", err, out)
	}
}

func TestGoGenerateBuildsAndExercisesSumFixture(t *testing.T) {
	dir := prepareFixture(t, "sum")
	outputPath := filepath.Join(dir, "generate_types_adtgen.go")

	cmd := exec.Command("go", "generate", ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go generate failed: %v\n%s", err, out)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("os.ReadFile(got) error = %v", err)
	}
	if !bytes.Contains(got, []byte("String() string")) {
		t.Fatalf("sum output missing custom interface method:\n%s", got)
	}
	if bytes.Contains(got, []byte("func (x *Hoge) String() string")) {
		t.Fatalf("sum output unexpectedly generated Hoge.String:\n%s", got)
	}
	if bytes.Contains(got, []byte("func (x *Fuga) String() string")) {
		t.Fatalf("sum output unexpectedly generated Fuga.String:\n%s", got)
	}

	cmd = exec.Command("go", "test", ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sum fixture test failed: %v\n%s", err, out)
	}
}

func TestGoGenerateSeparatesOutputsPerSourceFile(t *testing.T) {
	dir := prepareFixture(t, "multi")
	alphaOutput := filepath.Join(dir, "generate_alpha_adtgen.go")
	betaOutput := filepath.Join(dir, "generate_beta_adtgen.go")

	cmd := exec.Command("go", "generate", ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go generate failed: %v\n%s", err, out)
	}

	alpha, err := os.ReadFile(alphaOutput)
	if err != nil {
		t.Fatalf("os.ReadFile(alpha) error = %v", err)
	}
	beta, err := os.ReadFile(betaOutput)
	if err != nil {
		t.Fatalf("os.ReadFile(beta) error = %v", err)
	}

	if !bytes.Contains(alpha, []byte("type Alpha interface")) {
		t.Fatalf("alpha output missing Alpha interface:\n%s", alpha)
	}
	if !bytes.Contains(alpha, []byte("func MatchAlpha")) {
		t.Fatalf("alpha output missing MatchAlpha:\n%s", alpha)
	}
	if !bytes.Contains(beta, []byte("type Beta struct")) {
		t.Fatalf("beta output missing Beta struct:\n%s", beta)
	}
	if !bytes.Contains(beta, []byte("func NewBeta")) {
		t.Fatalf("beta output missing NewBeta:\n%s", beta)
	}
	if bytes.Contains(alpha, []byte("type Beta")) {
		t.Fatalf("alpha output unexpectedly contains beta declaration:\n%s", alpha)
	}
	if bytes.Contains(beta, []byte("type Alpha")) {
		t.Fatalf("beta output unexpectedly contains alpha declaration:\n%s", beta)
	}

	cmd = exec.Command("go", "test", ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("multi fixture test failed: %v\n%s", err, out)
	}
}

func TestGoGeneratePreservesPassthroughCodeAndAllowsInlineTypesInAnnotations(t *testing.T) {
	dir := prepareFixture(t, "passthrough")
	outputPath := filepath.Join(dir, "generate_types_adtgen.go")

	cmd := exec.Command("go", "generate", ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go generate failed: %v\n%s", err, out)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("os.ReadFile(got) error = %v", err)
	}
	for _, want := range []string{
		"import \"fmt\"",
		"const InlinePrefix = \"inline\"",
		"var DefaultInline = Inline{Label: InlinePrefix}",
		"type Inline struct {",
		"func FormatInline(x Inline) string",
		"type Combined struct {",
		"Label string",
		"Count int",
	} {
		if !bytes.Contains(got, []byte(want)) {
			t.Fatalf("generated output missing %q:\n%s", want, got)
		}
	}

	cmd = exec.Command("go", "test", ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("passthrough fixture test failed: %v\n%s", err, out)
	}
}

func prepareFixture(t *testing.T, name string) string {
	t.Helper()

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("filepath.Abs(repo root) error = %v", err)
	}
	tempRoot, err := os.MkdirTemp(repoRoot, "e2e-fixture-")
	if err != nil {
		t.Fatalf("os.MkdirTemp() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(tempRoot)
	})

	srcDir := filepath.Join("..", "testdata", "fixtures", "e2e", name)
	dstDir := filepath.Join(tempRoot, name)
	copyDir(t, srcDir, dstDir)
	rewriteGenerateDirectives(t, dstDir, repoRoot)
	return dstDir
}

func copyDir(t *testing.T, srcDir, dstDir string) {
	t.Helper()

	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", dstDir, err)
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) error = %v", srcDir, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())
		if entry.IsDir() {
			copyDir(t, srcPath, dstPath)
			continue
		}
		copyFile(t, srcPath, dstPath)
	}
}

func copyFile(t *testing.T, srcPath, dstPath string) {
	t.Helper()

	src, err := os.Open(srcPath)
	if err != nil {
		t.Fatalf("os.Open(%q) error = %v", srcPath, err)
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		t.Fatalf("os.Create(%q) error = %v", dstPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		t.Fatalf("io.Copy(%q, %q) error = %v", dstPath, srcPath, err)
	}
}

func rewriteGenerateDirectives(t *testing.T, dir, repoRoot string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) error = %v", dir, err)
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			rewriteGenerateDirectives(t, path, repoRoot)
			continue
		}
		if filepath.Ext(path) != ".go" {
			continue
		}

		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("os.ReadFile(%q) error = %v", path, err)
		}

		content := string(src)
		content = strings.ReplaceAll(content, "../../../../../main.go", filepath.Join(repoRoot, "main.go"))
		content = strings.ReplaceAll(content, "../../../../../.", repoRoot)
		if content == string(src) {
			continue
		}

		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", path, err)
		}
	}
}
