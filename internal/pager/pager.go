package pager

import "os"

type pager struct {
	file      *os.File
	pageSize  int
	numpages  uint64
	pagecache map[uint64][]byte
}

func Open(fileName *os.File, pageSize int) *pager {

	return nil
}
