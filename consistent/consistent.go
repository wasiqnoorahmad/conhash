package consistent

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash"
	"sort"
)

// CNode represents a node on the consistent hash ring
type CNode struct {
	Key     string
	hash    string
	Weight  uint16
	virtual bool
}

type nodes []*CNode

func (n nodes) Len() int           { return len(n) }
func (n nodes) Less(i, j int) bool { return n[i].hash < n[j].hash }
func (n nodes) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }

// CRing contains the attributes related to consistent
// hashing.
type CRing struct {
	parents nodes
	nodes   nodes
	hasher  hash.Hash
}

// NewRing returns a new instance of a consistent hash ring
func NewRing() *CRing {
	return &CRing{
		hasher: md5.New(),
	}
}

// GenHash produces MD5 hash of given string
func (r *CRing) GenHash(key string) string {
	r.hasher.Write([]byte(key))
	digest := hex.EncodeToString(r.hasher.Sum(nil))
	return digest
}

// AddNode adds a new node into the ring
// todo: change it to receive hash and node struct
func (r *CRing) AddNode(hash string, node *CNode) {
	node.hash = hash
	node.virtual = false
	r.nodes = append(r.nodes, node)
	r.parents = append(r.parents, node)
	weight := node.Weight

	seed := r.GenHash(hash)
	for weight > 1 {
		node := CNode{hash: seed,
			Key:     node.Key,
			virtual: true,
			Weight:  node.Weight}
		r.nodes = append(r.nodes, &node)
		seed = r.GenHash(seed)
		weight--
	}
	sort.Sort(r.nodes)
	sort.Sort(r.parents)
}

// RemoveNode removes a node from the ring provided its key
// as the argument
func (r *CRing) RemoveNode(key string) {
	walk := 0

	for walk != r.nodes.Len() {
		node := r.nodes[walk]
		if node.Key == key {
			r.nodes = append(r.nodes[:walk], r.nodes[walk+1:]...)
		}
		walk++
	}

	walk = 0
	for walk != r.parents.Len() {
		node := r.parents[walk]
		if node.Key == key {
			r.parents = append(r.parents[:walk], r.parents[walk+1:]...)
		}
		walk++
	}
}

// GetNextParent returns the next parent in the
// consistent ring
func (r *CRing) GetNextParent(hash string) *CNode {
	walk := 0
	for walk != r.parents.Len() {
		if r.parents[walk].hash > hash {
			return r.parents[walk]
		}
		walk++
	}
	if r.parents.Len() == 1 {

		return nil
	}
	return r.parents[0]
}

// GetNext returns the next node in the consistent ring
// after the key hash
func (r *CRing) GetNext(hash string) *CNode {
	walk := 0
	for walk != r.nodes.Len() {
		if r.nodes[walk].hash >= hash {
			return r.nodes[walk]
		}
		walk++
	}

	if r.nodes.Len() == 1 {
		return nil
	}
	return r.nodes[0]
}

// Display prints the ring on the console
func (r *CRing) Display() {
	walk := 0
	for walk < r.nodes.Len() {
		node := r.nodes[walk]
		fmt.Printf("Hash %s | Key %s\n", node.hash, node.Key)
		walk++
	}
}
