package model

import "go/types"

type GeneratedType struct {
	Kind           DeclarationKind
	Name           string
	TypeParameters []string
	Inputs         []GeneratedInput
	Fields         []GeneratedField
	Sum            *GeneratedSum
}

type GeneratedInput struct {
	Expression    string
	ParameterName string
	Type          types.Type
	MethodName    string
	FieldNames    []string
}

type GeneratedField struct {
	Name      string
	Type      types.Type
	Tag       string
	Anonymous bool
}

type GeneratedSum struct {
	GenerateSetters  bool
	InterfaceMethods []GeneratedInterfaceMethod
	Variants         []GeneratedSumVariant
	CommonFields     []GeneratedCommonField
}

type GeneratedInterfaceMethod struct {
	Name      string
	Signature *types.Signature
}

type GeneratedSumVariant struct {
	Expression string
	Type       types.Type
	TypeName   string
}

type GeneratedCommonField struct {
	Name       string
	Type       types.Type
	GetterName string
	SetterName string
	Paths      [][]string
}

type GeneratedFile struct {
	PackagePath      string
	PackageName      string
	Imports          []PassthroughImport
	PassthroughDecls []string
	Generated        []GeneratedType
}
