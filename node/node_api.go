package node

// Node is daga rora dagga pista
type Node interface {
	StartNode(dst string) error

	// Closes the Node
	Close()
}
