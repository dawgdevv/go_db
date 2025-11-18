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

func (node BNode) getPtr(idx uint16) uint64 {
	pos := 4 + 8*idx
	return binary.LittleEndian.Uint64(node[pos : pos+8])
}
func (node BNode) setPtr(idx uint16, val uint64) {
	pos := 4 + 8*idx
	binary.LittleEndian.PutUint64(node[pos:pos+8], val)
}

//// offset array access

func (node BNode) getOffset(idx uint16) uint16 {
	if idx == 0 {
		return 0
	}

	pos := 4 + 8*node.nkeys() + 2*(idx-1)
	return binary.LittleEndian.Uint16(node[pos : pos+2])

}
func (node BNode) setOffset(idx uint16, val uint16) {
	pos := 4 + 8*node.nkeys() + 2*(idx-1)
	binary.LittleEndian.PutUint16(node[pos:pos+2], val)
}

func (node BNode) kvPos(idx uint16) uint16 {
	return 4 + 8*node.nkeys() + 2*node.nkeys() + node.getOffset(idx)
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
	return node.kvPos(node.nkeys())
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

func nodeAppendKv(new BNode, idx uint16, ptr uint64, key []byte, val []byte) {
	new.setPtr(idx, ptr) /// set the pointer

	// compute start position for this KV based on previous offset or zero
	var offset uint16
	if idx == 0 {
		offset = 0
	} else {
		prevPos := new.kvPos(idx - 1)
		prevKeyLen := binary.LittleEndian.Uint16(new[prevPos : prevPos+2])
		prevValLen := binary.LittleEndian.Uint16(new[prevPos+2+prevKeyLen : prevPos+4+prevKeyLen])
		offset = new.getOffset(idx-1) + 4 + prevKeyLen + prevValLen
	}

	new.setOffset(idx, offset)
	pos := new.kvPos(idx)

	binary.LittleEndian.PutUint16(new[pos:pos+2], uint16(len(key)))                                     // write key length
	copy(new[pos+2:], key)                                                                              // write key
	binary.LittleEndian.PutUint16(new[pos+2+uint16(len(key)):pos+4+uint16(len(key))], uint16(len(val))) // write value length
	copy(new[pos+4+uint16(len(key)):], val)                                                             // write value
}

func nodeAppendRange(new BNode, old BNode, dstNew uint16, srcOld uint16, n uint16) {
	// copy pointers and recompute offsets/KVs one by one to avoid layout corruption
	for i := uint16(0); i < n; i++ {
		ptr := old.getPtr(srcOld + i)
		key := old.getKey(srcOld + i)
		val := old.getVal(srcOld + i)
		new.setHeader(old.btype(), dstNew+i+1) // ensure nkeys reflects appended keys progressively
		nodeAppendKv(new, dstNew+i, ptr, key, val)
	}
}
