## Every package is used is go core package nothing external is used externally and it is implementd from scratch

| Package           | Why it's used                     |
| ----------------- | --------------------------------- |
| `encoding/binary` | Write/read integers in byte pages |
| `bytes`           | Compare keys, slice manipulation  |
| `os`              | File I/O, Open, Write, Sync       |
| `fmt`             | Debug / printing                  |
| `errors`          | Error handling                    |
| `sync`            | For locks later (if needed)       |
| `testing`         | Unit tests                        |
| `io`              | For some reader interfaces        |
| `math/rand`       | Random temp file names            |
| `path/filepath`   | Directory operations              |

---

## 🗺️ Low-Level Database Implementation Plan

### Current Status
✅ **Phase 1.1: B-Tree Node Structure** - COMPLETED
- Basic BNode serialization/deserialization (`internal/btree/node.go`)
- Header, pointer array, offset array, and KV access methods

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    KV API (pkg/kv)                          │
│              Get(key) / Put(key,val) / Delete(key)          │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌──────────────┐    ┌──────────────┐      ┌──────────────┐
│ Transaction  │    │   Storage    │      │     WAL      │
│   Manager    │◄───┤   Engine     │─────►│ Write-Ahead  │
│   (txn/)     │    │  (storage/)  │      │   Log (wal/) │
└──────────────┘    └──────────────┘      └──────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
┌──────────────┐    ┌──────────────┐   ┌──────────────┐
│    B-Tree    │    │  Buffer Pool │   │    Pager     │
│  (btree/)    │◄───┤ (bufferpool/)│◄──┤   (pager/)   │
└──────────────┘    └──────────────┘   └──────────────┘
        │                   │                   │
        └───────────────────┴───────────────────┘
                            ▼
                    ┌──────────────┐
                    │  Disk Files  │
                    │  (OS Layer)  │
                    └──────────────┘
```

---

### 📋 Implementation Roadmap

#### **PHASE 1: Core B-Tree Implementation** 🌳
*Foundation for all database operations*

- [x] **1.1 Node Serialization** (`internal/btree/node.go`)
  - BNode structure with header, pointers, offsets, KV pairs
  - Getter/setter methods for all node components

- [ ] **1.2 Node Operations** (`internal/btree/node.go`)
  - `nodeLookup(key)` - Binary search to find key position
  - `nodeAppendKV(idx, ptr, key, val)` - Insert KV at position
  - `nodeAppendRange(old, begin, end)` - Copy range of KVs
  
- [ ] **1.3 Node Split/Merge** (`internal/btree/operations.go`)
  - `nodeSplit2(left, right, old)` - Split node into 2
  - `nodeSplit3(left, middle, right, old)` - Split node into 3
  - `nodeMerge(target, left, right)` - Merge two nodes
  - `nodeReplace2Kid(new, old, idx, ptr, key)` - Replace child pointer

- [ ] **1.4 B-Tree Core Structure** (`internal/btree/btree.go`)
  ```go
  type BTree struct {
      root uint64                    // Root node page ID
      get  func(uint64) []byte       // Callback to load page
      new  func([]byte) uint64       // Callback to allocate new page
      del  func(uint64)               // Callback to free page
  }
  ```
  - `treeInsert(tree, node, key, val)` - Recursive insert
  - `treeDelete(tree, node, key)` - Recursive delete
  - `treeLookup(tree, node, key)` - Recursive search

- [ ] **1.5 Testing** (`tests/btree_test.go`)
  - Unit tests for all node operations
  - Integration tests for insert/delete/lookup
  - Edge cases: splits, merges, empty tree

**Learning Outcome**: Understand self-balancing trees, binary serialization, and how databases organize data on disk.

---

#### **PHASE 2: Pager - Disk Page Management** 💾
*Maps logical page IDs to physical disk locations*

- [ ] **2.1 Page Structure** (`internal/pager/pager.go`)
  ```go
  type Pager struct {
      file      *os.File              // Database file handle
      pageSize  int                   // Fixed page size (4096)
      numPages  uint64                // Total pages in file
      pageCache map[uint64][]byte     // Simple page cache
  }
  ```

- [ ] **2.2 Core Pager Operations**
  - `Open(filename)` - Open/create database file
  - `ReadPage(pageID)` - Read page from disk
  - `WritePage(pageID, data)` - Write page to disk
  - `AllocPage()` - Allocate new page ID
  - `FreePage(pageID)` - Mark page as free (freelist)
  - `Sync()` - fsync to ensure durability

- [ ] **2.3 Free Page Management** (`internal/pager/freelist.go`)
  - Maintain list of freed pages for reuse
  - Store freelist in special pages
  - Implement freelist allocation strategy

- [ ] **2.4 File Header** (`internal/pager/header.go`)
  ```
  [Magic(4)][Version(2)][PageSize(2)][NumPages(8)][Root(8)][Freelist(8)]
  ```
  - Database metadata in first page
  - Validate on open, update on close

**Learning Outcome**: How databases persist data, manage disk space, and handle file I/O.

---

#### **PHASE 3: Buffer Pool - Memory Management** 🎯
*Caches hot pages in memory with eviction policy*

- [ ] **3.1 Buffer Pool Structure** (`internal/bufferpool/pool.go`)
  ```go
  type BufferPool struct {
      pager     *Pager
      pool      map[uint64]*Frame     // Page ID -> Frame
      lru       *LRUList              // Eviction policy
      maxFrames int                   // Memory limit
      mutex     sync.RWMutex          // Thread safety
  }
  
  type Frame struct {
      pageID   uint64
      data     []byte
      dirty    bool                   // Needs write-back
      pinCount int                    // Reference count
  }
  ```

- [ ] **3.2 Buffer Pool Operations**
  - `FetchPage(pageID)` - Get page (from cache or disk)
  - `UnpinPage(pageID, dirty)` - Release page reference
  - `FlushPage(pageID)` - Write dirty page to disk
  - `FlushAll()` - Write all dirty pages
  - `EvictPage()` - LRU eviction when pool is full

- [ ] **3.3 LRU Eviction** (`internal/bufferpool/lru.go`)
  - Doubly-linked list for LRU tracking
  - Move to front on access
  - Evict from back when needed
  - Skip pinned pages

**Learning Outcome**: Memory management, caching strategies, and performance optimization.

---

#### **PHASE 4: Write-Ahead Log (WAL)** 📝
*Durability and crash recovery*

- [ ] **4.1 WAL Structure** (`internal/wal/wal.go`)
  ```go
  type WAL struct {
      file      *os.File
      offset    int64                 // Current write position
      lsn       uint64                // Log Sequence Number
  }
  
  type LogRecord struct {
      LSN       uint64
      TxnID     uint64
      Type      RecordType            // BEGIN, UPDATE, COMMIT, ABORT
      PageID    uint64
      Before    []byte                // Old data (for undo)
      After     []byte                // New data (for redo)
  }
  ```

- [ ] **4.2 WAL Operations**
  - `AppendLog(record)` - Write log entry
  - `Flush()` - Force log to disk (fsync)
  - `ReadLog()` - Read all log records
  - `Truncate(lsn)` - Remove old logs after checkpoint

- [ ] **4.3 Recovery** (`internal/wal/recovery.go`)
  - Scan WAL on database open
  - Redo committed transactions
  - Undo uncommitted transactions
  - Restore database to consistent state

- [ ] **4.4 Checkpointing**
  - Periodic flush of dirty pages
  - Write checkpoint record to WAL
  - Allow truncation of old WAL entries

**Learning Outcome**: ACID properties, crash recovery, and durability guarantees.

---

#### **PHASE 5: Transaction Manager** 🔒
*ACID transactions and concurrency control*

- [ ] **5.1 Transaction Structure** (`internal/txn/txn.go`)
  ```go
  type Transaction struct {
      id        uint64
      state     TxnState              // ACTIVE, COMMITTED, ABORTED
      readSet   map[uint64][]byte     // Pages read
      writeSet  map[uint64][]byte     // Pages modified
      wal       *WAL
      isolation IsolationLevel
  }
  
  type TxnManager struct {
      activeTxns map[uint64]*Transaction
      nextTxnID  uint64
      lockTable  *LockManager
  }
  ```

- [ ] **5.2 Transaction Operations**
  - `Begin()` - Start new transaction
  - `Commit()` - Apply changes and release locks
  - `Abort()` - Rollback changes
  - `Read(key)` - Read with isolation
  - `Write(key, val)` - Buffer write

- [ ] **5.3 Lock Manager** (`internal/txn/lock.go`)
  - Shared/Exclusive locks on pages/rows
  - Lock table with wait queues
  - Deadlock detection (timeout or graph)
  - Lock escalation

- [ ] **5.4 Isolation Levels**
  - Read Uncommitted (no locks)
  - Read Committed (release read locks early)
  - Repeatable Read (hold read locks)
  - Serializable (range locks)

**Learning Outcome**: Concurrency control, locking protocols, and transaction isolation.

---

#### **PHASE 6: Storage Engine** 🏗️
*Orchestrates all components*

- [ ] **6.1 Storage Manager** (`internal/storage/storage.go`)
  ```go
  type StorageEngine struct {
      btree      *BTree
      bufferPool *BufferPool
      pager      *Pager
      wal        *WAL
      txnMgr     *TxnManager
  }
  ```

- [ ] **6.2 Integration**
  - Wire B-Tree callbacks to buffer pool
  - Coordinate WAL writes before page writes
  - Manage transaction lifecycle
  - Handle startup/shutdown

- [ ] **6.3 Crash Recovery Flow**
  1. Open database file via pager
  2. Read file header for metadata
  3. Initialize buffer pool
  4. Replay WAL for recovery
  5. Load B-Tree root page
  6. Start accepting operations

**Learning Outcome**: System integration and lifecycle management.

---

#### **PHASE 7: KV API & Utilities** 🔌
*User-facing interface*

- [ ] **7.1 KV Store Interface** (`pkg/kv/kv.go`)
  ```go
  type KVStore struct {
      storage *StorageEngine
  }
  
  func (kv *KVStore) Get(key []byte) ([]byte, error)
  func (kv *KVStore) Put(key, val []byte) error
  func (kv *KVStore) Delete(key []byte) error
  func (kv *KVStore) Scan(start, end []byte) Iterator
  ```

- [ ] **7.2 Iterator** (`pkg/kv/iterator.go`)
  - Range scans over B-Tree
  - Prefix matching
  - Cursor-based iteration

- [ ] **7.3 Utilities** (`internal/util/`)
  - Key comparisons
  - Byte slice helpers
  - Debug printing/visualization

- [ ] **7.4 Encoding** (`internal/encoding/`)
  - Value encoding (int, string, json)
  - Schema-less or schema support
  - Compression (optional)

**Learning Outcome**: API design and usability.

---

#### **PHASE 8: Advanced Features** 🚀
*Optional enhancements*

- [ ] **8.1 LSM Tree** (`internal/lsm/`)
  - Write-optimized alternative to B-Tree
  - MemTable + SSTable design
  - Compaction strategies
  - Bloom filters

- [ ] **8.2 Optimization**
  - Prefix compression in B-Tree
  - Bulk loading
  - Parallel scans
  - Adaptive page splitting

- [ ] **8.3 Observability**
  - Statistics (page reads/writes, cache hit rate)
  - Query profiling
  - Storage usage metrics

**Learning Outcome**: Performance tuning and advanced data structures.

---

#### **PHASE 9: Testing & Examples** ✅
*Validation and documentation*

- [ ] **9.1 Comprehensive Tests** (`tests/`)
  - Unit tests for each component
  - Integration tests
  - Crash recovery tests
  - Concurrent transaction tests
  - Benchmarks

- [ ] **9.2 Example Programs** (`examples/`)
  - Basic CRUD operations
  - Concurrent access demo
  - Bulk import
  - Recovery simulation

- [ ] **9.3 CLI Tool** (`cmd/db/`)
  - Interactive shell
  - Execute commands
  - Inspect database internals
  - Performance testing

**Learning Outcome**: Quality assurance and real-world usage.

---

### 🎯 Key Concepts to Understand

| Component      | Key Concepts                                                    |
|----------------|-----------------------------------------------------------------|
| **B-Tree**     | Self-balancing, node splits/merges, binary search              |
| **Pager**      | Page-based storage, file I/O, space management                 |
| **Buffer Pool**| Caching, LRU eviction, dirty pages, memory limits              |
| **WAL**        | Write-ahead logging, redo/undo, crash recovery                 |
| **Transactions**| ACID, isolation levels, 2PL, deadlock detection               |
| **Storage**    | System integration, durability, consistency                    |

---

### 📚 Learning Path

1. **Start with B-Tree**: Understand how data is organized
2. **Add Pager**: Learn how memory maps to disk
3. **Build Buffer Pool**: Optimize with caching
4. **Implement WAL**: Ensure durability
5. **Add Transactions**: Handle concurrent access
6. **Integrate Everything**: Build the full system

---

### 🔍 Debugging Tips

- **Visualize B-Tree**: Print tree structure after operations
- **Track Page I/O**: Log every read/write to understand performance
- **Test Recovery**: Crash database randomly and verify recovery
- **Use Small Page Sizes**: Easier to debug with smaller pages initially
- **Add Assertions**: Validate invariants (node size, key ordering, etc.)

---

### 📖 Recommended Resources

- **Books**: "Database Internals" by Alex Petrov, "Designing Data-Intensive Applications" by Martin Kleppmann
- **Papers**: "ARIES" recovery algorithm, "B-Tree vs LSM-Tree" comparisons
- **Open Source**: Study SQLite, RocksDB, BoltDB source code
- **Practice**: Implement each phase incrementally, test thoroughly

---

### 🎓 What You'll Learn

By building this database from scratch, you'll deeply understand:

1. **Data Structures**: How B-Trees work at a byte level
2. **Systems Programming**: File I/O, memory management, concurrency
3. **Database Internals**: Storage engines, query execution, transactions
4. **Performance**: Caching, indexing, optimization techniques
5. **Reliability**: Crash recovery, durability, consistency guarantees

This is not just a database—it's a journey into the heart of systems programming! 🚀


