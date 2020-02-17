package node

import (
	"conhash/rpcs"
)

// Node ...
type Node interface {
	StartNode(dst string) error
	Close()
}

type replicaEx struct {
	args *rpcs.ReplicaArgs
	rep  chan rpcs.Ack
}

type requestEx struct {
	args *rpcs.ReqArgs
	rep  chan rpcs.Ack
}

type removeEx struct {
	args *rpcs.RemoveAll
	rep  chan rpcs.Ack
}

type copyEx struct {
	args *rpcs.CopyArgs
	rep  chan rpcs.Ack
}

type replaceEx struct {
	args *rpcs.ReplaceArgs
	rep  chan rpcs.Ack
}

type lookupEx struct {
	args *rpcs.LookupInfo
	rep  chan rpcs.Ack
}

type bulkEx struct {
	args *rpcs.LookupInfo
	rep  chan rpcs.BulkStates
}

type stateEx struct {
	args *rpcs.SyncArgs
	rep  chan rpcs.Ack
}
