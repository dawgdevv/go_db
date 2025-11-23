# B-Tree Node Layout Documentation

## Overview

This document describes the corrected and consistent B-Tree node layout implementation.

## Node Layout Structure

### Common Header (4 bytes)

- **Bytes 0-1**: Node type (`btype`)
  - `1` = BNODE_INTERNAL (internal node)
  - `2` = BNODE_LEAF (leaf node)
- **Bytes 2-3**: Number of keys (`nkeys`)

### Pointer Array

- **Leaf nodes**: `nkeys` pointer slots (8 bytes each, unused/always 0)
- **Internal nodes**: `nkeys + 1` pointer slots (8 bytes each)
  - Stores child page IDs
  - For `n` keys, we have `n+1` children
  - Child[i] contains keys < Key[i]
  - Child[nkeys] contains keys >= Key[nkeys-1]

### Offset Array

- **Size**: `2 * nkeys` bytes
- Stores byte offsets for each key-value pair in the KV area
- Offset for key 0 is implicit (always 0)
- Offsets for keys 1..nkeys-1 are explicitly stored

### Key-Value Area

- Variable-length data for each key-value pair
- **Format per entry**:
  ```
  [2 bytes: keyLen][keyLen bytes: key][2 bytes: valLen][valLen bytes: value]
  ```
- **Leaf nodes**: Store actual key-value pairs with non-empty values
- **Internal nodes**: Store separator keys with empty values (valLen = 0)

## Size Calculation

Total node size = `4 + 8*ptrSlots + 2*nkeys + kvDataSize`

Where:

- `ptrSlots = nkeys` for leaf nodes
- `ptrSlots = nkeys + 1` for internal nodes

## Key APIs

### Node Type Helpers

- `ptrSlots()` - Returns number of pointer slots (nkeys or nkeys+1)
- `nbytes()` - Calculates total bytes used by the node
- `kvPos(idx)` - Returns byte position of key-value pair at index

### Leaf Node Operations

- `leafAppendKv(new, idx, key, val)` - Append key-value to leaf
- `leafAppendRange(new, old, dstNew, srcOld, n)` - Copy range of entries

### Internal Node Operations

- `internalAppendKv(new, idx, key)` - Append separator key (no value)
- `internalAppendRange(new, old, dstNew, srcOld, n)` - Copy keys and pointers
  - Copies `n` keys and `n+1` child pointers

## Critical Invariants

1. **Header Stability**: Callers must set the final `nkeys` via `setHeader()` before any append operations
2. **Internal Node Children**: Always maintain `nkeys + 1` child pointers
3. **Separator Keys**: Internal node key[i] separates children[i] and children[i+1]
4. **Scratch Buffer**: Operations use 2×PAGE_SIZE scratch buffers to allow temporary overflow before splitting

## Node Split Strategy

1. **Single node** (`nsplit=1`): Node fits in PAGE_SIZE, no split needed
2. **Two-way split** (`nsplit=2`): Node split into two PAGE_SIZE nodes
3. **Three-way split** (`nsplit=3`): Node split into three PAGE_SIZE nodes
   - Occurs when initial 2-way split leaves the left node too large

When creating a new root after split:

- Root has `nsplit` children (pointers 0..nsplit-1)
- Root has `nsplit-1` separator keys
- Key[i] = first key of child[i+1]

## Test Coverage

All tests pass:

- ✅ `TestBTreeInsertSingle` - Single key insertion
- ✅ `TestBTreeUpdate` - Key update (same key, new value)
- ✅ `TestBTreeMultipleInserts` - Multiple keys with correct ordering
- ✅ `TestBTreeSplit` - Large insertions causing node splits and internal root creation

## Example: 2-Way Split Root

After inserting enough keys to cause a split:

```
Root (INTERNAL, nkeys=1):
  Pointer[0] -> Left Child
  Key[0] = "key100" (separator)
  Pointer[1] -> Right Child
```

Left and Right children are LEAF nodes containing the split data.
