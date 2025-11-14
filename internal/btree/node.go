package btree

import (
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

func (n BNode) btype() uint16 {
	return binary.LittleEndian.Uint16(n[0:2])
}

func (n BNode) nkeys() uint16 {
	return binary.LittleEndian.Uint16(n[2:4])
}

func (n BNode) setHeader(btype uint16, nkeys uint16) {
	binary.LittleEndian.PutUint16(n[0:2], btype)
	binary.LittleEndian.PutUint16(n[2:4], nkeys)
}

///pointer array access

func (n BNode) getPtr(idx uint16) uint64 {
	pos := 4 + 8*idx
	return binary.LittleEndian.Uint64(n[pos : pos+8])
}
func (n BNode) setPtr(idx uint16, val uint64) {
	pos := 4 + 8*idx
	binary.LittleEndian.PutUint64(n[pos:pos+8], val)
}

// offset array access

func (n BNode) getOffset(idx uint16) uint16 {
	if idx == 0 {
		return 0
	}

	pos := 4 + 8*n.nkeys() + 2*(idx-1)
	return binary.LittleEndian.Uint16(n[pos : pos+2])

}
func (n BNode) setOffset(idx uint16, val uint16) {
	pos := 4 + 8*n.nkeys() + 2*(idx-1)
	binary.LittleEndian.PutUint16(n[pos:pos+2], val)
}

func (n BNode) kvPos(idx uint16) uint16 {
	return 4 + 8*n.nkeys() + 2*(n.nkeys()) + n.getOffset(idx)
}

func (n BNode) getKey(idx uint16) []byte {
	pos := n.kvPos(idx)
	keyLen := binary.LittleEndian.Uint16(n[pos : pos+2])
	return n[pos+2 : pos+2+keyLen]
}

func (n BNode) getVal(idx uint16) []byte {
	pos := n.kvPos(idx)
	keyLen := binary.LittleEndian.Uint16(n[pos : pos+2])
	valLen := binary.LittleEndian.Uint16(n[pos+2+keyLen : pos+4+keyLen])
	return n[pos+4+keyLen : pos+4+keyLen+valLen]
}

func (n BNode) nbytes() uint16 {
	return n.kvPos(n.nkeys())
}
