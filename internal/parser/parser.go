package parser

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"strconv"
	"strings"

	"github.com/walnuts1018/go-adtgen/internal/model"
)

const (
	productDirective = "+adtgen:product"
	sumDirective     = "+adtgen:sum"
)

func CollectDeclarations(fset *token.FileSet, files []*ast.File) ([]model.Declaration, error) {
	collected, err := CollectFiles(fset, files)
	if err != nil {
		return nil, err
	}

	var declarations []model.Declaration
	for _, file := range collected {
		declarations = append(declarations, file.Declarations...)
	}
	return declarations, nil
}

func CollectFiles(fset *token.FileSet, files []*ast.File) ([]model.SourceFile, error) {
	collected := make([]model.SourceFile, 0, len(files))
	for _, file := range files {
		sourceFile := model.SourceFile{
			SourceFilename: fset.Position(file.Pos()).Filename,
		}

		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				sourceFile.PassthroughDecls = append(sourceFile.PassthroughDecls, renderNode(fset, decl))
				continue
			}
			switch genDecl.Tok {
			case token.IMPORT:
				imports, err := collectPassthroughImports(genDecl)
				if err != nil {
					pos := fset.Position(genDecl.Pos())
					return nil, fmt.Errorf("%s: %w", pos, err)
				}
				sourceFile.PassthroughImports = append(sourceFile.PassthroughImports, imports...)
			case token.TYPE:
				passthroughDecl, declarations, err := collectTypeDecl(fset, genDecl)
				if err != nil {
					return nil, err
				}
				sourceFile.Declarations = append(sourceFile.Declarations, declarations...)
				if passthroughDecl != "" {
					sourceFile.PassthroughDecls = append(sourceFile.PassthroughDecls, passthroughDecl)
				}
			default:
				sourceFile.PassthroughDecls = append(sourceFile.PassthroughDecls, renderNode(fset, decl))
			}
		}

		if len(sourceFile.Declarations) == 0 && len(sourceFile.PassthroughImports) == 0 && len(sourceFile.PassthroughDecls) == 0 {
			continue
		}
		collected = append(collected, sourceFile)
	}

	return collected, nil
}

func collectTypeDecl(fset *token.FileSet, genDecl *ast.GenDecl) (string, []model.Declaration, error) {
	var declarations []model.Declaration
	passthroughSpecs := make([]ast.Spec, 0, len(genDecl.Specs))

	for _, spec := range genDecl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		kind, expression, options, annotated, err := declarationSpecForTypeSpec(genDecl, typeSpec)
		if err != nil {
			pos := fset.Position(typeSpec.Pos())
			return "", nil, fmt.Errorf("%s: %w", pos, err)
		}
		if !annotated {
			passthroughSpecs = append(passthroughSpecs, spec)
			continue
		}
		if typeSpec.Assign.IsValid() {
			pos := fset.Position(typeSpec.Pos())
			return "", nil, fmt.Errorf("%s: annotated declaration %s must not be a type alias", pos, typeSpec.Name.Name)
		}
		interfaceMethods, err := validateDeclarationShape(fset, typeSpec, kind)
		if err != nil {
			pos := fset.Position(typeSpec.Pos())
			return "", nil, fmt.Errorf("%s: %w", pos, err)
		}

		position := fset.Position(typeSpec.Pos())
		declarations = append(declarations, model.Declaration{
			Kind:             kind,
			Name:             typeSpec.Name.Name,
			Expression:       expression,
			Options:          options,
			TypeParameters:   collectTypeParameters(fset, typeSpec.TypeParams),
			InterfaceMethods: interfaceMethods,
			Position:         position,
			SourceFilename:   position.Filename,
		})
	}

	if len(passthroughSpecs) == 0 {
		return "", declarations, nil
	}

	filtered := *genDecl
	filtered.Specs = passthroughSpecs
	return renderNode(fset, &filtered), declarations, nil
}

func collectPassthroughImports(genDecl *ast.GenDecl) ([]model.PassthroughImport, error) {
	imports := make([]model.PassthroughImport, 0, len(genDecl.Specs))
	for _, spec := range genDecl.Specs {
		importSpec, ok := spec.(*ast.ImportSpec)
		if !ok {
			continue
		}
		path, err := strconv.Unquote(importSpec.Path.Value)
		if err != nil {
			return nil, err
		}
		imp := model.PassthroughImport{Path: path}
		if importSpec.Name != nil {
			imp.Name = importSpec.Name.Name
		}
		imports = append(imports, imp)
	}
	return imports, nil
}

func declarationSpecForTypeSpec(genDecl *ast.GenDecl, typeSpec *ast.TypeSpec) (model.DeclarationKind, string, model.DeclarationOptions, bool, error) {
	if kind, expression, options, ok, err := findDeclarationSpec(typeSpec.Doc); ok || err != nil {
		return kind, expression, options, ok, err
	}
	if kind, expression, options, ok, err := findDeclarationSpec(typeSpec.Comment); ok || err != nil {
		return kind, expression, options, ok, err
	}
	if len(genDecl.Specs) == 1 {
		return findDeclarationSpec(genDecl.Doc)
	}
	return "", "", model.DeclarationOptions{}, false, nil
}

func findDeclarationSpec(group *ast.CommentGroup) (model.DeclarationKind, string, model.DeclarationOptions, bool, error) {
	if group == nil {
		return "", "", model.DeclarationOptions{}, false, nil
	}

	for _, comment := range group.List {
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		if text == "" {
			continue
		}

		directive := ""
		var kind model.DeclarationKind
		switch {
		case strings.HasPrefix(text, productDirective):
			if len(text) > len(productDirective) && text[len(productDirective)] != '=' {
				continue
			}
			directive = productDirective
			kind = model.DeclarationKindProduct
		case strings.HasPrefix(text, sumDirective):
			if len(text) > len(sumDirective) && text[len(sumDirective)] != '=' {
				continue
			}
			directive = sumDirective
			kind = model.DeclarationKindSum
		default:
			continue
		}
		expression, options, err := parseDirectiveSpec(kind, strings.TrimSpace(strings.TrimPrefix(text, directive)))
		if err != nil {
			return "", "", model.DeclarationOptions{}, false, err
		}
		return kind, expression, options, true, nil
	}

	return "", "", model.DeclarationOptions{}, false, nil
}

func parseDirectiveSpec(kind model.DeclarationKind, spec string) (string, model.DeclarationOptions, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", model.DeclarationOptions{}, fmt.Errorf("missing directive body")
	}

	segments := strings.Split(spec, ";")
	expressionSegment := strings.TrimSpace(segments[0])
	expression, err := parseExpressionSegment(expressionSegment)
	if err != nil {
		return "", model.DeclarationOptions{}, err
	}

	var options model.DeclarationOptions
	for _, segment := range segments[1:] {
		key, value, ok := strings.Cut(strings.TrimSpace(segment), "=")
		if !ok {
			return "", model.DeclarationOptions{}, fmt.Errorf("malformed directive segment %q", segment)
		}
		if key != "options" {
			return "", model.DeclarationOptions{}, fmt.Errorf("unknown directive key %q", key)
		}
		parsed, err := parseDeclarationOptions(value)
		if err != nil {
			return "", model.DeclarationOptions{}, err
		}
		options.NoSetter = options.NoSetter || parsed.NoSetter
	}

	if options.NoSetter && kind != model.DeclarationKindSum {
		return "", model.DeclarationOptions{}, fmt.Errorf("no-setter option is only supported for sum declarations")
	}

	return expression, options, nil
}

func parseExpressionSegment(segment string) (string, error) {
	_, value, ok := strings.Cut(segment, "=")
	if !ok {
		return "", fmt.Errorf("missing directive expression")
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("missing directive expression")
	}

	inputs := strings.Split(value, ",")
	parts := make([]string, 0, len(inputs))
	for _, input := range inputs {
		part := strings.TrimSpace(input)
		if part == "" {
			return "", fmt.Errorf("empty directive input")
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, " "), nil
}

func parseDeclarationOptions(value string) (model.DeclarationOptions, error) {
	var options model.DeclarationOptions
	for _, raw := range strings.Split(value, ",") {
		option := strings.TrimSpace(raw)
		if option == "" {
			return model.DeclarationOptions{}, fmt.Errorf("empty option")
		}
		switch option {
		case "no-setter":
			options.NoSetter = true
		default:
			return model.DeclarationOptions{}, fmt.Errorf("unknown option %q", option)
		}
	}
	return options, nil
}

func collectTypeParameters(fset *token.FileSet, fieldList *ast.FieldList) []string {
	if fieldList == nil {
		return nil
	}

	params := make([]string, 0, len(fieldList.List))
	for _, field := range fieldList.List {
		constraint := renderNode(fset, field.Type)
		for _, name := range field.Names {
			param := name.Name
			if constraint != "" {
				param += " " + constraint
			}
			params = append(params, param)
		}
	}

	return params
}

func validateDeclarationShape(fset *token.FileSet, typeSpec *ast.TypeSpec, kind model.DeclarationKind) ([]model.DeclaredInterfaceMethod, error) {
	if kind == model.DeclarationKindSum {
		return collectDeclaredInterfaceMethods(fset, typeSpec)
	}

	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok || structType.Fields == nil || len(structType.Fields.List) != 0 {
		return nil, fmt.Errorf("annotated declaration %s must be an empty struct", typeSpec.Name.Name)
	}
	return nil, nil
}

func collectDeclaredInterfaceMethods(fset *token.FileSet, typeSpec *ast.TypeSpec) ([]model.DeclaredInterfaceMethod, error) {
	interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
	if !ok {
		return nil, fmt.Errorf("sum declaration %s must be an interface", typeSpec.Name.Name)
	}
	if interfaceType.Methods == nil || len(interfaceType.Methods.List) == 0 {
		return nil, nil
	}

	methods := make([]model.DeclaredInterfaceMethod, 0, len(interfaceType.Methods.List))
	for _, field := range interfaceType.Methods.List {
		if len(field.Names) != 1 {
			return nil, fmt.Errorf("sum declaration %s must only declare methods", typeSpec.Name.Name)
		}
		funcType, ok := field.Type.(*ast.FuncType)
		if !ok {
			return nil, fmt.Errorf("sum declaration %s must only declare methods", typeSpec.Name.Name)
		}
		methods = append(methods, model.DeclaredInterfaceMethod{
			Name:      field.Names[0].Name,
			Signature: renderNode(fset, funcType),
		})
	}
	return methods, nil
}

func renderNode(fset *token.FileSet, node ast.Node) string {
	if node == nil {
		return ""
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return ""
	}

	return buf.String()
}
