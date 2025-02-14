package app

// BType defines the type of value stored in BNode.
type BType int

const (
	BString BType = iota
	BInt
	BList
	BDict
)

// BNode represents a self-referencing structure to hold Bencode values.
type BNode struct {
	Type BType
	Str  string
	Int  int
	List []*BNode
	Dict map[string]*BNode
}
