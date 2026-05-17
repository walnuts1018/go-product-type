package composer

import (
	"fmt"
	"go/types"
	"sort"
	"strings"
	"unicode"

	"github.com/walnuts1018/go-adtgen/internal/model"
)

type FieldSpec struct {
	Name      string
	Type      types.Type
	Tag       string
	Anonymous bool
}

func ExtractFields(st *types.Struct) []FieldSpec {
	if st == nil {
		return nil
	}

	fields := make([]FieldSpec, 0, st.NumFields())
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		fields = append(fields, FieldSpec{
			Name:      field.Name(),
			Type:      field.Type(),
			Tag:       st.Tag(i),
			Anonymous: field.Embedded(),
		})
	}
	return fields
}

func ComposeFields(groups [][]FieldSpec) ([]FieldSpec, error) {
	var composed []FieldSpec

	for _, group := range groups {
		for _, field := range group {
			index := findComposedField(composed, field)
			if index < 0 {
				composed = append(composed, field)
				continue
			}

			existing := composed[index]
			if existing.Anonymous != field.Anonymous {
				return nil, fmt.Errorf("composer: conflicting field %s", field.Name)
			}
			if field.Anonymous {
				// Identical anonymous fields merge by effective name/type, and the
				// first encountered field metadata (for example tags) is preserved.
				if !types.Identical(existing.Type, field.Type) {
					return nil, fmt.Errorf("composer: conflicting field %s", field.Name)
				}
				continue
			}

			if !types.Identical(existing.Type, field.Type) {
				return nil, fmt.Errorf("composer: conflicting field %s", field.Name)
			}
			if existing.Tag != field.Tag {
				return nil, fmt.Errorf("composer: conflicting tag for field %s", field.Name)
			}
		}
	}

	return composed, nil
}

func BuildGeneratedType(declaration model.ResolvedDeclaration) (model.GeneratedType, error) {
	if declarationKind(declaration.Declaration) == model.DeclarationKindSum {
		return buildSumGeneratedType(declaration)
	}

	return buildProductGeneratedType(declaration)
}

func buildProductGeneratedType(declaration model.ResolvedDeclaration) (model.GeneratedType, error) {
	groups := make([][]FieldSpec, 0, len(declaration.Inputs))
	inputs := make([]model.GeneratedInput, 0, len(declaration.Inputs))
	nameCounts := countBaseInputNames(declaration.Inputs)
	for _, input := range declaration.Inputs {
		fields := ExtractFields(input.Struct)
		groups = append(groups, fields)
		inputs = append(inputs, model.GeneratedInput{
			Expression:    input.Expr,
			ParameterName: constructorParameterName(input.Expr, nameCounts),
			Type:          input.Type,
			MethodName:    splitMethodName(input.Expr, nameCounts),
			FieldNames:    fieldNames(fields),
		})
	}

	fields, err := ComposeFields(groups)
	if err != nil {
		return model.GeneratedType{}, err
	}

	generated := model.GeneratedType{
		Kind:           model.DeclarationKindProduct,
		Name:           declaration.Declaration.Name,
		TypeParameters: append([]string(nil), declaration.Declaration.TypeParameters...),
		Inputs:         inputs,
		Fields:         make([]model.GeneratedField, 0, len(fields)),
	}
	for _, field := range fields {
		generated.Fields = append(generated.Fields, model.GeneratedField{
			Name:      field.Name,
			Type:      field.Type,
			Tag:       field.Tag,
			Anonymous: field.Anonymous,
		})
	}

	return generated, nil
}

func buildSumGeneratedType(declaration model.ResolvedDeclaration) (model.GeneratedType, error) {
	variants := make([]model.GeneratedSumVariant, 0, len(declaration.Inputs))
	fieldSets := make([]map[string]accessibleField, 0, len(declaration.Inputs))
	fieldOrders := make([][]string, 0, len(declaration.Inputs))

	for _, input := range declaration.Inputs {
		variants = append(variants, model.GeneratedSumVariant{
			Expression: input.Expr,
			Type:       input.Type,
			TypeName:   inputTypeName(input.Type),
		})

		fields, order, err := collectAccessibleFields(input.Type, input.Struct)
		if err != nil {
			return model.GeneratedType{}, err
		}
		fieldSets = append(fieldSets, fields)
		fieldOrders = append(fieldOrders, order)
	}

	commonFields, err := buildCommonFields(variants, fieldSets, fieldOrders)
	if err != nil {
		return model.GeneratedType{}, err
	}

	return model.GeneratedType{
		Kind:           model.DeclarationKindSum,
		Name:           declaration.Declaration.Name,
		TypeParameters: append([]string(nil), declaration.Declaration.TypeParameters...),
		Sum: &model.GeneratedSum{
			Variants:     variants,
			CommonFields: commonFields,
		},
	}, nil
}

func findComposedField(fields []FieldSpec, target FieldSpec) int {
	for i, field := range fields {
		if field.Name == target.Name {
			return i
		}
	}
	return -1
}

func declarationKind(declaration model.Declaration) model.DeclarationKind {
	if declaration.Kind == "" {
		return model.DeclarationKindProduct
	}
	return declaration.Kind
}

type accessibleField struct {
	Name string
	Type types.Type
	Path []string
}

type structLevel struct {
	Struct *types.Struct
	Path   []string
}

func collectAccessibleFields(typ types.Type, st *types.Struct) (map[string]accessibleField, []string, error) {
	if st == nil {
		return nil, nil, nil
	}

	fields := make(map[string]accessibleField)
	order := make([]string, 0)
	current := []structLevel{{Struct: st}}

	for len(current) > 0 {
		next := make([]structLevel, 0)
		levelCandidates := make(map[string][]accessibleField)

		for _, level := range current {
			for i := 0; i < level.Struct.NumFields(); i++ {
				field := level.Struct.Field(i)
				path := appendPath(level.Path, field.Name())
				if !field.Embedded() {
					levelCandidates[field.Name()] = append(levelCandidates[field.Name()], accessibleField{
						Name: field.Name(),
						Type: field.Type(),
						Path: path,
					})
				}

				if !field.Embedded() {
					continue
				}
				embeddedStruct, ok := embeddedStructType(field.Type())
				if !ok {
					continue
				}
				next = append(next, structLevel{
					Struct: embeddedStruct,
					Path:   path,
				})
			}
		}

		names := make([]string, 0, len(levelCandidates))
		for name := range levelCandidates {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			if _, seen := fields[name]; seen {
				continue
			}
			candidates := levelCandidates[name]
			if len(candidates) > 1 {
				return nil, nil, fmt.Errorf("composer: ambiguous promoted field %s in %s", name, inputTypeName(typ))
			}
			fields[name] = candidates[0]
			order = append(order, name)
		}

		current = next
	}

	return fields, order, nil
}

func embeddedStructType(typ types.Type) (*types.Struct, bool) {
	switch t := types.Unalias(typ).(type) {
	case *types.Named:
		st, ok := t.Underlying().(*types.Struct)
		return st, ok
	case *types.Pointer:
		named, ok := types.Unalias(t.Elem()).(*types.Named)
		if !ok {
			return nil, false
		}
		st, ok := named.Underlying().(*types.Struct)
		return st, ok
	default:
		return nil, false
	}
}

func buildCommonFields(variants []model.GeneratedSumVariant, fieldSets []map[string]accessibleField, fieldOrders [][]string) ([]model.GeneratedCommonField, error) {
	if len(variants) == 0 {
		return nil, nil
	}

	common := make([]model.GeneratedCommonField, 0)
	for _, name := range fieldOrders[0] {
		first := fieldSets[0][name]
		paths := make([][]string, 0, len(fieldSets))
		paths = append(paths, append([]string(nil), first.Path...))

		include := true
		for i := 1; i < len(fieldSets); i++ {
			field, ok := fieldSets[i][name]
			if !ok {
				include = false
				break
			}
			if !types.Identical(first.Type, field.Type) {
				return nil, fmt.Errorf("composer: conflicting common field %s", name)
			}
			paths = append(paths, append([]string(nil), field.Path...))
		}
		if !include {
			continue
		}

		common = append(common, model.GeneratedCommonField{
			Name:       name,
			Type:       first.Type,
			GetterName: "Get" + name,
			SetterName: "Set" + name,
			Paths:      paths,
		})
	}

	return common, nil
}

func appendPath(base []string, name string) []string {
	path := make([]string, 0, len(base)+1)
	path = append(path, base...)
	path = append(path, name)
	return path
}

func inputTypeName(typ types.Type) string {
	named, ok := types.Unalias(typ).(*types.Named)
	if !ok || named.Obj() == nil {
		return types.TypeString(typ, nil)
	}
	return named.Obj().Name()
}

func fieldNames(fields []FieldSpec) []string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.Name)
	}
	return names
}

func countBaseInputNames(inputs []model.ResolvedType) map[string]int {
	counts := make(map[string]int, len(inputs))
	for _, input := range inputs {
		counts[baseInputName(input.Expr)]++
	}
	return counts
}

func constructorParameterName(expr string, counts map[string]int) string {
	base := baseInputName(expr)
	if counts[base] == 1 {
		return toLowerCamel(base)
	}
	return toLowerCamel(safeIdentifier(expr))
}

func splitMethodName(expr string, counts map[string]int) string {
	base := baseInputName(expr)
	if counts[base] == 1 {
		return "To" + toCamel(base)
	}
	return "To" + safeIdentifier(expr)
}

func baseInputName(expr string) string {
	trimmed := expr
	if index := strings.Index(trimmed, "["); index >= 0 {
		trimmed = trimmed[:index]
	}
	if index := strings.LastIndex(trimmed, "."); index >= 0 {
		trimmed = trimmed[index+1:]
	}
	trimmed = strings.TrimLeft(trimmed, "[]*")

	if strings.HasPrefix(trimmed, "struct{") || trimmed == "" {
		return safeIdentifier(expr)
	}
	return trimmed
}

func safeIdentifier(expr string) string {
	var builder strings.Builder
	for _, r := range expr {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		} else {
			builder.WriteRune('_')
		}
	}
	safe := builder.String()
	for strings.Contains(safe, "__") {
		safe = strings.ReplaceAll(safe, "__", "_")
	}
	safe = strings.Trim(safe, "_")
	if safe == "" {
		return "Anon"
	}

	parts := strings.Split(safe, "_")
	var camelBuilder strings.Builder
	for _, part := range parts {
		camelBuilder.WriteString(toCamel(part))
	}
	return camelBuilder.String()
}

func toCamel(name string) string {
	if name == "" {
		return ""
	}
	runes := []rune(name)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func toLowerCamel(name string) string {
	if name == "" {
		return ""
	}
	runes := []rune(name)

	upperCount := 0
	for _, r := range runes {
		if unicode.IsUpper(r) {
			upperCount++
		} else {
			break
		}
	}

	if upperCount == 0 {
		return name
	}

	if upperCount == len(runes) {
		return strings.ToLower(name)
	}

	if upperCount == 1 {
		runes[0] = unicode.ToLower(runes[0])
		return string(runes)
	}

	for i := 0; i < upperCount-1; i++ {
		runes[i] = unicode.ToLower(runes[i])
	}

	return string(runes)
}
