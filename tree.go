// Copyright © 2019, Oleksandr Krykovliuk <k33nice@gmail.com>.
// Use of this source code is governed by the
// MIT license that can be found in the LICENSE file.

package art

type tree struct {
	root *artNode
	size int64
}

func newArt() *tree {
	return &tree{root: nil, size: 0}
}

// Returns the node that contains the passed in key, or nil if not found.
func (t *tree) Search(key Key) Value {
	return t.searchHelper(t.root, key, 0)
}

// Recursive search helper function that traverses the tree.
// Returns the node that contains the passed in key, or nil if not found.
func (t *tree) searchHelper(current *artNode, key []byte, depth int) interface{} {
	// While we have nodes to search
	for current != nil {
		// Check if the current is a match
		if current.IsLeaf() {
			if current.IsMatch(key) {
				return current.leaf().value
			}

			// Bail if no match
			return nil
		}

		// Check if our key mismatches the current compressed path
		if current.PrefixMismatch(key, depth) != current.node().prefixLen {
			// Bail if there's a mismatch during traversal.
			return nil
		}
		// Otherwise, increase depth accordingly.
		depth += current.node().prefixLen

		// Find the next node at the specified index, and update depth.
		var keyChar byte
		if depth < 0 || depth >= len(key) {
			keyChar = byte(0)
		} else {
			keyChar = key[depth]
		}
		current = *(current.FindChild(keyChar))
		depth++
	}

	return nil
}

// Inserts the passed in value that is indexed by the passed in key into the ArtTree.
func (t *tree) Insert(key Key, value Value) {
	t.insertHelper(&t.root, key, value, 0)
}

// Recursive helper function that traverses the tree until an insertion point is found.
// There are four methods of insertion:
//
// If the current node is null, a new node is created with the passed in key-value pair
// and inserted at the current position.
//
// If the current node is a leaf node, it will expand to a new artNode of type Node4
// to contain itself and a new leaf node containing the passed in key-value pair.
//
// If the current node's prefix differs from the key at a specified depth,
// a new artNode of type Node4 is created to contain the current node and the new leaf node
// with an adjusted prefix to account for the mismatch.
//
// If there is no child at the specified key at the current depth of traversal, a new leaf node
// is created and inserted at this position.
func (t *tree) insertHelper(currentRef **artNode, key []byte, value interface{}, depth int) {
	// @spec: Usually, the leaf can
	//        simply be inserted into an existing inner node, after growing
	//        it if necessary.
	if *currentRef == nil {
		*currentRef = newLeafNode(key, value)
		t.size++
		return
	}
	current := *currentRef

	// @spec: If, because of lazy expansion,
	//        an existing leaf is encountered, it is replaced by a new
	//        inner node storing the existing and the new leaf
	if current.IsLeaf() {

		// TODO Determine if we should overwrite keys if they are attempted to overwritten.
		//      Currently, we bail if the key matches.
		if current.IsMatch(key) {
			return
		}

		// Create a new Inner Node to contain the new Leaf and the current node.
		newNode4 := newNode4()
		newLeafNode := newLeafNode(key, value)

		// Determine the longest common prefix between our current node and the key
		limit := current.LongestCommonPrefix(newLeafNode, depth)

		newNode4.node().prefixLen = limit

		memcpy(newNode4.node().prefix[:], key[depth:], min(newNode4.node().prefixLen, maxPrefixLen))

		*currentRef = newNode4

		// Add both children to the new Inner Node
		if depth+newNode4.node().prefixLen < 0 || depth+newNode4.node().prefixLen >= len(current.leaf().key) {
			newNode4.AddChild(0, current)
		} else {
			newNode4.AddChild(current.leaf().key[depth+newNode4.node().prefixLen], current)
		}

		if depth+newNode4.node().prefixLen < 0 || depth+newNode4.node().prefixLen >= len(key) {
			newNode4.AddChild(0, newLeafNode)
		} else {
			newNode4.AddChild(key[depth+newNode4.node().prefixLen], newLeafNode)
		}

		t.size++
		return
	}

	// @spec: Another special case occurs if the key of the new leaf
	//        differs from a compressed path: A new inner node is created
	//        above the current node and the compressed paths are adjusted accordingly.
	node := current.node()
	if node.prefixLen != 0 {

		mismatch := current.PrefixMismatch(key, depth)

		// If the key differs from the compressed path
		if mismatch != node.prefixLen {

			// Create a new Inner Node that will contain the current node
			// and the desired insertion key
			newNode4 := newNode4()
			*currentRef = newNode4
			newNode4.node().prefixLen = mismatch

			// Copy the mismatched prefix into the new inner node.
			memcpy(newNode4.node().prefix[:], node.prefix[:], mismatch)

			// Adjust prefixes so they fit underneath the new inner node
			if node.prefixLen < maxPrefixLen {
				newNode4.AddChild(node.prefix[mismatch], current)
				node.prefixLen -= (mismatch + 1)
				memmove(node.prefix[:], node.prefix[mismatch+1:], min(node.prefixLen, maxPrefixLen))
			} else {
				node.prefixLen -= (mismatch + 1)
				minKey := current.Minimum().leaf().key
				newNode4.AddChild(minKey[depth+mismatch], current)
				memmove(node.prefix[:], minKey[depth+mismatch+1:], min(node.prefixLen, maxPrefixLen))
			}

			// Attach the desired insertion key
			newLeafNode := newLeafNode(key, value)
			newNode4.AddChild(key[depth+mismatch], newLeafNode)

			t.size++
			return
		}

		depth += node.prefixLen
	}

	// Find the next child
	next := current.FindChild(key[depth])

	// If we found a child that matches the key at the current depth
	if *next != nil {
		// Recurse, and keep looking for an insertion point
		t.insertHelper(next, key, value, depth+1)
	} else {
		// Otherwise, Add the child at the current position.
		current.AddChild(key[depth], newLeafNode(key, value))
		t.size++
	}
}

// Delete the child that is accessed by the passed in key.
func (t *tree) Delete(key []byte) bool {
	return t.removeHelper(&t.root, key, 0)
}

// Recursive helper for Removing child nodes.
// There are two methods for removal:
//
// If the current node is a leaf and matches the specified key, remove it.
//
// If the next child at the specifed key and depth matches,
// the current node shall remove it accordingly.
func (t *tree) removeHelper(currentRef **artNode, key []byte, depth int) bool {
	// Bail early if we are at a nil node.
	if t == nil || *currentRef == nil || len(key) == 0 {
		return false
	}

	current := *currentRef
	// If the current node matches, remove it.
	if current.IsLeaf() {
		if current.IsMatch(key) {
			*currentRef = nil
			t.size--
			return true
		}
	}

	// If the current node contains a prefix length
	if current.node().prefixLen != 0 {

		// Bail out if we encounter a mismatch
		mismatch := current.PrefixMismatch(key, depth)
		if mismatch != current.node().prefixLen {
			return false
		}

		// Increase traversal depth
		depth += current.node().prefixLen
	}

	// Find the next child
	var keyChar byte
	if depth < 0 || depth >= len(key) {
		keyChar = byte(0)
	} else {
		keyChar = key[depth]
	}
	next := current.FindChild(keyChar)

	// Let the Inner Node handle the removal logic if the child is a match
	if *next != nil && (*next).IsLeaf() && (*next).IsMatch(key) {
		current.RemoveChild(keyChar)
		t.size--
		return true
	}
	return t.removeHelper(next, key, depth+1)
}

// Convenience method for EachPreorder
func (t *tree) Each(callback Callback, opts ...int) {
	t.eachHelper(t.root, callback)
}

func (t *tree) Size() int {
	return int(t.size)
}

// Recursive helper for iterative over the tree.  Iterates over all nodes in the tree,
// executing the passed in callback as specified by the passed in traversal type.
func (t *tree) eachHelper(current *artNode, callback Callback) {
	// Bail early if there's no node to iterate over
	if current == nil {
		return
	}

	callback(current)

	switch current.kind {
	case Node4:
		t.eachChildren(current.node4().children[:], callback)

	case Node16:
		t.eachChildren(current.node16().children[:], callback)

	// Nodes of type Node48 do not necessarily store their children in sorted order.
	// So we must instead iterate over their keys, acccess the children, and iterate properly.
	case Node48:
		node := current.node48()
		child := node.children[node48Max]
		if child != nil {
			t.eachHelper(child, callback)
		}

		for _, i := range node.keys {
			if i > 0 {
				next := current.node48().children[i-1]
				if next != nil {
					t.eachHelper(next, callback)
				}
			}
		}

	case Node256:
		t.eachChildren(current.node256().children[:], callback)
	}
}

func (t *tree) eachChildren(children []*artNode, callback Callback) {
	nullChild := children[len(children)-1]
	if nullChild != nil {
		t.eachHelper(nullChild, callback)
	}

	for _, child := range children {
		if child != nil && child != nullChild {
			t.eachHelper(child, callback)
		}
	}
}

func memcpy(dest []byte, src []byte, numBytes int) {
	for i := 0; i < numBytes && i < len(src) && i < len(dest); i++ {
		dest[i] = src[i]
	}
}

func memmove(dest []byte, src []byte, numBytes int) {
	for i := 0; i < numBytes; i++ {
		dest[i] = src[i]
	}
}
