package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/walnuts1018/go-adtgen/internal/composer"
	"github.com/walnuts1018/go-adtgen/internal/emitter"
	"github.com/walnuts1018/go-adtgen/internal/loader"
	"github.com/walnuts1018/go-adtgen/internal/model"
	"github.com/walnuts1018/go-adtgen/internal/parser"
	"github.com/walnuts1018/go-adtgen/internal/resolver"
	"github.com/walnuts1018/go-adtgen/internal/writer"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	pattern := "."
	if len(args) > 0 {
		pattern = args[0]
	}

	pkg, err := loader.LoadGeneratePackage(loader.Config{Pattern: pattern})
	if err != nil {
		return err
	}

	declarations, err := parser.CollectDeclarations(pkg.Fset, pkg.SyntaxFiles())
	if err != nil {
		return err
	}
	if len(declarations) == 0 {
		return fmt.Errorf("generator: no declarations found")
	}

	grouped, err := groupDeclarationsBySourceFilename(declarations)
	if err != nil {
		return err
	}

	filenames := sortedSourceFilenames(grouped)
	if err := removeLegacyGeneratedFile(filenames[0]); err != nil {
		return err
	}

	for _, filename := range filenames {
		resolved, err := resolver.ResolveDeclarations(pkg, grouped[filename])
		if err != nil {
			return err
		}
		ordered, err := composer.OrderDeclarations(resolved)
		if err != nil {
			return err
		}

		generated := make([]model.GeneratedType, 0, len(ordered))
		for _, declaration := range ordered {
			generatedType, err := composer.BuildGeneratedType(declaration)
			if err != nil {
				return err
			}
			generated = append(generated, generatedType)
		}

		src, err := emitter.RenderForPackage(pkg.Package.PkgPath, pkg.Package.Name, generated)
		if err != nil {
			return err
		}

		output, err := outputPathFromSourceFilename(filename)
		if err != nil {
			return err
		}
		if err := writer.WriteFile(output, src); err != nil {
			return err
		}
	}

	return nil
}

func outputPathFromSourceFilename(filename string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("generator: source filename is required")
	}
	if filepath.Ext(filename) != ".go" {
		return "", fmt.Errorf("generator: source filename must end in .go: %s", filename)
	}

	dir := filepath.Dir(filename)
	base := strings.TrimSuffix(filepath.Base(filename), ".go")
	return filepath.Join(dir, base+"_adtgen.go"), nil
}

func groupDeclarationsBySourceFilename(declarations []model.Declaration) (map[string][]model.Declaration, error) {
	grouped := make(map[string][]model.Declaration)
	for _, declaration := range declarations {
		if declaration.SourceFilename == "" {
			return nil, fmt.Errorf("generator: declaration %s has no source filename", declaration.Name)
		}
		grouped[declaration.SourceFilename] = append(grouped[declaration.SourceFilename], declaration)
	}
	return grouped, nil
}

func sortedSourceFilenames(grouped map[string][]model.Declaration) []string {
	filenames := make([]string, 0, len(grouped))
	for filename := range grouped {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)
	return filenames
}

func removeLegacyGeneratedFile(sourceFilename string) error {
	if sourceFilename == "" {
		return fmt.Errorf("generator: source filename is required")
	}

	legacyFilename := filepath.Join(filepath.Dir(sourceFilename), "zz_generated.adtgen.go")
	if err := os.Remove(legacyFilename); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
