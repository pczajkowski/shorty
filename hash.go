package main

import (
	"fmt"
	"hash/crc32"
)

var crc32q *crc32.Table = crc32.MakeTable(0xD5828281)

func getHash(link string) string {
	return fmt.Sprintf("%08x", crc32.Checksum([]byte(link), crc32q))
}
