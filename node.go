// Copyright © 2019, Oleksandr Krykovliuk <k33nice@gmail.com>.
// Use of this source code is governed by the
// MIT license that can be found in the LICENSE file.

package art

import (
	"bytes"
	"sort"
	"unsafe"
)

const (
	// Inner nodes of type Node4 must have between 2 and 4 children.
	node4Min = 2
	node4Max = 4

	// Inner nodes of type Node16 must have between 5 and 16 children.
	node16Min = 5
	node16Max = 16

	// Inner nodes of type Node48 must have between 17 and 48 children.
	node48Min = 17
	node48Max = 48

	// Inner nodes of type Node256 must have between 49 and 256 children.
	node256Min = 49
	node256Max = 256

	MAX_PREFIX_LEN = 10
)

type node struct {
	size      int
	prefixLen int
	prefix    [MAX_PREFIX_LEN]byte
}

type node4 struct {
	node
	keys     [node4Max]byte
	children [node4Max + 1]*artNode
}

// Node with 16 children
type node16 struct {
	node
	keys     [node16Max]byte
	children [node16Max + 1]*artNode
}

// Node with 48 children
type node48 struct {
	node
	keys     [node256Max]byte
	children [node48Max + 1]*artNode
}

// Node with 256 children
type node256 struct {
	node
	children [node256Max + 1]*artNode
}

// Leaf node with variable key length
type leaf struct {
	key   Key
	value interface{}
}

// Defines a single artNode and its attributes.
type artNode struct {
	kind Kind
	ref  unsafe.Pointer
}

func newLeafNode(key []byte, value interface{}) *artNode {
	newKey := make([]byte, len(key))
	copy(newKey, key)
	return &artNode{
		kind: Leaf,
		ref:  unsafe.Pointer(&leaf{key: newKey, value: value}),
	}
}

// From the specification: The smallest node type can store up to 4 child
// pointers and uses an array of length 4 for keys and another
// array of the same length for pointers. The keys and pointers
// are stored at corresponding positions and the keys are sorted.
func newNode4() *artNode {
	return &artNode{kind: Node4, ref: unsafe.Pointer(&node4{})}
}

// From the specification: This node type is used for storing between 5 and
// 16 child pointers. Like the Node4, the keys and pointers
// are stored in separate arrays at corresponding positions, but
// both arrays have space for 16 entries. A key can be found
// efﬁciently with binary search or, on modern hardware, with
// parallel comparisons using SIMD instructions.
func newNode16() *artNode {
	return &artNode{kind: Node16, ref: unsafe.Pointer(&node16{})}
}

// From the specification: As the number of entries in a node increases,
// searching the key array becomes expensive. Therefore, nodes
// with more than 16 pointers do not store the keys explicitly.
// Instead, a 256-element array is used, which can be indexed
// with key bytes directly. If a node has between 17 and 48 child
// pointers, this array stores indexes into a second array which
// contains up to 48 pointers.
func newNode48() *artNode {
	return &artNode{kind: Node48, ref: unsafe.Pointer(&node48{})}
}

// From the specification: The largest node type is simply an array of 256
// pointers and is used for storing between 49 and 256 entries.
// With this representation, the next node can be found very
// efﬁciently using a single lookup of the key byte in that array.
// No additional indirection is necessary. If most entries are not
// null, this representation is also very space efﬁcient because
// only pointers need to be stored.
func newNode256() *artNode {
	return &artNode{kind: Node256, ref: unsafe.Pointer(&node256{})}
}

func (n *artNode) Key() Key {
	if n.IsLeaf() {
		return n.leaf().key
	}
	return nil
}

// Returns the value of the given node, or nil if it is not a leaf.
func (n *artNode) Value() interface{} {
	if n.kind != Leaf {
		return nil
	}
	return n.leaf().value
}

func (n *artNode) Kind() Kind {
	return n.kind
}

// Returns whether or not this particular art node is full.
func (n *artNode) IsFull() bool {
	return n.node().size == n.MaxSize()
}

// Returns whether or not this particular art node is a leaf node.
func (n *artNode) IsLeaf() bool { return n.kind == Leaf }

// Returns whether or not the key stored in the leaf matches the passed in key.
func (n *artNode) IsMatch(key []byte) bool {

	// Bail if user tries to compare  anything but a leaf node
	if n.kind != Leaf {
		return false
	}

	if len(n.leaf().key) != len(key) {
		return false
	}

	return bytes.Compare(n.leaf().key[:len(key)], key) == 0
}

// Returns the number of bytes that differ between the passed in key
// and the compressed path of the current node at the specified depth.
func (n *artNode) PrefixMismatch(key []byte, depth int) int {
	index := 0

	if n.node().prefixLen > MAX_PREFIX_LEN {
		for ; index < MAX_PREFIX_LEN; index++ {
			if key[depth+index] != n.node().prefix[index] {
				return index
			}
		}

		minKey := n.Minimum().leaf().key

		for ; index < n.node().prefixLen; index++ {
			if key[depth+index] != minKey[depth+index] {
				return index
			}
		}

	} else {

		for ; index < n.node().prefixLen; index++ {
			if key[depth+index] != n.node().prefix[index] {
				return index
			}
		}
	}

	return index
}

func (n *artNode) Index(key byte) int {
	switch n.kind {
	case Node4:
		// artNodes of type Node4 have a relatively simple lookup algorithm since
		// they are of very small size:  Simply iterate over all keys and check to see if they match.
		node := n.node4()
		for i := 0; i < node.size; i++ {
			if node.keys[i] == key {
				return int(i)
			}
		}
		return -1
	case Node16:
		return bytes.IndexByte(n.node16().keys[:], key)

	case Node48:
		// artNodes of type Node48 store the indicies in which to access their children
		// in the keys array which are byte-accessible by the desired key.
		// However, when this key array initialized, it contains many 0 value indicies.
		// In order to distinguish if a child actually exists, we increment this value
		// during insertion and decrease it during retrieval.
		index := int(n.node48().keys[key])
		if index > 0 {
			return int(index) - 1
		}

		return -1
	case Node256:
		// artNodes of type Node256 possibly have the simplest lookup algorithm.
		// Since all of their keys are byte-addressable, we can simply index to the specific child with the key.
		return int(key)
	}

	return -1
}

// FindChild returns a pointer to the child that matches the passed in key,
// or nil if not present.
func (n *artNode) FindChild(key byte) **artNode {
	var nullNode *artNode = nil

	if n == nil {
		return &nullNode
	}

	var idx int
	switch n.kind {
	case Node4, Node16, Node48:
		idx = n.Index(key)
		if idx < 0 {
			return &nullNode
		}
	case Node256:
		idx = int(key)
		if n.node256().children[idx] == nil {
			return &nullNode
		}
	}

	if idx >= 0 {
		switch n.kind {
		case Node4:
			return &n.node4().children[idx]
		case Node16:
			return &n.node16().children[idx]
		case Node48:
			return &n.node48().children[idx]
		case Node256:
			return &n.node256().children[idx]
		}
	}

	return &nullNode
}

// AddChild adds the passed in node to the current artNode's children at the specified key.
// The current node will grow if necessary in order for the insertion to take place.
func (n *artNode) AddChild(key byte, node *artNode) {
	switch n.kind {
	case Node4:
		n4 := n.node4()
		nn := n.node()
		if nn.size < n.MaxSize() {
			index := 0
			for ; index < nn.size; index++ {
				if key < n4.keys[index] {
					break
				}
			}

			for i := nn.size; i > index; i-- {
				if n4.keys[i-1] > key {
					n4.keys[i] = n4.keys[i-1]
					n4.children[i] = n4.children[i-1]
				}
			}

			n4.keys[index] = key
			n4.children[index] = node
			nn.size++
		} else {
			n.grow()
			n.AddChild(key, node)
		}

	case Node16:
		n16 := n.node16()
		if n16.size < n.MaxSize() {

			index := sort.Search(n16.size, func(i int) bool {
				return key <= n16.keys[byte(i)]
			})

			for i := n16.size; i > index; i-- {
				if n16.keys[i-1] > key {
					n16.keys[i] = n16.keys[i-1]
					n16.children[i] = n16.children[i-1]
				}
			}
			n16.keys[index] = key
			n16.children[index] = node
			n16.size++
		} else {
			n.grow()
			n.AddChild(key, node)
		}

	case Node48:
		n48 := n.node48()
		nn := n.node()
		if nn.size < n.MaxSize() {
			index := 0

			for n48.children[index] != nil {
				index++
			}

			n48.children[index] = node
			n48.keys[key] = byte(index + 1)
			nn.size++
		} else {
			n.grow()
			n.AddChild(key, node)
		}

	case Node256:
		if !n.IsFull() {
			n.node256().children[key] = node

			n.node().size++
		}
	}
}

// RemoveChild remove the child by the passed in key is removed if found
// and the current artNode is shrunk if it falls below its minimum size.
func (n *artNode) RemoveChild(key byte) {
	switch n.kind {
	case Node4:
		node := n.node4()

		idx := n.Index(key)

		node.keys[idx] = 0
		node.children[idx] = nil

		if idx >= 0 {
			for i := idx; i < node.size-1; i++ {
				node.keys[i] = node.keys[i+1]
				node.children[i] = node.children[i+1]
			}

		}

		node.keys[node.size-1] = 0
		node.children[node.size-1] = nil

		node.size--
	case Node16:
		node := n.node16()

		idx := n.Index(key)

		node.keys[idx] = 0
		node.children[idx] = nil

		if idx >= 0 {
			for i := idx; i < node.size-1; i++ {
				node.keys[i] = node.keys[i+1]
				node.children[i] = node.children[i+1]
			}

		}

		node.keys[node.size-1] = 0
		node.children[node.size-1] = nil

		node.size--

	case Node48:
		node := n.node48()
		idx := n.Index(key)

		if idx >= 0 {
			child := node.children[idx]
			if child != nil {
				node.children[idx] = nil
				node.keys[key] = 0
				node.size--
			}
		}

	case Node256:
		node := n.node256()
		idx := n.Index(key)

		child := node.children[idx]
		if child != nil {
			node.children[idx] = nil
			node.size--
		}

	}

	if n.node().size < n.MinSize() {
		n.shrink()
	}
}

// Grows the current artNode to the next biggest size.
// artNodes of type Node4 will grow to Node16
// artNodes of type Node16 will grow to Node48.
// artNodes of type Node48 will grow to Node256.
// artNodes of type Node256 will not grow, as they are the biggest type of artNodes
func (n *artNode) grow() {
	switch n.kind {
	case Node4:
		other := newNode16()
		other.copyMeta(n)
		other16 := other.node16()
		n4 := n.node4()
		for i := 0; i < n4.size; i++ {
			other16.keys[i] = n4.keys[i]
			other16.children[i] = n4.children[i]
		}

		n.replaceWith(other)

	case Node16:
		other := newNode48()
		other.copyMeta(n)
		other48 := other.node48()
		n16 := n.node16()
		for i := 0; i < n16.size; i++ {
			child := n16.children[i]
			if child != nil {
				index := 0

				for j := 0; j < len(other48.children); j++ {
					if other48.children[index] != nil {
						index++
					}
				}

				other48.children[index] = child
				other48.keys[n16.keys[i]] = byte(index + 1)
			}
		}

		n.replaceWith(other)

	case Node48:
		other := newNode256()
		other.copyMeta(n)
		other256 := other.node256()
		n48 := n.node48()
		for i := 0; i < len(n48.keys); i++ {
			child := *(n.FindChild(byte(i)))
			if child != nil {
				other256.children[byte(i)] = child
			}
		}

		n.replaceWith(other)

	case Node256:
		// Can't get no bigger
	}
}

// Shrinks the current artNode to the next smallest size.
// artNodes of type Node256 will grow to Node48
// artNodes of type Node48 will grow to Node16.
// artNodes of type Node16 will grow to Node4.
// artNodes of type Node4 will collapse into its first child.
// If that child is not a leaf, it will concatenate its current prefix with that of its childs
// before replacing itself.
func (n *artNode) shrink() {
	switch n.kind {
	case Node4:
		// From the specification: If that node now has only one child, it is replaced by its child
		// and the compressed path is adjusted.
		n4 := n.node4()
		other := n4.children[0]

		if !other.IsLeaf() {
			currentPrefixLen := n4.prefixLen

			if currentPrefixLen < MAX_PREFIX_LEN {
				n4.prefix[currentPrefixLen] = n4.keys[0]
				currentPrefixLen++
			}

			if currentPrefixLen < MAX_PREFIX_LEN {
				childPrefixLen := min(other.node().prefixLen, MAX_PREFIX_LEN-currentPrefixLen)
				memcpy(n4.prefix[currentPrefixLen:], other.node().prefix[:], childPrefixLen)
				currentPrefixLen += childPrefixLen
			}

			memcpy(other.node().prefix[:], n4.prefix[:], min(currentPrefixLen, MAX_PREFIX_LEN))
			other.node().prefixLen += n4.prefixLen + 1
		}

		n.replaceWith(other)

	case Node16:
		other := newNode4()
		other.copyMeta(n)
		other.node4().size = 0

		for i := 0; i < len(other.node4().keys); i++ {
			other.node4().keys[i] = n.node16().keys[i]
			other.node4().children[i] = n.node16().children[i]
			other.node16().size++
		}

		n.replaceWith(other)

	case Node48:
		other := newNode16()
		other.copyMeta(n)
		other.node16().size = 0

		for i := 0; i < len(n.node48().keys); i++ {
			idx := n.node48().keys[byte(i)]
			if idx > 0 {
				child := n.node48().children[idx-1]
				if child != nil {
					other.node16().children[other.node16().size] = child
					other.node16().keys[other.node16().size] = byte(i)
					other.node16().size++
				}
			}
		}

		n.replaceWith(other)

	case Node256:
		other := newNode48()
		other.copyMeta(n)
		other.node48().size = 0

		for i := 0; i < len(n.node256().children); i++ {
			child := n.node256().children[byte(i)]
			if child != nil {
				other.node48().children[other.node48().size] = child
				other.node48().keys[byte(i)] = byte(other.node48().size + 1)
				other.node48().size++
			}
		}

		n.replaceWith(other)
	}
}

// Returns the longest number of bytes that match between the current node's prefix
// and the passed in node at the specified depth.
func (n *artNode) LongestCommonPrefix(other *artNode, depth int) int {
	limit := min(len(n.leaf().key), len(other.leaf().key)) - depth

	i := 0
	for ; i < limit; i++ {
		if n.leaf().key[depth+i] != other.leaf().key[depth+i] {
			return i
		}
	}
	return i
}

// Returns the minimum number of children for the current node.
func (n *artNode) MinSize() int {
	switch n.kind {
	case Node4:
		return node4Min
	case Node16:
		return node16Min
	case Node48:
		return node48Min
	case Node256:
		return node256Min
	}
	return 0
}

// Returns the maximum number of children for the current node.
func (n *artNode) MaxSize() int {
	switch n.kind {
	case Node4:
		return node4Max
	case Node16:
		return node16Max
	case Node48:
		return node48Max
	case Node256:
		return node256Max
	}
	return 0
}

// Returns the Minimum child at the current node.
// The minimum child is determined by recursively traversing down the tree
// by selecting the smallest possible byte in each child
// until a leaf has been reached.
func (n *artNode) Minimum() *artNode {
	if n == nil {
		return nil
	}

	switch n.kind {
	case Leaf:
		return n

	case Node4:
		return n.node4().children[0].Minimum()
	case Node16:
		return n.node16().children[0].Minimum()

	case Node48:
		i := 0

		for n.node48().keys[i] == 0 {
			i++
		}

		child := n.node48().children[n.node48().keys[i]-1]

		return child.Minimum()

	case Node256:
		i := 0
		for n.node256().children[i] == nil {
			i++
		}
		return n.node256().children[i].Minimum()

	}

	return n
}

// Returns the Maximum child at the current node.
// The maximum child is determined by recursively traversing down the tree
// by selecting the biggest possible byte in each child
// until a leaf has been reached.
func (n *artNode) Maximum() *artNode {
	if n == nil {
		return nil
	}

	switch n.kind {
	case Leaf:
		return n

	case Node4:
		node := n.node4()
		return node.children[node.size-1].Maximum()
	case Node16:
		node := n.node16()
		return node.children[node.size-1].Maximum()

	case Node48:
		node := n.node48()
		i := len(node.keys) - 1
		for node.keys[i] == 0 {
			i--
		}

		child := node.children[node.keys[i]-1]
		return child.Maximum()

	case Node256:

		node := n.node256()
		i := len(node.children) - 1

		for i > 0 && node.children[byte(i)] == nil {
			i--
		}

		return node.children[i].Maximum()

	}

	return nil
}

func (n *artNode) node() *node {
	return (*node)(n.ref)
}

func (n *artNode) node4() *node4 {
	return (*node4)(n.ref)
}

func (n *artNode) node16() *node16 {
	return (*node16)(n.ref)
}

func (n *artNode) node48() *node48 {
	return (*node48)(n.ref)
}

func (n *artNode) node256() *node256 {
	return (*node256)(n.ref)
}

func (n *artNode) leaf() *leaf {
	return (*leaf)(n.ref)
}

// Replaces the current node with the passed in artNode.
func (n *artNode) replaceWith(other *artNode) {
	*n = *other
}

// Copies the prefix and size metadata from the passed in artNode
// to the current node.
func (n *artNode) copyMeta(src *artNode) {
	if src == nil {
		return
	}
	to := n.node()
	from := src.node()
	to.size = from.size
	to.prefixLen = from.prefixLen

	for i, limit := 0, min(from.prefixLen, MAX_PREFIX_LEN); i < limit; i++ {
		to.prefix[i] = from.prefix[i]
	}
}

// Returns the smallest of the two passed in integers.
func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
