package base

//go:generate go run ../../../../../main.go

type A[T any] struct {
	ID T
}

type B struct {
	Name string
}
