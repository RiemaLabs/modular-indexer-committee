package statelessness

import (
	"fmt"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-verkle"
)

type Tree struct {
	tree verkle.VerkleNode
	db   *pebble.DB
}

func NewTree() (*Tree, error) {
	db, err := pebble.Open("verkle.pebble", new(pebble.Options))
	if err != nil {
		return nil, err
	}
	return &Tree{tree: verkle.New(), db: db}, nil
}

func (t *Tree) Get(key []byte) ([]byte, error) {
	return t.tree.Get(key, t.restoreNode)
}

func (t *Tree) GetUnresolved(key []byte) (val []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	return t.tree.Get(key, nil)
}

func (t *Tree) Insert(key, value []byte) error {
	return t.tree.Insert(key, value, t.restoreNode)
}

func (t *Tree) Flush() {
	// TODO: When to flush, what to flush?
	if node, ok := t.tree.(*verkle.InternalNode); ok {
		node.Flush(t.storeNode)
	}
}

func (t *Tree) restoreNode(key []byte) ([]byte, error) {
	v, closer, err := t.db.Get(key)
	if err != nil {
		return nil, err
	}
	defer func() { _ = closer.Close() }()
	ret := v // make a copy because closer would invalidate the data
	return ret, nil
}

func (t *Tree) storeNode(key []byte, node verkle.VerkleNode) {
	val, err := node.Serialize()
	if err != nil {
		panic(err) // TODO
	}
	if err := t.db.Set(key, val, new(pebble.WriteOptions)); err != nil {
		panic(err) // TODO
	}
}
