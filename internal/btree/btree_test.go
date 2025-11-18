package btree

import (
	"fmt"
	"testing"
)

// ============================================================================
// TEST INFRASTRUCTURE
// ============================================================================

// testPager simulates page-based storage in memory for testing
type testPager struct {
	pages map[uint64][]byte // page ID -> page data
	next  uint64            // next available page ID
}

func newTestPager() *testPager {
	return &testPager{
		pages: make(map[uint64][]byte),
		next:  1, // start from 1; 0 means "no root"
	}
}

func (p *testPager) New(data []byte) uint64 {
	id := p.next
	p.next++
	// Store a copy to prevent external mutations
	buf := make([]byte, len(data))
	copy(buf, data)
	p.pages[id] = buf
	return id
}

func (p *testPager) Get(id uint64) []byte {
	page, ok := p.pages[id]
	if !ok {
		panic(fmt.Sprintf("page %d not found", id))
	}
	return page
}

func (p *testPager) Del(id uint64) {
	delete(p.pages, id)
}

// Helper to create a BTree with our test pager
func newTestBTree() (*BTree, *testPager) {
	p := newTestPager()
	tree := &BTree{
		root: 0,
		get:  p.Get,
		new:  p.New,
		del:  p.Del,
		put:  func([]byte) {}, // unused for now
	}
	return tree, p
}

// ============================================================================
// TEST 1: INSERT SINGLE KEY-VALUE
// ============================================================================

// TestBTreeInsertSingle tests inserting one key-value pair into an empty tree
func TestBTreeInsertSingle(t *testing.T) {
	// TODO: Let's implement this together!
	// What should we do first?
	// 1. Create a new tree
	// 2. Insert one key-value pair
	// 3. Check that root was created
	// 4. Check that it's a leaf node
	// 5. Check that it has 1 key
	// 6. Verify the key and value are correct

	tree, pager := newTestBTree()
	key := []byte("nishant")
	val := []byte("goodboy")

	tree.Insert(key, val)

	if tree.root == 0 {
		t.Fatal("expected root to be set after insert")
	}

	rootNode := BNode(pager.Get(tree.root))

	if rootNode.btype() != BNODE_LEAF {
		t.Fatalf("expected root to be LEAF, but got type %d", rootNode.btype())
	}

	if rootNode.nkeys() != 1 {
		t.Fatalf("expected 1 key, but got %d keys", rootNode.nkeys())
	}

	gotKey := rootNode.getKey(0)
	gotVal := rootNode.getVal(0)

	if string(gotKey) != "nishant" {
		t.Fatalf("expected key %q,but got %q ", "nishant", string(gotKey))
	}

	if string(gotVal) != "goodboy" {
		t.Fatalf("expected value %q,but got %q ", "goodboy", string(gotVal))
	}

	t.Skip("TODO: Implement together")
}

// ============================================================================
// TEST 2: UPDATE EXISTING KEY
// ============================================================================

// TestBTreeUpdate tests updating an existing key with a new value
func TestBTreeUpdate(t *testing.T) {
	// TODO: Let's implement this together!
	// What should happen when we insert the same key twice?
	// 1. Insert key="foo", value="bar"
	// 2. Insert key="foo", value="baz" (update)
	// 3. Check that we still have only 1 key (not 2)
	// 4. Check that the value is the new one ("baz")
	tree, pager := newTestBTree()

	key := []byte("apple")
	tree.Insert(key, []byte("red"))
	tree.Insert(key, []byte("green")) // Same key "apple", new value "green"

	rootNode := BNode(pager.Get(tree.root))

	if rootNode.nkeys() != 1 {

		t.Fatalf("expected 1 key after update,but got %d keys", rootNode.nkeys())

	}

	gotVal := rootNode.getVal(0)

	if string(gotVal) != "green" {
		t.Fatalf("expected value %q after update ,but got %q", "green", string(gotVal))
	}

	t.Skip("TODO: Implement together")
}

// ============================================================================
// TEST 3: MULTIPLE INSERTS (ORDERING)
// ============================================================================

// TestBTreeMultipleInserts tests inserting multiple keys and verifies they're sorted
func TestBTreeMultipleInserts(t *testing.T) {
	// TODO: Let's implement this together!
	// If we insert keys out of order, they should be stored sorted
	// 1. Insert: "zebra", "apple", "mango", "banana"
	// 2. Check we have 4 keys
	// 3. Verify they're in sorted order: "apple", "banana", "mango", "zebra"

	t.Skip("TODO: Implement together")
}

// ============================================================================
// TEST 4: NODE SPLIT (100+ KEYS)
// ============================================================================

// TestBTreeSplit tests that inserting many keys causes node splits
func TestBTreeSplit(t *testing.T) {
	// TODO: Let's implement this together!
	// When we insert enough keys, the leaf should split and create an internal root
	// 1. Insert 100+ keys (small keys/values)
	// 2. Check that root becomes INTERNAL (not LEAF anymore)
	// 3. Check that root has multiple children
	// 4. Verify all keys across all nodes are sorted

	t.Skip("TODO: Implement together")
}
