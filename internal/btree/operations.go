package btree

// assert panics if the condition is false
func assert(condition bool) {
	if !condition {
		panic("assertion failed")
	}
}

// nodeSplit2 splits one node into two nodes
func nodeSplit2(left BNode, right BNode, old BNode) {
	assert(old.nkeys() >= 2)

	// Start by splitting roughly in half
	nleft := old.nkeys() / 2
	assert(nleft >= 1)

	// Calculate bytes needed for right node (header + pointers + offsets + KV data)
	calcRightBytes := func(split uint16) uint16 {
		nright := old.nkeys() - split
		kvDataSize := old.nbytes() - old.kvPos(split)
		// For internal nodes: nright keys, nright+1 pointers
		// For leaf nodes: nright keys, nright pointers (unused)
		ptrSlots := nright
		if old.btype() == BNODE_INTERNAL {
			ptrSlots = nright + 1
		}
		return 4 + 8*ptrSlots + 2*nright + kvDataSize
	}

	// Adjust split point to ensure right node fits in page
	for calcRightBytes(nleft) > BTREE_PAGE_SIZE && nleft < old.nkeys()-1 {
		nleft++
	}

	assert(nleft < old.nkeys())
	nright := old.nkeys() - nleft

	// Set headers once with final key counts and copy data
	left.setHeader(old.btype(), nleft)
	right.setHeader(old.btype(), nright)

	if old.btype() == BNODE_LEAF {
		leafAppendRange(left, old, 0, 0, nleft)
		leafAppendRange(right, old, 0, nleft, nright)
	} else {
		internalAppendRange(left, old, 0, 0, nleft)
		internalAppendRange(right, old, 0, nleft, nright)
	}

	assert(right.nbytes() <= BTREE_PAGE_SIZE)
}

func nodeSplit3(old BNode) (uint16, [3]BNode) {
	if old.nbytes() <= BTREE_PAGE_SIZE {
		return 1, [3]BNode{old, nil, nil}
	}

	left := BNode(make([]byte, BTREE_PAGE_SIZE))
	right := BNode(make([]byte, BTREE_PAGE_SIZE))
	nodeSplit2(left, right, old)

	if left.nbytes() <= BTREE_PAGE_SIZE {
		left = left[:BTREE_PAGE_SIZE]
		return 2, [3]BNode{left, right, nil}
	}

	leftleft := BNode(make([]byte, BTREE_PAGE_SIZE))
	middle := BNode(make([]byte, BTREE_PAGE_SIZE))
	nodeSplit2(leftleft, middle, left)

	assert(leftleft.nbytes() <= BTREE_PAGE_SIZE)
	return 3, [3]BNode{leftleft, middle, right}
}

func nodeMerge(target BNode, left BNode, right BNode) {
	assert(left.btype() == right.btype())

	nkeys := left.nkeys() + right.nkeys()

	target.setHeader(left.btype(), nkeys)

	if left.btype() == BNODE_LEAF {
		leafAppendRange(target, left, 0, 0, left.nkeys())
		leafAppendRange(target, right, left.nkeys(), 0, right.nkeys())
	} else {
		internalAppendRange(target, left, 0, 0, left.nkeys())
		internalAppendRange(target, right, left.nkeys(), 0, right.nkeys())
	}

	assert(target.nbytes() <= BTREE_PAGE_SIZE)
}

func nodeReplace2kid(new BNode, old BNode, idx uint16, ptr0 uint64, key1 []byte, ptr1 uint64) {
	assert(old.btype() == BNODE_INTERNAL)
	assert(idx < old.nkeys())

	new.setHeader(BNODE_INTERNAL, old.nkeys()+1)

	// Copy keys/children before idx
	internalAppendRange(new, old, 0, 0, idx)

	// Insert two new children
	new.setPtr(idx, ptr0)
	internalAppendKv(new, idx, key1)
	new.setPtr(idx+1, ptr1)
	internalAppendKv(new, idx+1, old.getKey(idx))

	// Copy remaining keys/children after idx
	if idx+1 < old.nkeys() {
		internalAppendRange(new, old, idx+2, idx+1, old.nkeys()-(idx+1))
	}
}
