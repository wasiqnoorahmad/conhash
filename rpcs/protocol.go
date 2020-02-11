package rpcs

// JoinArgs is used for proving args for join RPCs
type JoinArgs struct {
	Port   int
	ID     string
	Weight int
}

// Ack is used to provide acknowledgments for RPCs
type Ack struct {
	Success bool
}
