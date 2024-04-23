package statelessness

import (
	"bytes"
	"log"
	"testing"
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
	log.Println(commitUnflushed)
}

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
	log.Println(commitFlushed)
}
