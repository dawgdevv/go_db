# B-Tree Algorithm Documentation

## Overview

This document describes the B-tree implementation used in this database. The B-tree is a self-balancing tree data structure that maintains sorted data and allows efficient insertion, deletion, and search operations.

## Node Structure

### Node Layout in Memory

Each node is stored as a contiguous byte array with the following layout:

```
| Header (4 bytes) | Pointers (8*nkeys bytes) | Offsets (2*nkeys bytes) | KV Data (variable) |
```

**Header (4 bytes):**

- Bytes 0-1: Node type (BNODE_LEAF=2 or BNODE_INTERNAL=1)
- Bytes 2-3: Number of keys (nkeys)

**Pointers (8 bytes each):**

- For leaf nodes: Always 0 (unused)
- For internal nodes: Child page IDs (uint64)
- Array of nkeys pointers

**Offsets (2 bytes each):**

- Relative byte offsets to each KV pair in the KV data region
- First offset is always 0 (implicit)
- Array stores offsets for keys 1 through nkeys-1

**KV Data (variable length):**

- Concatenated key-value pairs
- Each KV pair format: `[keyLen(2)][key bytes][valLen(2)][val bytes]`

### Constants

- `BTREE_PAGE_SIZE = 4096` bytes
- `BTREE_MAX_KEY_SIZE = 1000` bytes
- `BTREE_MAX_VAL_SIZE = 3000` bytes

## Core Algorithms

### 1. Node Lookup (Binary Search)

**Purpose:** Find the position of a key in a sorted node.

**Algorithm:**

```
function nodeLookup(node, key):
    if node.nkeys == 0:
        return (0, false)

    lo = 0
    hi = node.nkeys

    while lo < hi:
        mid = lo + (hi - lo) / 2
        cmp = compare(node.getKey(mid), key)

        if cmp < 0:        // mid key < search key
            lo = mid + 1
        else if cmp > 0:   // mid key > search key
            hi = mid
        else:              // mid key == search key
            return (mid, true)

    // lo is insertion position or child pointer index
    return (lo, false)
```

**Returns:**

- `(index, true)` if key found at index
- `(index, false)` if key not found, where index is:
  - For leaf: insertion position
  - For internal: child pointer to follow

**Time Complexity:** O(log n) where n = number of keys in node

---

### 2. Node Append KV

**Purpose:** Append a key-value pair to a node at a specific index, recomputing offsets correctly.

**Algorithm:**

```
function nodeAppendKv(new, idx, ptr, key, val):
    // Set pointer
    new.setPtr(idx, ptr)

    // Compute offset for this KV based on previous entry
    if idx == 0:
        offset = 0
    else:
        prevPos = new.kvPos(idx - 1)
        prevKeyLen = read_uint16(new[prevPos])
        prevValLen = read_uint16(new[prevPos + 2 + prevKeyLen])
        offset = new.getOffset(idx - 1) + 4 + prevKeyLen + prevValLen

    // Store offset and write KV data
    new.setOffset(idx, offset)
    pos = new.kvPos(idx)

    write_uint16(new[pos], len(key))
    copy(new[pos + 2], key)
    write_uint16(new[pos + 2 + len(key)], len(val))
    copy(new[pos + 4 + len(key)], val)
```

**Key Points:**

- Offsets are recomputed from previous KV, not blindly copied
- Ensures proper layout even when building new nodes from scratch
- Each KV entry is self-contained with length prefixes

---

### 3. Node Append Range

**Purpose:** Copy a range of KV entries from one node to another, reconstructing layout.

**Algorithm:**

```
function nodeAppendRange(new, old, dstNew, srcOld, n):
    for i = 0 to n-1:
        ptr = old.getPtr(srcOld + i)
        key = old.getKey(srcOld + i)
        val = old.getVal(srcOld + i)

        // Update header to reflect current key count
        new.setHeader(old.btype(), dstNew + i + 1)

        // Append this KV, recomputing offset
        nodeAppendKv(new, dstNew + i, ptr, key, val)
```

**Key Points:**

- Does NOT blindly copy offsets or KV regions
- Reconstructs each entry to ensure correct layout
- Header's nkeys is updated incrementally as entries are added
- More expensive than bulk copy but guarantees correctness

---

### 4. Leaf Insert

**Purpose:** Insert a new key-value pair into a leaf node.

**Algorithm:**

```
function leafInsert(new, old, idx, key, val):
    assert(old is LEAF)
    assert(idx <= old.nkeys)

    // New node will have one more key
    new.setHeader(LEAF, old.nkeys + 1)

    // Copy keys before insertion point
    nodeAppendRange(new, old, 0, 0, idx)

    // Insert new key at idx
    nodeAppendKv(new, idx, 0, key, val)

    // Copy remaining keys (if any)
    if idx < old.nkeys:
        nodeAppendRange(new, old, idx+1, idx, old.nkeys - idx)

    assert(new.nbytes <= PAGE_SIZE)
```

**Cases:**

- `idx = 0`: Insert at beginning, copy all old keys after
- `idx = nkeys`: Append at end, no tail copy needed
- `0 < idx < nkeys`: Split copy around insertion point

---

### 5. Leaf Update

**Purpose:** Update an existing key's value in a leaf node.

**Algorithm:**

```
function leafUpdate(new, old, idx, key, val):
    assert(old is LEAF)
    assert(idx < old.nkeys)

    // Key count doesn't change
    new.setHeader(LEAF, old.nkeys)

    // Copy keys before update point
    nodeAppendRange(new, old, 0, 0, idx)

    // Write updated key-value
    nodeAppendKv(new, idx, 0, key, val)

    // Copy remaining keys (if any)
    if idx+1 < old.nkeys:
        nodeAppendRange(new, old, idx+1, idx+1, old.nkeys - (idx+1))

    assert(new.nbytes <= PAGE_SIZE)
```

---

### 6. Node Split (2-way)

**Purpose:** Split an oversized node into two nodes that fit in pages.

**Algorithm:**

```
function nodeSplit2(left, right, old):
    assert(old.nkeys >= 2)

    // Start with roughly half split
    nleft = old.nkeys / 2

    // Calculate size needed for right node
    calcRightBytes(split):
        nright = old.nkeys - split
        kvDataSize = old.nbytes - old.kvPos(split)
        return 4 + 8*nright + 2*nright + kvDataSize

    // Adjust split point until right fits in page
    while calcRightBytes(nleft) > PAGE_SIZE and nleft < old.nkeys - 1:
        nleft++

    nright = old.nkeys - nleft

    // Rebuild both nodes
    left.setHeader(old.btype, nleft)
    right.setHeader(old.btype, nright)
    nodeAppendRange(left, old, 0, 0, nleft)
    nodeAppendRange(right, old, 0, nleft, nright)

    assert(right.nbytes <= PAGE_SIZE)
```

**Key Points:**

- Adjusts split point based on actual byte sizes, not just key count
- Ensures right node fits in one page
- Left node may still be too large (handled by 3-way split)

---

### 7. Node Split (3-way)

**Purpose:** Handle cases where 2-way split isn't enough.

**Algorithm:**

```
function nodeSplit3(old):
    if old.nbytes <= PAGE_SIZE:
        return (1, [old, nil, nil])

    // Try 2-way split first
    left = allocate(PAGE_SIZE)
    right = allocate(PAGE_SIZE)
    nodeSplit2(left, right, old)

    if left.nbytes <= PAGE_SIZE:
        return (2, [left, right, nil])

    // Left is still too large, split it again
    leftleft = allocate(PAGE_SIZE)
    middle = allocate(PAGE_SIZE)
    nodeSplit2(leftleft, middle, left)

    assert(leftleft.nbytes <= PAGE_SIZE)
    return (3, [leftleft, middle, right])
```

**Returns:**

- Count of resulting nodes (1, 2, or 3)
- Array of up to 3 nodes

---

### 8. Internal Node Insert

**Purpose:** Insert a key-value into a child of an internal node, handling splits.

**Algorithm:**

```
function nodeInsert(tree, new, node, idx, key, val, found):
    assert(node is INTERNAL)
    assert(idx <= node.nkeys)

    // Get child pointer at idx
    kptr = node.getPtr(idx)

    // Recursively insert into child
    knode = tree.get(kptr)
    knode = treeInsert(tree, knode, key, val)

    // Split child if needed
    (nsplit, splitted) = nodeSplit3(knode)

    // Delete old child page
    tree.del(kptr)

    // Rebuild parent node based on split count
    switch nsplit:
        case 1:
            // Child didn't split, just replace pointer
            new.setHeader(INTERNAL, node.nkeys)
            nodeAppendRange(new, node, 0, 0, idx)
            nodeAppendKv(new, idx, tree.new(splitted[0]), splitted[0].getKey(0), nil)
            if idx+1 <= node.nkeys:
                nodeAppendRange(new, node, idx+1, idx+1, node.nkeys - (idx+1))

        case 2:
            // Child split into 2, add separator key
            nodeReplace2kid(new, node, idx,
                tree.new(splitted[0]), splitted[1].getKey(0))
            new.setPtr(idx+1, tree.new(splitted[1]))

        case 3:
            // Child split into 3, add two separator keys
            nodeReplace3kid(new, node, idx,
                tree.new(splitted[0]), splitted[1].getKey(0),
                tree.new(splitted[1]), splitted[2].getKey(0),
                tree.new(splitted[2]))
```

**Key Points:**

- Recursively inserts into appropriate child
- Handles child splits by adding separator keys
- May cause parent to grow and split as well

---

### 9. Replace 2 Kids

**Purpose:** Replace one child pointer with two, adding a separator key.

**Algorithm:**

```
function nodeReplace2kid(new, old, idx, ptr, key):
    assert(old is INTERNAL)
    assert(idx < old.nkeys)

    // New node has one more key
    new.setHeader(INTERNAL, old.nkeys + 1)

    // Copy entries before split point
    nodeAppendRange(new, old, 0, 0, idx)

    // Add first new child with separator key
    nodeAppendKv(new, idx, ptr, key, nil)

    // Add second new child (old pointer at idx, old key at idx)
    nodeAppendKv(new, idx+1, old.getPtr(idx), old.getKey(idx), nil)

    // Copy remaining entries
    if idx+1 < old.nkeys:
        nodeAppendRange(new, old, idx+2, idx+1, old.nkeys - (idx+1))
```

---

### 10. Replace 3 Kids

**Purpose:** Replace one child pointer with three, adding two separator keys.

**Algorithm:**

```
function nodeReplace3kid(new, old, idx, ptr1, key1, ptr2, key2, ptr3):
    assert(old is INTERNAL)

    // New node has two more keys
    new.setHeader(INTERNAL, old.nkeys + 2)

    // Copy entries before split point
    nodeAppendRange(new, old, 0, 0, idx)

    // Add three new children with separator keys
    nodeAppendKv(new, idx, ptr1, key1, nil)
    nodeAppendKv(new, idx+1, ptr2, key2, nil)
    nodeAppendKv(new, idx+2, ptr3, old.getKey(idx), nil)

    // Copy remaining entries
    if idx+1 < old.nkeys:
        nodeAppendRange(new, old, idx+3, idx+1, old.nkeys - (idx+1))

    assert(new.nbytes <= PAGE_SIZE)
```

---

### 11. Tree Insert (Recursive)

**Purpose:** Main insert logic that dispatches to leaf or internal insert.

**Algorithm:**

```
function treeInsert(tree, node, key, val):
    // Allocate scratch buffer (2x page size for safety)
    new = allocate(2 * PAGE_SIZE)

    // Find position for key
    (idx, found) = nodeLookup(node, key)

    switch node.btype:
        case LEAF:
            if found:
                leafUpdate(new, node, idx, key, val)
            else:
                leafInsert(new, node, idx, key, val)

        case INTERNAL:
            nodeInsert(tree, new, node, idx, key, val, found)

        default:
            panic("invalid node type")

    return new
```

---

### 12. Public Insert API

**Purpose:** User-facing insert operation that handles root splits.

**Algorithm:**

```
function BTree.Insert(key, val):
    assert(len(key) > 0 and len(key) <= MAX_KEY_SIZE)
    assert(len(val) <= MAX_VAL_SIZE)

    // Initialize empty tree
    if tree.root == 0:
        root = allocate(PAGE_SIZE)
        root.setHeader(LEAF, 0)
        tree.root = tree.new(root)
        return

    // Load and delete old root
    node = tree.get(tree.root)
    tree.del(tree.root)

    // Insert into tree
    node = treeInsert(tree, node, key, val)

    // Handle root split
    (nsplit, splitted) = nodeSplit3(node)

    if nsplit > 1:
        // Root split, create new internal root
        root = allocate(PAGE_SIZE)
        root.setHeader(INTERNAL, nsplit)

        for i = 0 to nsplit-1:
            ptr = tree.new(splitted[i])
            if i > 0:
                // Add separator key
                nodeAppendKv(root, i-1, ptr, splitted[i].getKey(0), nil)
            root.setPtr(i, ptr)

        tree.root = tree.new(root)
    else:
        // No split, just update root
        tree.root = tree.new(splitted[0])
```

**Key Points:**

- Handles first insertion (empty tree)
- Root splits increase tree height
- Old root is deleted before persisting new structure

---

## Complexity Analysis

### Time Complexity

- **Search/Lookup:** O(log n)

  - Binary search within each node: O(log k) where k = keys per node
  - Tree height: O(log_k n)
  - Total: O(log_k n × log k) = O(log n)

- **Insert:** O(log n)

  - Path traversal: O(log n)
  - Node modifications: O(k) per node
  - Total: O(k × log n) ≈ O(log n) for fixed k

- **Split:** O(k)
  - Linear scan to find split point
  - Linear copy of entries

### Space Complexity

- **Per Node:** O(k) where k = keys per node
- **Tree Height:** O(log_k n)
- **Insert Operation:** O(log n) stack space for recursion

---

## Invariants

1. **Node Size:** All nodes fit within BTREE_PAGE_SIZE (4096 bytes)
2. **Sorted Keys:** Keys within each node are sorted
3. **Leaf Depth:** All leaves are at the same depth
4. **Fanout:** Internal nodes have k children and k-1 keys
5. **Root:** May have fewer children/keys than minimum
6. **Offsets:** Always recomputed, never blindly copied
7. **Layout:** KV data region starts after header+pointers+offsets

---

## Design Decisions

### Why Recompute Offsets?

Blindly copying offsets from old nodes breaks when:

- Node headers change size (different nkeys)
- Entries are inserted/deleted in the middle
- Nodes are split or merged

Recomputing ensures correctness at the cost of performance.

### Why 2×PAGE_SIZE Scratch Buffers?

During insert operations, a node may temporarily exceed PAGE_SIZE before splitting. Using 2× buffers:

- Prevents buffer overflows
- Simplifies insert logic
- Final nodes are trimmed/split to fit

### Why Binary Search?

Linear scan risks:

- O(n) worst case per node
- Fragile break conditions
- Off-by-one errors

Binary search provides:

- O(log k) guaranteed performance
- Clear semantics for not-found cases
- Correct insertion positions

---

## Future Optimizations

1. **Bulk Copy:** When copying entire node ranges without modifications, bulk memcpy could replace nodeAppendRange
2. **Split Heuristics:** Better split point selection to balance nodes
3. **Sibling Merging:** Merge underfull nodes during deletion
4. **Prefix Compression:** Compress common key prefixes to save space
5. **Lazy Splits:** Defer splits until absolutely necessary

---

## References

- Original B-tree paper: Bayer & McCreight (1972)
- Modern B-tree textbook: "Database Internals" by Alex Petrov
- Implementation inspired by: "Build Your Own Database" tutorial series
