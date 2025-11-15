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
	found := uint16(0)

	for i := uint16(0); i < nkeys; i++ {
		cmp := bytes.Compare(node.getKey(i), key)

		if cmp <= 0 {
			found = i
		}

		if cmp >= 0 {
			break
		}
	}

	// Check outside the loop
	if found < nkeys {
		cmp := bytes.Compare(node.getKey(found), key)
		if cmp == 0 {
			return found, true
		}
	}

	return found, false
}

func nodeAppendKv(new BNode, idx uint16, ptr uint64, key []byte, val []byte) {
	new.setPtr(idx, ptr) /// set the pointer

	pos := new.kvPos(idx) /// calculate the position to write the kv

	binary.LittleEndian.PutUint16(new[pos:pos+2], uint16(len(key))) //write key length

	copy(new[pos+2:], key) //write key

	binary.LittleEndian.PutUint16(new[pos+2+uint16(len(key)):pos+4+uint16(len(key))], uint16(len(val))) //write value length

	copy(new[pos+4+uint16(len(key)):], val) //write value

	///update offset for next kv
	if idx < new.nkeys()-1 {
		new.setOffset(idx+1, new.getOffset(idx)+4+uint16(len(key))+uint16(len(val)))
	}
}

func nodeAppendRange(new BNode, old BNode, dstNew uint16, srcOld uint16, n uint16) {
	//// Copy pointers
	for i := uint16(0); i < n; i++ {
		new.setPtr(dstNew+i, old.getPtr(srcOld+i))
	}

	//// Copy offsets
	for i := uint16(0); i < n; i++ {
		new.setOffset(dstNew+i, old.getOffset(srcOld+i))
	}

	//// Copy KV data in bulk
	srcBegin := old.kvPos(srcOld)
	srcEnd := old.kvPos(srcOld + n)
	dstBegin := new.kvPos(dstNew)
	copy(new[dstBegin:], old[srcBegin:srcEnd])
}
