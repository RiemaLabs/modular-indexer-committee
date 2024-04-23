package statelessness

import (
	"bytes"
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

	tree.Flush()
	if _, err := tree.GetUnresolved(key); err == nil {
		t.Fatal(err)
	}
	if v, err := tree.Get(key); err != nil || !bytes.Equal(val, v) {
		t.Fatal(err, v)
	}
}
