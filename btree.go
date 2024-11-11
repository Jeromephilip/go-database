package btree

import (
	"fmt"
	"encoding/binary"
	"unsafe"
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
	assert(idx < node.nkeys())
	pos := HEADER + 8*idx
	return binary.LittleEndian.Uint64(node[pos:])
}

// Sets the pointer with idx and value
func (node BNode) setPtr(idx uint16, val uint64)


func offsetPos(node BNode, idx uint16) uint16 {
	assert(1 <= idx && idx <= node.nkeys())
}

// Manage key-value offsets within the node
func (node BNode) getOffset(idx uint16) uint16 {
	if idx == 0 {
		return 0
	}

	return binary.LittleEndian.Uint16(node[offsetPos(node, idx):])
}
func (node BNode) setOffset(idx uint16 offset uint16)

// kvPos returns the positon of the nth KV pair relative to the whole node.
func (node BNode) kvPos(idx uint16) uint16 {
	assert(idx <= node.nkeys())
	return HEADER + 8*node.nkeys() + 2*node.nkeys() + node.getOffset(idx)
}

func (node BNode) getKey(idx uint16) []byte {
	assert(idx < node.nkeys())
	pos := node.kvPos(idx)
	klen := binary.LittleEndian.Uint16(node[pos:])

	return node[pos+4:][:klen]
}

func (node BNode) getVal(idx uint16) []byte

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
func leafInsert(
	new BNode, old BNode, idx uint16,
	key []byte, val []byte 
) {
	new.setHeader(BNODE_LEAF, old.nkeys() + 1) // setup the header
	nodeAppendRange(new, old, 0, 0, idx)
	nodeAppendKV(new, idx, 0, key, val)
	nodeAppendRange(new, old, idx+1, idx, old.nkeys()-idx)
}

// NODE COPYING FUNCTIONS
// copy a KV into the position
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
		nodeAppendKV(new, idx+uint16(i), tree.new(Node), node.getKey(0), nil)
		// 				  ^position      ^pointer        ^key            ^val
	}
	nodeAppendRange(new, old, idx+inc, idx+1, old.nkeys()-(idx+1))
}


func init() {
	node1max := HEADER + 8 + 2 + 4 + BTREE_MAX_KEY_SIZE + BTREE_MAX_VAL_SIZE
	assert(node1max <= BTREE_PAGE_SIZE)
}