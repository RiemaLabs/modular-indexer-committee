package kvstore

import (
	"fmt"
	"testing"
)

func TestKVStore(t *testing.T) {
	bm, _ := NewByteMap("test.db")
	key := []byte("key1")
	value := []byte("value1")
	bm.Insert(key, value)

	retrievedValue, err := bm.Get(key)
	if err != nil {
		t.Fatalf("Error retrieving value: %v", err)
	}

	if string(retrievedValue) != string(value) {
		t.Fatalf("Expected value %s, got %s", value, retrievedValue)
	}

	fmt.Printf("Current Length of Map: %d\n", bm.Length())
	if bm.Length() != 1 {
		t.Fatalf("Expected length of 1")
	}

	bm.Delete(key)
	if _, err := bm.Get(key); err == nil {
		t.Fatalf("Key not deleted.")
	}
	if bm.Length() != 0 {
		t.Fatalf("Expected length of 0")
	}
}
