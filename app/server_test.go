package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestGzip(t *testing.T) {
	s := "abc"
	bytesToCompress := []byte(s)
	expected, _ := hex.DecodeString("1F8B08000000000000034B4C4A0600C241243503000000")

	result, _ := Compress(bytesToCompress)

	fmt.Printf("Compressed string: %s\nCompressed string decompressed (should be %s): %s\n", string(result[:]), s, string(Decompress(result)[:]))

	if !bytes.Equal(expected, result) {
		t.Errorf("Compression failed\nEXPECTED:\n%s\n\nGOT:\n%s", hex.EncodeToString(expected), hex.EncodeToString(result))
	}
}
