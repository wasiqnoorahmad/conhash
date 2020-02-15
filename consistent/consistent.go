package consistent

import (
	"conhash/rpcs"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
	"net/rpc"
	"sort"
	"strconv"
)

// CNode represents a node on the consistent hash ring
type CNode struct {
	Conn      *rpc.Client // Connection to the node
	Key       string      // id of the node
	Weight    int         // weight of the node
	Parent    uint64      // parent hash of the node
	ParentKey string      // key of the parent node
	Hash      uint64      // hash of the node
}

type nodes []*CNode

func (n nodes) Len() int           { return len(n) }
func (n nodes) Less(i, j int) bool { return n[i].Hash < n[j].Hash }
func (n nodes) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }

// CRing contains the attributes related to consistent hashing
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
	return digestAsUint64
}

// GetVirKey returns the next virtual key
// with regards to n
func (r *CRing) GetVirKey(key string, n int) string {
	if n == 0 {
		return key
	}
	return key + strconv.Itoa(n)
}

// AddNode adds a new node into the ring
func (r *CRing) AddNode(args *rpcs.JoinArgs) bool {
	weight := args.Weight
	key := args.ID

	if _, exist := r.parents[key]; exist {
		return false
	}

	// Try connecting the node
	// TODO: Get IP via something else ...
	conn, err := rpc.DialHTTP("tcp", ":"+strconv.Itoa(args.Port))
	if err != nil {
		return false
	}

	hash := r.GenHash(key)
	// Setting the parent node
	node := CNode{
		Conn:      conn,
		ParentKey: key,
		Parent:    hash,
		Hash:      hash,
		Key:       key,
		Weight:    weight,
	}
	r.parents[key] = &node
	r.nodes = append(r.nodes, &node)
	weight--

	for weight > 0 {
		seed := r.GetVirKey(key, weight)
		hash := r.GenHash(seed)
		virNode := CNode{
			Conn:      conn,
			ParentKey: node.Key,
			Parent:    node.Hash,
			Hash:      hash,
			Key:       seed,
			Weight:    node.Weight,
		}
		r.nodes = append(r.nodes, &virNode)
		weight--
	}
	sort.Sort(r.nodes)

	return true
}

// RemoveNode removes a node from the ring provided its key
// as the argument
func (r *CRing) RemoveNode(key string) {
	parent, exist := r.parents[key]

	if !exist {
		return
	}

	walk := 0
	for walk != r.nodes.Len() {
		node := r.nodes[walk]
		if node.Parent == parent.Hash {
			r.nodes = append(r.nodes[:walk], r.nodes[walk+1:]...)
		}
		walk++
	}
	delete(r.parents, key)
}

// GetPrevParent returns the previous node in the ring
// from other parent...
func (r *CRing) GetPrevParent(node *CNode) *CNode {
	if r.Size() == 1 {
		return nil
	}

	walk := r.nodes.Len() - 1
	for walk != -1 {
		curr := r.nodes[walk]
		if curr.Hash < node.Hash && curr.Parent != node.Parent {
			return curr
		}
		walk--
	}

	walk = r.nodes.Len() - 1
	for walk != -1 {
		curr := r.nodes[walk]
		if curr.Parent != node.Parent {
			return curr
		}
		walk--
	}
	return nil
}

// GetNextExcept returns the next parent in the consistent ring
// except other than they key specified
func (r *CRing) GetNextExcept(node *CNode, key string) *CNode {
	ret := r.GetNextParent(node)
	for ret.ParentKey == key {
		ret = r.GetNextParent(ret)
	}
	return ret
}

// GetNextParent returns the next parent in the consistent ring
func (r *CRing) GetNextParent(node *CNode) *CNode {
	if r.Size() == 1 {
		return nil
	}

	walk := 0
	for walk != r.nodes.Len() {
		curr := r.nodes[walk]
		if curr.Hash > node.Hash && curr.Parent != node.Parent {
			return curr
		}
		walk++
	}
	walk = 0
	// Otherwise get the first node in ring from other parent
	for walk != r.nodes.Len() {
		curr := r.nodes[walk]
		if curr.Parent != node.Parent {
			return curr
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
	// if r.Size() == 1 {
	// 	return nil
	// }

	hash := r.GenHash(key)
	walk := 0

	for walk != r.nodes.Len() {
		curr := r.nodes[walk]
		if curr.Hash >= hash {
			return curr
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
		fmt.Println("Hash:", node.Hash, "\tKey:", node.Key)
		walk++
	}
	fmt.Println("=====================")
}
