package uart

import (
	//"fmt"
	"reflect"
	"runtime"
	"unsafe"
)

// Helper function to get total allocated bytes for an object and its children
func getObjectSize(obj interface{}) int64 {
	// Get memory stats before
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Force obj to be allocated if it isn't already
	runtime.KeepAlive(obj)

	// Get memory stats after
	runtime.ReadMemStats(&m2)

	// Return the difference in allocated bytes
	return int64(m2.TotalAlloc - m1.TotalAlloc)
}

// deepSize attempts to get a more accurate size including pointed-to data
func deepSize(v interface{}) (size uintptr) {
	if v == nil {
		return 0
	}

	size = unsafe.Sizeof(v)

	switch x := v.(type) {
	case *Tree:
		//vv("deepSize Tree")
		size += deepSize(x.root)
		return
	case *inner:
		//vv("deepSize inner")
		size += deepSize(x.compressed)
		size += deepSize(x.Node)
		return
	case *Leaf:
		size += deepSize(x.Key)
		size += deepSize(x.Value)
		return
	case *bnode:
		//vv("deepSize bnode")
		if x.isLeaf {
			size += deepSize(x.leaf)
		} else {
			size += deepSize(x.inner)
		}
		return
	case *node4:
		for _, ch := range x.children {
			if ch != nil {
				size += deepSize(ch)
			}
		}
	case *node16:
		for _, ch := range x.children {
			if ch != nil {
				size += deepSize(ch)
			}
		}
	case *node48:
		for _, ch := range x.children {
			if ch != nil {
				size += deepSize(ch)
			}
		}
	case *node256:
		for _, ch := range x.children {
			if ch != nil {
				size += deepSize(ch)
			}
		}
		return
	}

	// Use reflect to examine the structure
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return size
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.String:
		// For strings, we need to add the actual string data size
		// String headers are 16 bytes on 64-bit systems (2 words)
		// but we also need to account for the actual string data
		size += uintptr(val.Len())
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			if val.Field(i).CanInterface() {
				size += deepSize(val.Field(i).Interface())
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			if val.Index(i).CanInterface() {
				size += deepSize(val.Index(i).Interface())
			}
		}
	case reflect.Map:
		iter := val.MapRange()
		for iter.Next() {
			if iter.Key().CanInterface() {
				size += deepSize(iter.Key().Interface())
			}
			if iter.Value().CanInterface() {
				size += deepSize(iter.Value().Interface())
			}
		}
	}

	return size
}

/*
func main() {
	// Example struct with pointers
	type Node struct {
		Data    string
		Next    *Node
		Numbers []int
		Map     map[string]int
	}

	node := &Node{
		Data:    "test",
		Numbers: make([]int, 100),
		Map:     make(map[string]int),
	}

	// Basic size using unsafe.Sizeof
	fmt.Printf("Basic size: %d bytes\n", unsafe.Sizeof(*node))

	// Deep size including pointed-to data
	fmt.Printf("Deep size: %d bytes\n", deepSize(node))

	// Runtime allocation measurement
	size := getObjectSize(node)
	fmt.Printf("Total allocated size: %d bytes\n", size) // 0
}
*/
