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
}

// ============================================================================
// TEST 2: UPDATE EXISTING KEY
// ============================================================================

// TestBTreeUpdate tests updating an existing key with a new value
func TestBTreeUpdate(t *testing.T) {
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
}

// ============================================================================
// TEST 3: MULTIPLE INSERTS (ORDERING)
// ============================================================================

// TestBTreeMultipleInserts tests inserting multiple keys and verifies they're sorted
func TestBTreeMultipleInserts(t *testing.T) {
	tree, pager := newTestBTree()

	keys := [][]byte{
		[]byte("zebra"),
		[]byte("apple"),
		[]byte("mango"),
		[]byte("banana"),
	}

	for _, key := range keys {
		tree.Insert(key, []byte{})
	}

	rootNode := BNode(pager.Get(tree.root))

	if rootNode.nkeys() != 4 {
		t.Fatalf("expected 4 keys,but got %d keys", rootNode.nkeys())
	}

	expectedOrder := []string{"apple", "banana", "mango", "zebra"}

	for i, expectedKey := range expectedOrder {
		gotKey := rootNode.getKey(uint16(i))
		if string(gotKey) != expectedKey {
			t.Fatalf("at index %d, expected key %q,but got %q", i, expectedKey, string(gotKey))
		}
	}
}

// ============================================================================
// TEST 4: NODE SPLIT (100+ KEYS)
// ============================================================================

// TestBTreeSplit tests that inserting many keys causes node splits
func TestBTreeSplit(t *testing.T) {
	tree, pager := newTestBTree()

	// Insert enough keys to trigger a split (need > 178 keys with "key###" + "val" format)
	for i := 0; i < 200; i++ {
		key := []byte(fmt.Sprintf("key%03d", i))
		val := []byte("val")
		tree.Insert(key, val)
	}

	rootNode := BNode(pager.Get(tree.root))

	if rootNode.btype() != BNODE_INTERNAL {
		t.Fatalf("expected root to be INTERNAL after splits, but got type %d", rootNode.btype())
	}

	if rootNode.nkeys() < 1 {
		t.Fatalf("expected root to have at least 1 key after splits, but got %d keys", rootNode.nkeys())
	}
	// Verify all keys are sorted across the tree
	var allKeys []string

	var collectKeys func(nodeID uint64)
	collectKeys = func(nodeID uint64) {
		node := BNode(pager.Get(nodeID))
		if node.btype() == BNODE_LEAF {
			for i := uint16(0); i < node.nkeys(); i++ {
				allKeys = append(allKeys, string(node.getKey(i)))
			}
		} else {
			for i := uint16(0); i < node.nkeys(); i++ {
				childPtr := node.getPtr(i)
				collectKeys(uint64(childPtr))
				allKeys = append(allKeys, string(node.getKey(i)))
			}
			// Collect keys from the last child
			childPtr := node.getPtr(node.nkeys())
			collectKeys(uint64(childPtr))
		}
	}

	collectKeys(tree.root)

	for i := 1; i < len(allKeys); i++ {
		if allKeys[i-1] > allKeys[i] {
			t.Fatalf("keys are not sorted: %q came before %q", allKeys[i-1], allKeys[i])
		}
	}
}
