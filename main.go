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

	files, err := parser.CollectFiles(pkg.Fset, pkg.SyntaxFiles())
	if err != nil {
		return err
	}
	declarations := make([]model.Declaration, 0)
	for _, file := range files {
		declarations = append(declarations, file.Declarations...)
	}
	if len(declarations) == 0 {
		return fmt.Errorf("generator: no declarations found")
	}

	filenames := sortedSourceFilenames(files)
	if err := removeLegacyGeneratedFile(filenames[0]); err != nil {
		return err
	}

	for _, file := range files {
		if len(file.Declarations) == 0 {
			continue
		}

		resolved, err := resolver.ResolveDeclarations(pkg, file.Declarations)
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

		src, err := emitter.RenderFile(model.GeneratedFile{
			PackagePath:      pkg.Package.PkgPath,
			PackageName:      pkg.Package.Name,
			Imports:          file.PassthroughImports,
			PassthroughDecls: file.PassthroughDecls,
			Generated:        generated,
		})
		if err != nil {
			return err
		}

		output, err := outputPathFromSourceFilename(file.SourceFilename)
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

func sortedSourceFilenames(files []model.SourceFile) []string {
	filenames := make([]string, 0, len(files))
	for _, file := range files {
		if file.SourceFilename == "" || len(file.Declarations) == 0 {
			continue
		}
		filenames = append(filenames, file.SourceFilename)
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
