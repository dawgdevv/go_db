package btree

import (
	"bytes"
	"encoding/binary"
)

const BTREE_PAGE_SIZE = 4096
const BTREE_MAX_KEY_SIZE = 1000
const BTREE_MAX_VAL_SIZE = 3000

const (
	BNODE_INTERNAL = 1
	BNODE_LEAF     = 2
)

type BNode []byte

// getters
func (node BNode) btype() uint16 {
	return binary.LittleEndian.Uint16(node[0:2])
}

func (node BNode) nkeys() uint16 {
	return binary.LittleEndian.Uint16(node[2:4])
}

/// setters

func (node BNode) setHeader(btype uint16, nkeys uint16) {
	binary.LittleEndian.PutUint16(node[0:2], btype)
	binary.LittleEndian.PutUint16(node[2:4], nkeys)
}

//// pointer array access
// For internal nodes: nkeys keys and nkeys+1 child pointers (indices 0..nkeys)
// For leaf nodes: nkeys keys and nkeys dummy pointers (unused, always 0)

func (node BNode) getPtr(idx uint16) uint64 {
	pos := 4 + 8*idx
	return binary.LittleEndian.Uint64(node[pos : pos+8])
}

func (node BNode) setPtr(idx uint16, val uint64) {
	pos := 4 + 8*idx
	binary.LittleEndian.PutUint64(node[pos:pos+8], val)
}

// ptrSlots returns the number of pointer slots needed
func (node BNode) ptrSlots() uint16 {
	if node.btype() == BNODE_INTERNAL {
		return node.nkeys() + 1 // internal nodes have nkeys+1 children
	}
	return node.nkeys() // leaf nodes have nkeys entries (pointers unused)
}

//// offset array access

func (node BNode) getOffset(idx uint16) uint16 {
	if idx == 0 {
		// For the first key we treat offset as 0; it is implicit
		return 0
	}
	ptrSlots := node.ptrSlots()
	pos := 4 + 8*ptrSlots + 2*(idx-1)
	return binary.LittleEndian.Uint16(node[pos : pos+2])
}

func (node BNode) setOffset(idx uint16, val uint16) {
	ptrSlots := node.ptrSlots()
	pos := 4 + 8*ptrSlots + 2*(idx-1)
	binary.LittleEndian.PutUint16(node[pos:pos+2], val)
}

func (node BNode) kvPos(idx uint16) uint16 {
	ptrSlots := node.ptrSlots()
	return 4 + 8*ptrSlots + 2*node.nkeys() + node.getOffset(idx)
}

func (node BNode) getKey(idx uint16) []byte {
	pos := node.kvPos(idx)
	keyLen := binary.LittleEndian.Uint16(node[pos : pos+2])
	return node[pos+2 : pos+2+keyLen]
}

func (node BNode) getVal(idx uint16) []byte {
	pos := node.kvPos(idx)
	keyLen := binary.LittleEndian.Uint16(node[pos : pos+2])
	valLen := binary.LittleEndian.Uint16(node[pos+2+keyLen : pos+4+keyLen])
	return node[pos+4+keyLen : pos+4+keyLen+valLen]
}

func (node BNode) nbytes() uint16 {
	ptrSlots := node.ptrSlots()
	if node.nkeys() == 0 {
		return 4 + 8*ptrSlots + 2*node.nkeys()
	}
	// total bytes = header + pointers + offsets + last kv offset + last kv size
	lastIdx := node.nkeys() - 1
	lastPos := node.kvPos(lastIdx)
	keyLen := binary.LittleEndian.Uint16(node[lastPos : lastPos+2])
	valLen := binary.LittleEndian.Uint16(node[lastPos+2+keyLen : lastPos+4+keyLen])
	return lastPos + 4 + keyLen + valLen
}

func nodeLookup(node BNode, key []byte) (uint16, bool) {
	nkeys := node.nkeys()
	if nkeys == 0 {
		return 0, false
	}

	// binary search over sorted keys
	lo, hi := uint16(0), nkeys
	for lo < hi {
		mid := lo + (hi-lo)/2
		cmp := bytes.Compare(node.getKey(mid), key)
		if cmp < 0 {
			lo = mid + 1
		} else if cmp > 0 {
			hi = mid
		} else {
			return mid, true
		}
	}

	// lo is first index where key would be inserted
	// for leaf nodes: insertion position
	// for internal nodes: child pointer to follow
	return lo, false
}

// leafAppendKv appends a key-value pair to a leaf node at the given index.
// For leaf nodes, ptr is unused (always 0).
func leafAppendKv(new BNode, idx uint16, key []byte, val []byte) {
	assert(new.btype() == BNODE_LEAF)
	new.setPtr(idx, 0) // leaf nodes don't use pointers

	// compute start position for this KV based on previous kv end
	var offset uint16
	if idx == 0 {
		offset = 0
	} else {
		prevOffset := new.getOffset(idx - 1)
		prevPos := 4 + 8*new.ptrSlots() + 2*new.nkeys() + prevOffset
		prevKeyLen := binary.LittleEndian.Uint16(new[prevPos : prevPos+2])
		prevValLen := binary.LittleEndian.Uint16(new[prevPos+2+prevKeyLen : prevPos+4+prevKeyLen])
		offset = prevOffset + 4 + prevKeyLen + prevValLen
	}

	new.setOffset(idx, offset)
	pos := 4 + 8*new.ptrSlots() + 2*new.nkeys() + offset

	binary.LittleEndian.PutUint16(new[pos:pos+2], uint16(len(key)))                                     // write key length
	copy(new[pos+2:], key)                                                                              // write key
	binary.LittleEndian.PutUint16(new[pos+2+uint16(len(key)):pos+4+uint16(len(key))], uint16(len(val))) // write value length
	copy(new[pos+4+uint16(len(key)):], val)                                                             // write value
}

// internalAppendKv appends a separator key to an internal node at the given index.
// For internal nodes, values are always empty (nil).
func internalAppendKv(new BNode, idx uint16, key []byte) {
	assert(new.btype() == BNODE_INTERNAL)

	// compute start position for this KV based on previous kv end
	var offset uint16
	if idx == 0 {
		offset = 0
	} else {
		prevOffset := new.getOffset(idx - 1)
		prevPos := 4 + 8*new.ptrSlots() + 2*new.nkeys() + prevOffset
		prevKeyLen := binary.LittleEndian.Uint16(new[prevPos : prevPos+2])
		offset = prevOffset + 4 + prevKeyLen // no value for internal nodes
	}

	new.setOffset(idx, offset)
	pos := 4 + 8*new.ptrSlots() + 2*new.nkeys() + offset

	binary.LittleEndian.PutUint16(new[pos:pos+2], uint16(len(key))) // write key length
	copy(new[pos+2:], key)                                          // write key
	binary.LittleEndian.PutUint16(new[pos+2+uint16(len(key)):], 0)  // write zero value length
}

func leafAppendRange(new BNode, old BNode, dstNew uint16, srcOld uint16, n uint16) {
	// Caller must have set the final header (btype and nkeys) on "new"
	// before calling this function. Here we just copy data sequentially.
	assert(new.btype() == BNODE_LEAF && old.btype() == BNODE_LEAF)
	for i := uint16(0); i < n; i++ {
		key := old.getKey(srcOld + i)
		val := old.getVal(srcOld + i)
		leafAppendKv(new, dstNew+i, key, val)
	}
}

func internalAppendRange(new BNode, old BNode, dstNew uint16, srcOld uint16, n uint16) {
	// Copy child pointers and separator keys from old internal node to new.
	// Caller must have set the final header (btype and nkeys) on "new".
	assert(new.btype() == BNODE_INTERNAL && old.btype() == BNODE_INTERNAL)
	for i := uint16(0); i < n; i++ {
		// Copy child pointer at srcOld+i to dstNew+i
		new.setPtr(dstNew+i, old.getPtr(srcOld+i))
		// Copy separator key
		key := old.getKey(srcOld + i)
		internalAppendKv(new, dstNew+i, key)
	}
	// Copy the last child pointer (at srcOld+n)
	new.setPtr(dstNew+n, old.getPtr(srcOld+n))
}
