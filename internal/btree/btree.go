package btree

type BTree struct {
	root uint64
	get  func(uint64) []byte
	new  func([]byte) uint64
	put  func([]byte)
	del  func(uint64)
}

func treeInsert(tree *BTree, node BNode, key []byte, val []byte) BNode {
	new := BNode(make([]byte, 2*BTREE_PAGE_SIZE)) //allocation of the new node

	idx, found := nodeLookup(node, key) //Find the key pos

	switch node.btype() {
	case BNODE_LEAF:
		if found {
			leafUpdate(new, node, idx, key, val)
		} else {
			leafInsert(new, node, idx, key, val)
		}

	case BNODE_INTERNAL:
		nodeInsert(tree, new, node, idx, key, val, found)

	default:
		panic("invalid node type")
	}
	return new
}

func leafUpdate(new BNode, old BNode, idx uint16, key []byte, val []byte) {
	assert(old.btype() == BNODE_LEAF)
	assert(idx < old.nkeys())

	new.setHeader(BNODE_LEAF, old.nkeys())

	nodeAppendRange(new, old, 0, 0, idx)
	nodeAppendKv(new, idx, 0, key, val)
	if idx+1 < old.nkeys() {
		nodeAppendRange(new, old, idx+1, idx+1, old.nkeys()-(idx+1))
	}

	assert(new.nbytes() <= BTREE_PAGE_SIZE)
}

func leafInsert(new BNode, old BNode, idx uint16, key []byte, val []byte) {
	assert(old.btype() == BNODE_LEAF)
	assert(idx <= old.nkeys())

	new.setHeader(BNODE_LEAF, old.nkeys()+1)

	nodeAppendRange(new, old, 0, 0, idx)
	nodeAppendKv(new, idx, 0, key, val)
	if idx < old.nkeys() {
		nodeAppendRange(new, old, idx+1, idx, old.nkeys()-idx)
	}
	assert(new.nbytes() <= BTREE_PAGE_SIZE)
}

func nodeInsert(tree *BTree, new BNode, node BNode, idx uint16, key []byte, val []byte, found bool) {
	assert(node.btype() == BNODE_INTERNAL)
	assert(idx <= node.nkeys())

	kptr := node.getPtr(idx)

	knode := tree.get(kptr)
	knode = treeInsert(tree, knode, key, val)

	nsplit, splitted := nodeSplit3(knode)

	tree.del(kptr)

	switch nsplit {
	case 1:
		new.setHeader(BNODE_INTERNAL, node.nkeys())
		nodeAppendRange(new, node, 0, 0, idx)
		nodeAppendKv(new, idx, tree.new(splitted[0]), splitted[0].getKey(0), nil)
		if idx+1 <= node.nkeys() {
			nodeAppendRange(new, node, idx+1, idx+1, node.nkeys()-(idx+1))
		}
	case 2:
		nodeReplace2kid(new, node, idx, tree.new(splitted[0]), splitted[1].getKey(0))
		new.setPtr(idx+1, tree.new(splitted[1]))
	case 3:
		nodeReplace3kid(new, node, idx, tree.new(splitted[0]), splitted[1].getKey(0), tree.new(splitted[1]), splitted[2].getKey(0), tree.new(splitted[2]))
	default:
		panic("invalid split count")
	}
}

func nodeReplace3kid(new BNode, old BNode, idx uint16, ptr1 uint64, key1 []byte, ptr2 uint64, key2 []byte, ptr3 uint64) {
	assert(old.btype() == BNODE_INTERNAL)

	new.setHeader(BNODE_INTERNAL, old.nkeys()+2)
	nodeAppendRange(new, old, 0, 0, idx)
	nodeAppendKv(new, idx, ptr1, key1, nil)
	nodeAppendKv(new, idx+1, ptr2, key2, nil)
	nodeAppendKv(new, idx+2, ptr3, old.getKey(idx), nil)
	if idx+1 < old.nkeys() {
		nodeAppendRange(new, old, idx+3, idx+1, old.nkeys()-(idx+1))
	}

	assert(new.nbytes() <= BTREE_PAGE_SIZE)
}

func (tree *BTree) Insert(key []byte, val []byte) {
	assert(len(key) > 0 && len(key) <= BTREE_MAX_KEY_SIZE)
	assert(len(val) <= BTREE_MAX_VAL_SIZE)

	if tree.root == 0 {
		root := BNode(make([]byte, BTREE_PAGE_SIZE))
		root.setHeader(BNODE_LEAF, 0)
		tree.root = tree.new(root)

	}

	node := tree.get(tree.root)
	tree.del(tree.root)

	node = treeInsert(tree, node, key, val)

	nsplit, splitted := nodeSplit3(node)

	if nsplit > 1 {
		root := BNode(make([]byte, BTREE_PAGE_SIZE))
		root.setHeader(BNODE_INTERNAL, nsplit)
		for i := uint16(0); i < nsplit; i++ {
			ptr := tree.new(splitted[i])
			if i > 0 {
				nodeAppendKv(root, i-1, ptr, splitted[i].getKey(0), nil)
			}
			root.setPtr(i, ptr)
		}
		tree.root = tree.new(root)
	} else {
		tree.root = tree.new(splitted[0])
	}
}
