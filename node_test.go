package bolt

import (
	_ "fmt"
	"testing"
	"unsafe"
)

// Ensure that a node can insert a key/value.
func TestNode_put(t *testing.T) {
	n := &node{inodes: make(inodes, 0), bucket: &Bucket{tx: &Tx{meta: &meta{pgid: 1}}}}
	n.put([]byte("baz"), []byte("baz"), []byte("2"), 0, 0)
	n.put([]byte("foo"), []byte("foo"), []byte("0"), 0, 0)
	n.put([]byte("hi"), []byte("hi"), []byte("4"), 0, 0)
	n.put([]byte("bar"), []byte("bar"), []byte("1"), 0, 0)
	n.put([]byte("foo"), []byte("foo"), []byte("3"), 0, leafPageFlag)
	n.put([]byte("hi"), []byte("hello"), []byte("4"), 0, 0)

	if len(n.inodes) != 4 {
		t.Fatalf("exp=4; got=%d", len(n.inodes))
	}
	if k, v := n.inodes[0].key, n.inodes[0].value; string(k) != "bar" || string(v) != "1" {
		t.Fatalf("exp=<bar,1>; got=<%s,%s>", k, v)
	}
	if k, v := n.inodes[1].key, n.inodes[1].value; string(k) != "baz" || string(v) != "2" {
		t.Fatalf("exp=<baz,2>; got=<%s,%s>", k, v)
	}
	if k, v := n.inodes[2].key, n.inodes[2].value; string(k) != "foo" || string(v) != "3" {
		t.Fatalf("exp=<foo,3>; got=<%s,%s>", k, v)
	}
	if k, v := n.inodes[3].key, n.inodes[3].value; string(k) != "hello" || string(v) != "4" {
		t.Fatalf("exp=<hello,4>; got=<%s,%s>", k, v)
	}
	if n.inodes[2].flags != uint32(leafPageFlag) {
		t.Fatalf("not a leaf: %d", n.inodes[2].flags)
	}
	// n.dump()
}

// Ensure that a node can deserialize from a leaf page.
func TestNode_read_LeafPage(t *testing.T) {
	// Create a page.
	var buf [4096]byte
	page := (*page)(unsafe.Pointer(&buf[0]))
	page.flags = leafPageFlag
	page.count = 3

	// Insert 3 elements at the beginning. sizeof(leafPageElement) == 16
	// fmt.Printf("page: %v, %[1]T\n", uintptr(unsafe.Pointer(page)))
	// fmt.Printf("page.ptr: %v, %[1]T\n", page.ptr)
	// fmt.Printf("&page.ptr: %v, %[1]T\n", &page.ptr)
	nodes := (*[4]leafPageElement)(unsafe.Pointer(&page.ptr))
	// fmt.Printf("page.ptr: %v, %[1]T\n", page.ptr)
	// fmt.Printf("&page.ptr: %v, %[1]T\n", &page.ptr)
	nodes[0] = leafPageElement{flags: 0, pos: 48, ksize: 3, vsize: 4} // pos = sizeof(leafPageElement) * 3
	// fmt.Printf("0 page.ptr: %v, %[1]T\n", page.ptr)
	nodes[1] = leafPageElement{flags: 0, pos: 39, ksize: 10, vsize: 3} // pos = sizeof(leafPageElement) * 2 + 3 + 4
	// fmt.Printf("1 page.ptr: %v, %[1]T\n", page.ptr)
	nodes[2] = leafPageElement{flags: 0, pos: 36, ksize: 4, vsize: 6} // pos = sizeof(leafPageElement) + 3 + 4 + 10 + 3
	// fmt.Printf("2 page.ptr: %v, %[1]T\n", page.ptr)
	// fmt.Printf("page.ptr: %v, %[1]T\n", page.ptr)
	// fmt.Printf("&page.ptr: %v, %[1]T\n", &page.ptr)

	// Write data for the nodes at the end.
	data := (*[4096]byte)(unsafe.Pointer(&nodes[3]))
	copy(data[:], []byte("barfooz"))
	copy(data[7:], []byte("helloworldbye"))
	copy(data[20:], []byte("hahawohaha"))

	// Deserialize page into a leaf.
	n := &node{}
	n.read(page)
	// n.dump()

	// Check that there are two inodes with correct data.
	if !n.isLeaf {
		t.Fatal("expected leaf")
	}
	if len(n.inodes) != 3 {
		t.Fatalf("exp=2; got=%d", len(n.inodes))
	}
	if k, v := n.inodes[0].key, n.inodes[0].value; string(k) != "bar" || string(v) != "fooz" {
		t.Fatalf("exp=<bar,fooz>; got=<%s,%s>", k, v)
	}
	if k, v := n.inodes[1].key, n.inodes[1].value; string(k) != "helloworld" || string(v) != "bye" {
		t.Fatalf("exp=<helloworld,bye>; got=<%s,%s>", k, v)
	}
	if k, v := n.inodes[2].key, n.inodes[2].value; string(k) != "haha" || string(v) != "wohaha" {
		t.Fatalf("exp=<haha,wohaha>; got=<%s,%s>", k, v)
	}
}

// Ensure that a node can serialize into a leaf page.
func TestNode_write_LeafPage(t *testing.T) {
	// Create a node.
	n := &node{isLeaf: true, inodes: make(inodes, 0), bucket: &Bucket{tx: &Tx{db: &DB{}, meta: &meta{pgid: 1}}}}
	n.put([]byte("susy"), []byte("susy"), []byte("que"), 0, 0)
	n.put([]byte("ricki"), []byte("ricki"), []byte("lake"), 0, 0)
	n.put([]byte("john"), []byte("john"), []byte("johnson"), 0, 0)

	// n.dump()
	// Write it to a page.
	var buf [4096]byte
	p := (*page)(unsafe.Pointer(&buf[0]))
	n.write(p)

	// Read the page back in.
	n2 := &node{}
	n2.read(p)

	// Check that the two pages are the same.
	if len(n2.inodes) != 3 {
		t.Fatalf("exp=3; got=%d", len(n2.inodes))
	}
	if k, v := n2.inodes[0].key, n2.inodes[0].value; string(k) != "john" || string(v) != "johnson" {
		t.Fatalf("exp=<john,johnson>; got=<%s,%s>", k, v)
	}
	if k, v := n2.inodes[1].key, n2.inodes[1].value; string(k) != "ricki" || string(v) != "lake" {
		t.Fatalf("exp=<ricki,lake>; got=<%s,%s>", k, v)
	}
	if k, v := n2.inodes[2].key, n2.inodes[2].value; string(k) != "susy" || string(v) != "que" {
		t.Fatalf("exp=<susy,que>; got=<%s,%s>", k, v)
	}
}

// Ensure that a node can split into appropriate subgroups.
func TestNode_split(t *testing.T) {
	// Create a node.
	n := &node{inodes: make(inodes, 0), bucket: &Bucket{tx: &Tx{db: &DB{}, meta: &meta{pgid: 1}}}}
	n.put([]byte("00000001"), []byte("00000001"), []byte("0123456701234567"), 0, 0)
	n.put([]byte("00000002"), []byte("00000002"), []byte("0123456701234567"), 0, 0)
	n.put([]byte("00000003"), []byte("00000003"), []byte("0123456701234567"), 0, 0)
	n.put([]byte("00000004"), []byte("00000004"), []byte("0123456701234567"), 0, 0)
	n.put([]byte("00000005"), []byte("00000005"), []byte("0123456701234567"), 0, 0)
	n.put([]byte("00000006"), []byte("00000006"), []byte("0123456701234567"), 0, 0)
	n.put([]byte("00000007"), []byte("00000007"), []byte("0123456701234567"), 0, 0)

	// Split between 2 & 3, 4 & 5.
	// According to 100 * 10% size to split default.
	n.split(100)

	var parent = n.parent
	if len(parent.children) != 3 {
		t.Fatalf("exp=2; got=%d", len(parent.children))
	}
	if len(parent.children[0].inodes) != 2 {
		t.Fatalf("exp=2; got=%d", len(parent.children[0].inodes))
	}
	if len(parent.children[1].inodes) != 2 {
		t.Fatalf("exp=3; got=%d", len(parent.children[1].inodes))
	}
	if len(parent.children[2].inodes) != 3 {
		t.Fatalf("exp=3; got=%d", len(parent.children[2].inodes))
	}
}

// Ensure that a page with the minimum number of inodes just returns a single node.
func TestNode_split_MinKeys(t *testing.T) {
	// Create a node.
	n := &node{inodes: make(inodes, 0), bucket: &Bucket{tx: &Tx{db: &DB{}, meta: &meta{pgid: 1}}}}
	n.put([]byte("00000001"), []byte("00000001"), []byte("0123456701234567"), 0, 0)
	n.put([]byte("00000002"), []byte("00000002"), []byte("0123456701234567"), 0, 0)

	// Split.
	n.split(20)
	if n.parent != nil {
		t.Fatalf("expected nil parent")
	}
}

// Ensure that a node that has keys that all fit on a page just returns one leaf.
func TestNode_split_SinglePage(t *testing.T) {
	// Create a node.
	n := &node{inodes: make(inodes, 0), bucket: &Bucket{tx: &Tx{db: &DB{}, meta: &meta{pgid: 1}}}}
	n.put([]byte("00000001"), []byte("00000001"), []byte("0123456701234567"), 0, 0)
	n.put([]byte("00000002"), []byte("00000002"), []byte("0123456701234567"), 0, 0)
	n.put([]byte("00000003"), []byte("00000003"), []byte("0123456701234567"), 0, 0)
	n.put([]byte("00000004"), []byte("00000004"), []byte("0123456701234567"), 0, 0)
	n.put([]byte("00000005"), []byte("00000005"), []byte("0123456701234567"), 0, 0)

	// Split.
	n.split(4096)
	if n.parent != nil {
		t.Fatalf("expected nil parent")
	}
}
