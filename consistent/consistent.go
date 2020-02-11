package consistent

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
	"sort"
	"strconv"
)

// CNode represents a node on the consistent hash ring
type CNode struct {
	Key    string // id of the node
	Weight int    // weight of the node
	Parent uint64 // parent hash of the node
	Hash   uint64 // hash of the node
}

type nodes []*CNode

func (n nodes) Len() int           { return len(n) }
func (n nodes) Less(i, j int) bool { return n[i].Hash < n[j].Hash }
func (n nodes) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }

// CRing contains the attributes related to consistent
// hashing.
type CRing struct {
	parents map[string]*CNode
	nodes   nodes
	hasher  hash.Hash
}

// NewRing returns a new instance of a consistent hash ring
func NewRing() *CRing {
	return &CRing{
		parents: make(map[string]*CNode),
		hasher:  sha256.New(),
	}
}

// GenHash produces Sha256 hash of given string
func (r *CRing) GenHash(key string) uint64 {
	hasher := sha256.New()
	hasher.Write([]byte(key))
	digest := hasher.Sum(nil)
	digestAsUint64 := binary.LittleEndian.Uint64(digest)
	return digestAsUint64 % 2141
}

// AddNode adds a new node into the ring
// todo: check if node already exist in ring
func (r *CRing) AddNode(key string, weight int) bool {

	if _, exist := r.parents[key]; exist {
		return false
	}

	hash := r.GenHash(key)
	// Setting the parent node
	node := CNode{
		Parent: hash,
		Hash:   hash,
		Key:    key,
		Weight: weight,
	}
	r.parents[key] = &node
	r.nodes = append(r.nodes, &node)

	seed := hash
	for weight > 1 {
		node := CNode{
			Parent: hash,
			Hash:   seed,
			Key:    key,
			Weight: node.Weight,
		}
		r.nodes = append(r.nodes, &node)
		seed = r.GenHash(strconv.Itoa((int)(seed)))
		weight--
	}
	sort.Sort(r.nodes)

	return true
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
}

// GetNextParent returns the next parent in the consistent ring
func (r *CRing) GetNextParent(node *CNode) *CNode {

	if r.Size() == 1 {
		return nil
	}

	hash := r.GenHash(key)
	walk := 0

	for walk != r.nodes.Len() {
		node := r.nodes[walk]
		if node.Hash >= hash && node.Parent {
			return node
		}
		walk++
	}

	return nil
}

// Size returns the number of physical nodes in the ring
func (r *CRing) Size() int {
	return len(r.parents)
}

// GetNext returns the next node in the consistent ring
// after the key hash
func (r *CRing) GetNext(key string) *CNode {
	if r.Size() == 1 {
		return nil
	}

	hash := r.GenHash(key)
	walk := 0

	for walk != r.nodes.Len() {
		if r.nodes[walk].Hash >= hash {
			return r.nodes[walk]
		}
		walk++
	}

	return r.nodes[0]
}

// Display prints the ring on the console
func (r *CRing) Display() {
	walk := 0
	for walk < r.nodes.Len() {
		node := r.nodes[walk]
		fmt.Printf("Hash %s \t Key %s\n", node.hash, node.Key)
		walk++
	}
	fmt.Println("=====================")
}
