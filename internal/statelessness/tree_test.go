package statelessness

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"runtime"
	"testing"

	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
	"github.com/ethereum/go-verkle"
)

func TestTree_Insert(t *testing.T) {
	key := bytes.Repeat([]byte("a"), 32)
	val := bytes.Repeat([]byte("a"), 32)

	tree, err := NewTree()
	if err != nil {
		t.Fatal(err)
	}

	if err := tree.Insert(key, val); err != nil {
		t.Fatal(err)
	}
	if v, err := tree.Get(key); err != nil || !bytes.Equal(val, v) {
		t.Fatal(err, v)
	}
	commit := tree.Commit()

	tree.Flush()
	if b := tree.Commit(); !b.Equal(commit) {
		t.Fatal(commit, b)
	}

	if _, err := tree.GetUnresolved(key); err == nil {
		t.Fatal(err)
	}
	if v, err := tree.Get(key); err != nil || !bytes.Equal(val, v) {
		t.Fatal(err, v)
	}
	b := tree.Commit()
	log.Println("commit", commit)
	log.Println("b", b)
	if !b.Equal(commit) {
		t.Fatal(commit, b)
	}
}

// Get tree.insert(k1, v1).insert(k2, v2).commit()
func TestTree_InsertUnflushed(t *testing.T) {
	key1 := bytes.Repeat([]byte("a"), 32)
	val1 := bytes.Repeat([]byte("a"), 32)
	key2 := bytes.Repeat([]byte("b"), 32)
	val2 := bytes.Repeat([]byte("b"), 32)

	tree, err := NewTree()
	if err != nil {
		t.Fatal(err)
	}
	if err := tree.Insert(key1, val1); err != nil {
		t.Fatal(err)
	}
	if err := tree.Insert(key2, val2); err != nil {
		t.Fatal(err)
	}
	commitUnflushed := tree.Commit()
	log.Println(*commitUnflushed)
}

// Get tree.insert(k1, v1).flush().insert(k2, v2).commit(), and find it's different with the above commit
func TestTree_InsertFlushed(t *testing.T) {
	key1 := bytes.Repeat([]byte("a"), 32)
	val1 := bytes.Repeat([]byte("a"), 32)
	key2 := bytes.Repeat([]byte("b"), 32)
	val2 := bytes.Repeat([]byte("b"), 32)

	treeFlushed, err := NewTree()
	if err != nil {
		t.Fatal(err)
	}
	if err := treeFlushed.Insert(key1, val1); err != nil {
		t.Fatal(err)
	}
	treeFlushed.Flush()
	if err := treeFlushed.Insert(key2, val2); err != nil {
		t.Fatal(err)
	}
	commitFlushed := treeFlushed.Commit()
	log.Println(*commitFlushed)
}

func TestTree_Commit(t *testing.T) {
	const dir = "testCommit.verkle.pebble"
	_ = os.RemoveAll(dir)

	key1 := bytes.Repeat([]byte("a"), 32)
	val1 := bytes.Repeat([]byte("a"), 32)
	key2 := bytes.Repeat([]byte("b"), 32)
	val2 := bytes.Repeat([]byte("b"), 32)

	memTree0 := verkle.New()
	if err := memTree0.Insert(key1, val1, nil); err != nil {
		t.Fatal(err)
	}
	if err := memTree0.Insert(key2, val2, nil); err != nil {
		t.Fatal(err)
	}
	c0 := memTree0.Commit()

	flushTree1, err := NewTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := flushTree1.Insert(key1, val1); err != nil {
		t.Fatal(err)
	}
	flushTree1.Flush()
	if err := flushTree1.Insert(key2, val2); err != nil {
		t.Fatal(err)
	}
	c1 := flushTree1.Commit()

	if !c0.Equal(c1) {
		t.Fatal(c0, c1)
	}
}

// Get the memory usage of previous method, cost 7667 MB when size = 1000000
func TestTree_MemoryUnflushed(t *testing.T) {
	root := verkle.New()
	size := 1000000
	for _ = range size {
		randomKey := make([]byte, 32)
		randomVal := make([]byte, 32)

		_, err := rand.Read(randomKey)
		if err != nil {
			panic(err)
		}
		_, err = rand.Read(randomVal)
		if err != nil {
			panic(err)
		}
		root.Insert(randomKey, randomVal, stateless.NodeResolveFn)
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("Used memory = %v MiB", m.Alloc/1024/1024)
}

// Get memory usage of statelessness.tree with flushed method, cost 358 MB when size = 1000000
func TestTree_MemoryFlushed(t *testing.T) {
	tree, err := NewTree()
	if err != nil {
		t.Fatal(err)
	}
	size := 1000000
	for _ = range size {
		randomKey := make([]byte, 32)
		randomVal := make([]byte, 32)

		_, err := rand.Read(randomKey)
		if err != nil {
			panic(err)
		}
		_, err = rand.Read(randomVal)
		if err != nil {
			panic(err)
		}
		tree.Insert(randomKey, randomVal)
		tree.Flush()
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("Used memory = %v MiB", m.Alloc/1024/1024)
}
