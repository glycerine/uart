package uart

import (
	"bytes"
	"testing"
)

func TestNewArtTreeFromSortedNoCopy(t *testing.T) {
	keys := []Key{
		Key(""),
		Key("a"),
		Key("aa"),
		Key("ab"),
		Key("b"),
		Key("long/common/prefix/0"),
		Key("long/common/prefix/1"),
	}

	items := make([]BulkItem, len(keys))
	want := NewArtTree()
	want.SkipLocking = true
	for i, key := range keys {
		items[i] = BulkItem{Key: key, Value: key}
		want.InsertNoCopy(key, key)
	}

	got := NewArtTreeFromSortedNoCopy(items)
	got.SkipLocking = true
	if got.Size() != len(keys) {
		t.Fatalf("got size %d, want %d", got.Size(), len(keys))
	}

	var gotKeys []Key
	got.ScanLeaves(func(lf *Leaf) bool {
		gotKeys = append(gotKeys, lf.Key)
		return true
	})
	if len(gotKeys) != len(keys) {
		t.Fatalf("scan got %d keys, want %d", len(gotKeys), len(keys))
	}
	for i := range keys {
		if !bytes.Equal(gotKeys[i], keys[i]) {
			t.Fatalf("scan[%d] got %q, want %q", i, gotKeys[i], keys[i])
		}
		val, idx, found := got.FindExact(keys[i])
		if !found {
			t.Fatalf("FindExact(%q) did not find key", keys[i])
		}
		if idx != i {
			t.Fatalf("FindExact(%q) idx=%d, want %d", keys[i], idx, i)
		}
		if !bytes.Equal(val.(Key), keys[i]) {
			t.Fatalf("FindExact(%q) value=%q, want %q", keys[i], val, keys[i])
		}
	}
}
