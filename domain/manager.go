package domain

type NodeManager interface {
	Count() int
	GetAll() []Node
	GetById(id string) (Node, error)
}
