package btree

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unsafe"

	"github.com/Jeromephilip/go-database/utils"
)

const HEADER = 4

const BTREE_PAGE_SIZE = 4096
const BTREE_MAX_KEY_SIZE = 1000
const BTREE_MAX_VAL_SIZE = 3000

type BNode []byte // dumped to disk

const (
	BNODE_NODE = 1 // internal nodes without values
	BNODE_LEAF = 2 // leaf nodes with values
)

type BTree struct {
	root uint64
	get func(uint64) []byte // dereference a pointer
	new func([]byte) uint64 // allocate a new page
	del func(uint64) 		// deallocate a page
}

// return the type of node (internal or leaf) reading the first two bytes
func (node BNode) btype() uint16 {
	return binary.LittleEndian.Uint16(node[0:2])
}

// return the number of keys in the node
func (node BNode) nkeys() uint16 {
	return binary.LittleEndian.Uint16(node[2:4])
}

// set the node type and key count in the header
func (node BNode) setHeader(btype uint16, nkeys uint16) {
	binary.LittleEndian.PutUint16(node[0:2], btype)
	binary.LittleEndian.PutUint16(node[2:4], nkeys)
}

// manage pointers within the node by encoding and decoding 8-byte positions in mem
// retrives a pointer for a specific key index
func (node BNode) getPtr(idx uint16) uint64 {
	utils.Assert(idx < node.nkeys(), "index less than value")
	pos := HEADER + 8*idx
	return binary.LittleEndian.Uint64(node[pos:])
}

// Sets the pointer with idx and value
func (node BNode) setPtr(idx uint16, val uint64)


func offsetPos(node BNode, idx uint16) uint16 {
	utils.Assert(1 <= idx && idx <= node.nkeys(), "not found offset position")
}

// Manage key-value offsets within the node
func (node BNode) getOffset(idx uint16) uint16 {
	if idx == 0 {
		return 0
	}

	return binary.LittleEndian.Uint16(node[offsetPos(node, idx):])
}
func (node BNode) setOffset(idx uint16, offset uint16)

// kvPos returns the positon of the nth KV pair relative to the whole node.
func (node BNode) kvPos(idx uint16) uint16 {
	utils.Assert(idx <= node.nkeys(), "index is greater than nkeys")
	return HEADER + 8*node.nkeys() + 2*node.nkeys() + node.getOffset(idx)
}

// computes the position of the key-value pair within the node
func (node BNode) getKey(idx uint16) []byte {
	utils.Assert(idx < node.nkeys(), "index is greater than nkeys")
	pos := node.kvPos(idx)
	klen := binary.LittleEndian.Uint16(node[pos:])

	return node[pos+4:][:klen]
}

// Retrieves the key at a specific index by decoding it from the encoded position and length in the node
func (node BNode) getVal(idx uint16) []byte

func (node BNode) nbytes() uint16 {
	return node.kvPos(node.nkeys())
}

// Seek operation used for both range and point queries. So they are the same.
func nodeLookupLE(node BNode, key []byte) uint16 {
	nkeys := node.nkeys()
	found := uint16(0)
	// the first key is a copy from the parent node
	// thus it's always less than or equal to the key
	for i := uint16(1); i < nkeys; i++ {
		cmp := bytes.Compare(node.getKey(i), key)

		if cmp <= 0 {
			found = i
		}

		if cmp >= 0 {
			break
		}
	}

	return found
}

// Insering leaves into B+Tree
// GOALS:
// update the header to reflect the new key count,
// copies existing keys before and after the insertion index
// inserts the new key-value pair at the correct index
func leafInsert(
	new BNode, old BNode, idx uint16,
	key []byte, val []byte,
) {
	new.setHeader(BNODE_LEAF, old.nkeys() + 1) // setup the header
	nodeAppendRange(new, old, 0, 0, idx)
	nodeAppendKV(new, idx, 0, key, val)
	nodeAppendRange(new, old, idx+1, idx, old.nkeys()-idx)
}

// NODE COPYING FUNCTIONS
// copy a KV into the position
// add a specific key-value pair to a specific position within a node. It writes
// the key's length, value's length, and then copies the key and values bytes into
// correct location
func nodeAppendKV(new BNode, idx uint16, ptr uint64, key []byte, val []byte) {
	// ptrs
	new.setPtr(idx, ptr)

	// KV
	pos := new.kvPos(idx)
	binary.LittleEndian.PutUint16(new[pos+0:], uint16(len(key)))
	binary.LittleEndian.PutUint16(new[pos+2:], uint16(len(val)))
	copy(new[pos+4:], key)
	copy(new[pos+4+uint16(len(key)):], val)
	// the offset of the next key
	new.setOffset(idx+1, new.getOffset(idx)+4+uint16((len(key) + len(val))))
}

// copy multiple KVs into the position from the old node
func nodeAppendRange(
	new BNode, old BNode,
	dstNew uint16, srcOld uint16, n uint16,
)

func nodeReplaceKidN(
	tree *BTree, new BNode, old BNode, idx uint16,
	kids ...BNode,
) {
	inc := uint16(len(kids))
	new.setHeader(BNODE_NODE, old.nkeys()+inc-1)
	nodeAppendRange(new, old, 0, 0, idx)
	for i, node := range kids {
		nodeAppendKV(new, idx+uint16(i), tree.new(node), node.getKey(0), nil)
		// 				  ^position      ^pointer        ^key            ^val
	}
	nodeAppendRange(new, old, idx+inc, idx+1, old.nkeys()-(idx+1))
}

func nodeSplit2(left BNode, right BNode, old BNode) {

}

func nodeSplit3(old BNode) (uint16, [3]BNode) {
	if old.nbytes() <= BTREE_PAGE_SIZE {
		old = old[:BTREE_PAGE_SIZE]
		return 1, [3]BNode{old}
	}

	left := BNode(make([]byte, 2*BTREE_PAGE_SIZE))
	right := BNode(make([]byte, BTREE_PAGE_SIZE))
	nodeSplit2(left, right, old)

	if left.nbytes() <= BTREE_PAGE_SIZE {
		left = left[:BTREE_PAGE_SIZE]
		return 2, [3]BNode{left, right} // 2 nodes
	}

	leftleft := BNode(make([]byte, BTREE_PAGE_SIZE))
	middle := BNode(make([]byte, BTREE_PAGE_SIZE))
	nodeSplit2(leftleft, middle, left)
	utils.Assert(leftleft.nbytes() <= BTREE_PAGE_SIZE, "left node less than the defined page size")
	return 3, [3]BNode{leftleft, middle, right} // 3 nodes
}

func init() {
	node1max := HEADER + 8 + 2 + 4 + BTREE_MAX_KEY_SIZE + BTREE_MAX_VAL_SIZE
	utils.Assert(node1max <= BTREE_PAGE_SIZE, "Node is greater than defined page size")
}