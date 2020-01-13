// Copyright © 2019, Oleksandr Krykovliuk <k33nice@gmail.com>.
// Use of this source code is governed by the
// MIT license that can be found in the LICENSE file.

package art

import (
	"encoding/binary"
	_ "fmt"
	_ "log"
	"math/rand"
	"testing"

	"github.com/k33nice/art/internal/test"
	"github.com/stretchr/testify/assert"
)

// @spec: After a single insert operation, the tree should have a size of 1
//        and the root should be a leaf.
func TestArtTreeInsert(t *testing.T) {
	tree := newArt()
	tree.Insert(Key("hello"), "world")

	assert.Equal(t, int64(1), tree.size)
	assert.IsType(t, Leaf, tree.root.kind)
}

// @spec: After a single insert operation, the tree should be able
//        to retrieve there term it had inserted earlier
func TestArtTreeInsertAndSearch(t *testing.T) {
	tree := newArt()

	tree.Insert(Key("hello"), "world")
	res := tree.Search(Key("hello"))

	assert.Equal(t, "world", res)
}

// @spec: After Inserting twice and causing the root node to grow,
//        The tree should be able to successfully retrieve any of
//        the previous inserted values
func TestArtTreeInsert2AndSearch(t *testing.T) {
	tree := newArt()

	tree.Insert(Key("hello"), "world")
	tree.Insert(Key("yo"), "earth")

	res := tree.Search(Key("yo"))
	assert.NotNil(t, res)
	assert.Equal(t, "earth", res)

	res2 := tree.Search([]byte("hello"))
	assert.NotNil(t, res2)
	assert.Equal(t, "world", res2)
}

// An Art Node with a similar prefix should be split into new nodes accordingly
// And should be searchable as intended.
func TestArtTreeInsert2WithSimilarPrefix(t *testing.T) {
	tree := newArt()

	tree.Insert(Key("a"), "a")
	tree.Insert(Key("aa"), "aa")

	res := tree.Search(Key("aa"))

	assert.NotNil(t, res)
	assert.Equal(t, "aa", res)
}

// An Art Node with a similar prefix should be split into new nodes accordingly
// And should be searchable as intended.
func TestArtTreeInsert3AndSearchWords(t *testing.T) {
	tree := newArt()

	searchTerms := []string{"A", "a", "aa"}

	for i := range searchTerms {
		tree.Insert(Key(searchTerms[i]), searchTerms[i])
	}

	for i := range searchTerms {
		res := tree.Search(Key(searchTerms[i]))
		assert.NotNil(t, res)
		assert.Equal(t, searchTerms[i], res)
	}
}

func TestTreeInsertAndGrowToBiggerNode(t *testing.T) {
	var testData = []struct {
		totalNodes byte
		expected   Kind
	}{
		// {5, Node16},
		// {17, Node48},
		{49, Node256},
	}

	for _, data := range testData {
		tree := newArt()
		for i := byte(0); i < data.totalNodes; i++ {
			tree.Insert(Key{i}, i)
		}
		assert.Equal(t, int64(data.totalNodes), tree.size)
		assert.Equal(t, data.expected, tree.root.kind)
	}
}

// After inserting many words into the tree, we should be able to successfully retreive all of them
// To ensure their presence in the tree.
func TestInsertManyWordsAndEnsureSearchResultAndMinimumMaximum(t *testing.T) {
	tree := newArt()

	words := test.LoadTestFile("test/data/words.txt")

	for _, w := range words {
		tree.Insert(w, w)
	}

	for _, w := range words {

		res := tree.Search(w)

		assert.NotNil(t, res)

		assert.Equal(t, w, res)
	}

	minimum := tree.root.Minimum()
	assert.Equal(t, []byte("A"), minimum.Value().([]byte))

	maximum := tree.root.Maximum()
	assert.Equal(t, []byte("zythum"), maximum.Value().([]byte))
}

// After inserting many random UUIDs into the tree, we should be able to successfully retreive all of them
// To ensure their presence in the tree.
func TestInsertManyUUIDsAndEnsureSearchAndMinimumMaximum(t *testing.T) {
	tree := newArt()

	uuids := test.LoadTestFile("test/data/uuid.txt")

	for _, uuid := range uuids {
		tree.Insert(uuid, uuid)
	}

	for _, uuid := range uuids {
		res := tree.Search(uuid)

		assert.NotNil(t, res)
		assert.Equal(t, res, uuid)
	}

	minimum := tree.root.Minimum()
	assert.NotNil(t, minimum.Value())
	assert.Equal(t, []byte("00005076-6244-4739-808b-a58512fd6642"), minimum.Value().([]byte))

	maximum := tree.root.Maximum()
	assert.NotNil(t, maximum.Value())
	assert.Equal(t, []byte("ffffb7f1-20de-4a46-a3ec-8c87d5c7fce0"), maximum.Value().([]byte))
}

// Inserting a single value into the tree and removing it should result in a nil tree root.
func TestInsertAndRemove1(t *testing.T) {
	tree := newArt()

	tree.Insert(Key("test"), []byte("data"))

	tree.Delete(Key("test"))

	assert.Zero(t, tree.size)
	assert.Nil(t, tree.root)
}

// Inserting Two values into the tree and removing one of them
// should result in a tree root of type Leaf
func TestInsert2AndRemove1AndRootShouldBeLeafNode(t *testing.T) {
	tree := newArt()

	tree.Insert(Key("test"), []byte("data"))
	tree.Insert(Key("test2"), []byte("data"))

	tree.Delete(Key("test"))

	assert.Equal(t, int64(1), tree.size)
	assert.NotNil(t, tree.root)
	assert.IsType(t, Leaf, tree.root.kind)
}

// Inserting Two values into a tree and deleting them both
// should result in a nil tree root
// This tests the expansion of the root into a Node4 and
// successfully collapsing into a Leaf and then nil upon successive removals
func TestInsert2AndRemove2AndRootShouldBeNil(t *testing.T) {
	tree := newArt()

	tree.Insert(Key("test"), []byte("data"))
	tree.Insert(Key("test2"), []byte("data"))

	tree.Delete(Key("test"))
	tree.Delete(Key("test2"))

	assert.Zero(t, tree.size)
	assert.Nil(t, tree.root)
}

// Inserting Five values into a tree and deleting one of them
// should result in a tree root of type Node4
// This tests the expansion of the root into a Node16 and
// successfully collapsing into a Node4 upon successive removals
func TestInsert5AndRemove1AndRootShouldBeNode4(t *testing.T) {
	tree := newArt()

	for i := 0; i < 5; i++ {
		tree.Insert(Key{byte(i)}, []byte{byte(i)})
	}

	tree.Delete(Key{1})
	res := *(tree.root.FindChild(byte(1)))

	assert.Nil(t, res)
	assert.Equal(t, int64(4), tree.size)
	assert.NotNil(t, tree.root)
	assert.IsType(t, Node4, tree.root.kind)
}

// Inserting Five values into a tree and deleting all of them
// should result in a tree root of type nil
// This tests the expansion of the root into a Node16 and
// successfully collapsing into a Node4, Leaf, then nil
func TestInsert5AndRemove5AndRootShouldBeNil(t *testing.T) {
	tree := newArt()

	for i := 0; i < 5; i++ {
		tree.Insert(Key{byte(i)}, []byte{byte(i)})
	}

	for i := 0; i < 5; i++ {
		tree.Delete(Key{byte(i)})
	}

	res := tree.root.FindChild(byte(1))

	// Must fail if only both res and *res are not nil
	assert.Condition(t, func() bool {
		return res == nil || *res == nil
	})
	assert.Zero(t, tree.size)
	assert.Nil(t, tree.root)
}

// Inserting 17 values into a tree and deleting one of them should
// result in a tree root of type Node16
// This tests the expansion of the root into a Node48, and
// successfully collapsing into a Node16
func TestInsert17AndRemove1AndRootShouldBeNode16(t *testing.T) {
	tree := newArt()

	for i := 0; i < 17; i++ {
		tree.Insert(Key{byte(i)}, []byte{byte(i)})
	}

	tree.Delete(Key{2})
	res := *(tree.root.FindChild(byte(2)))

	assert.Nil(t, res)
	assert.Equal(t, int64(16), tree.size)
	assert.NotNil(t, tree.root)
	assert.IsType(t, Node16, tree.root.kind)
}

// Inserting 17 values into a tree and removing them all should
// result in a tree of root type nil
// This tests the expansion of the root into a Node48, and
// successfully collapsing into a Node16, Node4, Leaf, and then nil
func TestInsert17AndRemove17AndRootShouldBeNil(t *testing.T) {
	tree := newArt()

	for i := 0; i < 17; i++ {
		tree.Insert(Key{byte(i)}, []byte{byte(i)})
	}

	for i := 0; i < 17; i++ {
		tree.Delete(Key{byte(i)})
	}

	res := tree.root.FindChild(byte(1))

	assert.Condition(t, func() bool {
		return res == nil || *res == nil
	})
	assert.Zero(t, tree.size)
	assert.Nil(t, tree.root)
}

// Inserting 49 values into a tree and removing one of them should
// result in a tree root of type Node48
// This tests the expansion of the root into a Node256, and
// successfully collapasing into a Node48
func TestInsert49AndRemove1AndRootShouldBeNode48(t *testing.T) {
	tree := newArt()

	for i := 0; i < 49; i++ {
		tree.Insert(Key{byte(i)}, []byte{byte(i)})
	}

	tree.Delete(Key{2})
	res := *(tree.root.FindChild(byte(2)))
	assert.Nil(t, res)

	assert.Equal(t, int64(48), tree.size)

	assert.NotNil(t, tree.root)
	assert.IsType(t, Node48, tree.root.kind)
}

// Inserting 49 values into a tree and removing all of them should
// result in a nil tree root
// This tests the expansion of the root into a Node256, and
// successfully collapsing into a Node48, Node16, Node4, Leaf, and finally nil
func TestInsert49AndRemove49AndRootShouldBeNil(t *testing.T) {
	tree := newArt()

	for i := 0; i < 49; i++ {
		tree.Insert(Key{byte(i)}, []byte{byte(i)})
	}

	for i := 0; i < 49; i++ {
		tree.Delete(Key{byte(i)})
	}

	res := tree.root.FindChild(byte(1))
	assert.Condition(t, func() bool {
		return res == nil || *res == nil
	})
	assert.Zero(t, tree.size)
	assert.Nil(t, tree.root)
}

// A traversal of the tree should be in preorder
func TestEachPreOrderness(t *testing.T) {
	tree := newArt()
	tree.Insert(Key("1"), []byte("1"))
	tree.Insert(Key("2"), []byte("2"))

	var traversal []Node

	tree.Each(func(node Node) {
		traversal = append(traversal, node)
	})

	// Order should be Node4, 1, 2
	assert.Equal(t, traversal[0], tree.root)
	assert.Equal(t, Node4, traversal[0].Kind())

	assert.Equal(t, traversal[1].Key(), Key("1"))
	assert.Equal(t, Leaf, traversal[1].Kind())

	assert.Equal(t, traversal[2].Key(), Key("2"))
	assert.Equal(t, Leaf, traversal[2].Kind())
}

// A traversal of a Node48 node should preserve order
// And traverse in the same way for all other nodes.
// Node48s do not store their children in order, and require different logic to traverse them
// so we must test that logic seperately.
func TestEachNode48(t *testing.T) {
	tree := newArt()

	for i := 48; i > 0; i-- {
		tree.Insert(Key{byte(i)}, []byte{byte(i)})
	}

	var traversal []Node

	tree.Each(func(node Node) {
		traversal = append(traversal, node)
	})

	// Order should be Node48, then the rest of the keys in sorted order
	assert.Equal(t, traversal[0], tree.root)
	assert.Equal(t, Node48, traversal[0].Kind())

	for i := 1; i < 48; i++ {
		assert.Equal(t, traversal[i].Key(), Key{byte(i)})
		assert.Equal(t, Leaf, traversal[i].Kind())
	}
}

// After inserting many values into the tree, we should be able to iterate through all of them
// And get the expected number of nodes.
func TestEachFullIterationExpectCountOfAllTypes(t *testing.T) {
	tree := newArt()

	words := test.LoadTestFile("test/data/words.txt")

	for _, w := range words {
		tree.Insert(Key(w), []byte(w))
	}

	var leafCount int = 0
	var node4Count int = 0
	var node16Count int = 0
	var node48Count int = 0
	var node256Count int = 0

	tree.Each(func(node Node) {
		switch node.Kind() {
		case Node4:
			node4Count++
		case Node16:
			node16Count++
		case Node48:
			node48Count++
		case Node256:
			node256Count++
		case Leaf:
			leafCount++
		default:
		}
	})

	assert.Equalf(t, 235886, leafCount, "leaf count must be equal to 235886")
	assert.Equalf(t, 111616, node4Count, "node4 count must be equal to 111616")
	assert.Equalf(t, 12181, node16Count, "node16 count must be equal to 12181")
	assert.Equalf(t, 458, node48Count, "node48 count must be equal to 458")
	assert.Equalf(t, 1, node256Count, "node256 must be the only one")
}

// After Inserting many values into the tree, we should be able to remove them all
// And expect nothing to exist in the tree.
func TestInsertManyWordsAndRemoveThemAll(t *testing.T) {
	tree := newArt()

	words := test.LoadTestFile("test/data/words.txt")

	for _, w := range words {
		tree.Insert(Key(w), []byte(w))
	}

	numFound := 0

	for _, w := range words {
		tree.Delete(Key(w))
		dblCheck := tree.Search(Key(w))
		if dblCheck != nil {
			numFound++
		}
	}

	assert.Zero(t, tree.size)
	assert.Nil(t, tree.root)
}

// After Inserting many values into the tree, we should be able to remove them all
// And expect nothing to exist in the tree.
func TestInsertManyUUIDsAndRemoveThemAll(t *testing.T) {
	tree := newArt()

	uuids := test.LoadTestFile("test/data/uuid.txt")

	for _, uuid := range uuids {
		tree.Insert(Key(uuid), []byte(uuid))
	}

	numFound := 0

	for _, uuid := range uuids {
		tree.Delete(Key(uuid))

		dblCheck := tree.Search(Key(uuid))
		if dblCheck != nil {
			numFound++
		}
	}
	assert.Zero(t, tree.size)
	assert.Nil(t, tree.root)
}

// Regression test for issue/2
func TestInsertWithSameByteSliceAddress(t *testing.T) {
	rand.Seed(42)
	key := make([]byte, 8)
	tree := newArt()

	// Keep track of what we inserted
	keys := make(map[string]bool)

	for i := 0; i < 135; i++ {
		binary.BigEndian.PutUint64(key, uint64(rand.Int63()))
		tree.Insert(key, key)

		// Ensure that we can search these records later
		keys[string(key)] = true
	}

	assert.Equal(t, int64(len(keys)), tree.size)

	for k := range keys {
		n := tree.Search(Key(k))
		assert.NotNil(t, n)
	}
}

//
// Benchmarks
//
func BenchmarkWordsTreeInsert(b *testing.B) {
	words := test.LoadTestFile("test/data/words.txt")
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		tree := newArt()
		for _, w := range words {
			tree.Insert(w, w)
		}
	}
}

func BenchmarkWordsTreeSearch(b *testing.B) {
	words := test.LoadTestFile("test/data/words.txt")
	tree := newArt()
	for _, w := range words {
		tree.Insert(w, w)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, w := range words {
			tree.Search(w)
		}
	}
}

func BenchmarkWordsTreeForEach(b *testing.B) {
	words := test.LoadTestFile("test/data/words.txt")
	tree := newArt()
	for _, w := range words {
		tree.Insert(w, w)
	}
	b.ResetTimer()

	nodeTypes := make(map[Kind]int)
	tree.Each(func(n Node) {
		nodeTypes[n.Kind()]++
	})
	assert.Equal(b, map[Kind]int{Leaf: 235886, Node4: 111616, Node16: 12181, Node48: 458, Node256: 1}, nodeTypes)
}

func BenchmarkUUIDsTreeInsert(b *testing.B) {
	words := test.LoadTestFile("test/data/uuid.txt")
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		tree := newArt()
		for _, w := range words {
			tree.Insert(w, w)
		}
	}
}

func BenchmarkUUIDsTreeSearch(b *testing.B) {
	words := test.LoadTestFile("test/data/uuid.txt")
	tree := newArt()
	for _, w := range words {
		tree.Insert(w, w)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, w := range words {
			tree.Search(w)
		}
	}
}

func BenchmarkUUIDsTreeEach(b *testing.B) {
	words := test.LoadTestFile("test/data/uuid.txt")
	tree := newArt()
	for _, w := range words {
		tree.Insert(w, w)
	}
	b.ResetTimer()

	nodeTypes := make(map[Kind]int)
	tree.Each(func(n Node) {
		nodeTypes[n.Kind()]++
	})
	assert.Equal(b, map[Kind]int{Leaf: 500000, Node4: 103602, Node16: 56030}, nodeTypes)
}
