// Copyright © 2019, Oleksandr Krykovliuk <k33nice@gmail.com>.
// Use of this source code is governed by the
// MIT license that can be found in the LICENSE file.

package art

import (
	"bytes"
	_ "fmt"
	_ "sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A Leaf Node should be able to retreive its value
func TestValue(t *testing.T) {
	leaf := newLeafNode([]byte("foo"), "foo")

	if leaf.Value() != "foo" {
		t.Error("Unexpected value for leaf node")
	}

}

// An artNode4 should be able to find the expected child element
func TestAddChildAndFindChildForAllNodeTypes(t *testing.T) {
	nodes := []*artNode{newNode4(), newNode16(), newNode48(), newNode256()}

	// For each different type of node
	for node := range nodes {
		n := nodes[node]

		// Fill it up
		for i := 0; i < n.maxSize(); i++ {
			newChild := newLeafNode([]byte{byte(i)}, byte(i))
			n.AddChild(byte(i), newChild)
		}

		// Expect to find all children for that paticular type of node
		for i := 0; i < n.maxSize(); i++ {
			x := *(n.findChild(byte(i)))

			if x == nil {
				t.Error("Could not find child as expected")
			}
			if x.Value() != byte(i) {
				t.Error("Child value does not match as expected")
			}
		}
	}
}

// Index should be able to return the correct location of the child
// at the specfied key for all inner node types
func TestIndexForAllNodeTypes(t *testing.T) {
	nodes := []*artNode{newNode4(), newNode16(), newNode48(), newNode256()}

	// For each different type of node
	for node := range nodes {
		n := nodes[node]

		// Fill it up
		for i := 0; i < n.maxSize(); i++ {
			newChild := newLeafNode([]byte{byte(i)}, byte(i))
			n.AddChild(byte(i), newChild)
		}

		for i := 0; i < n.maxSize(); i++ {
			if n.index(byte(i)) != i {
				t.Error("Unexpected value for Index function")
			}
		}
	}
}

// An artNode4 should be able to add a child, and then return the expected child reference.
func TestArtNode4AddChild1AndFindChild(t *testing.T) {
	n := newNode4()
	n2 := newNode4()
	n.AddChild('a', n2)

	assert.Equal(t, 1, n.node().size)

	x := *(n.findChild('a'))
	assert.Equal(t, n2, x)
}

// An artNode4 should be able to add two child elements with differing prefixes
// And preserve the sorted order of the keys.
func TestArtNode4AddChildTwicePreserveSorted(t *testing.T) {
	n := newNode4()
	n2 := newNode4()
	n3 := newNode4()
	n.AddChild('b', n2)
	n.AddChild('a', n3)

	if n.node().size < 2 {
		t.Error("Size is incorrect after adding one child to empty Node4")
	}

	if n.node4().keys[0] != 'a' {
		t.Error("Unexpected key value for first key index")
	}

	if n.node4().keys[1] != 'b' {
		t.Error("Unexpected key value for second key index")
	}
}

// An artNode4 should be able to add 4 child elements with different prefixes
// And preserve the sorted order of the keys.
func TestArtNode4AddChild4PreserveSorted(t *testing.T) {
	n := newNode4()

	for i := 4; i > 0; i-- {
		n.AddChild(byte(i), newNode4())
	}

	if n.node4().size < 4 {
		t.Error("Size is incorrect after adding one child to empty Node4")
	}

	expectedKeys := []byte{1, 2, 3, 4}
	if bytes.Compare(n.node4().keys[:], expectedKeys) != 0 {
		t.Error("Unexpected key sequence")
	}
}

// Art Nodes of all types should grow to the next biggest size in sequence
func TestGrow(t *testing.T) {
	nodes := []*artNode{newNode4(), newNode16(), newNode48()}
	expectedTypes := []Kind{Node16, Node48, Node256}

	for i := range nodes {
		node := nodes[i]

		node.grow()
		if node.kind != expectedTypes[i] {
			t.Error("Unexpected node type after growing")
		}
	}
}

// Art Nodes of all types should next smallest size in sequence
func TestShrink(t *testing.T) {
	// nodes := []*artNode{newNode256(), newNode48(), newNode16(), newNode4()}
	// expectedTypes := []Kind{Node48, Node16, Node4, Leaf}
	nodes := []*artNode{newNode48()}
	expectedTypes := []Kind{Node16}

	for i := range nodes {
		node := nodes[i]

		for j := 0; j < node.minSize(); j++ {
			if node.kind != Node4 {
				node.AddChild(byte(i), newNode4())
			} else {
				// We want to test that the Node4 reduces itself to
				// A Leaf if its only child is a leaf
				node.AddChild(byte(i), newLeafNode(nil, nil))
			}
		}

		node.shrink()
		if node.kind != expectedTypes[i] {
			t.Error("Unexpected node type after shrinking")
		}
	}
}

func TestNewLeafNode(t *testing.T) {
	key := []byte{'a', 'r', 't'}
	value := "tree"
	l := newLeafNode(key, value)

	if &l.leaf().key == &key {
		t.Errorf("Address of key byte slices should not match.")
	}

	if bytes.Compare(l.leaf().key, key) != 0 {
		t.Errorf("Expected key value to match the one supplied")
	}

	if l.leaf().value != value {
		t.Errorf("Expected initial value to match the one supplied")
	}

	if l.kind != Leaf {
		t.Errorf("Expected Leaf node to be of Leaf type")
	}
}
