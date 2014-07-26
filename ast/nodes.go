package ast

type Node interface {
}

type Nodes []Node

type Block struct {
	Statements []Node
}

type ConstantInt struct {
	Value int
}

type ConstantFloat struct {
	Value float64
}

type SimpleString struct {
	Value string
}

type Symbol struct {
	Name string
}

type BareReference struct {
	Name string
}

type CallExpression struct {
	Func Node
	Args []Node
}

type FuncDecl struct {
	Name BareReference
	Args []Node
	Body []Node
}

type ClassDecl struct {
	Name       BareReference
	SuperClass BareReference
	Namespace  string
	Body       []Node
}
