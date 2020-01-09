// Copyright © 2019, Oleksandr Krykovliuk <k33nice@gmail.com>.
// Use of this source code is governed by the
// MIT license that can be found in the LICENSE file.

package art

// Kind - adaptive radix tree node type.
type Kind uint8

// Types of node.
const (
	Leaf Kind = iota
	Node4
	Node16
	Node48
	Node256
)

// Key type. Can be any sequence of characters.
type Key = []byte

// Value type.
type Value = interface{}

// Node - delineate node entity.
type Node interface {
	Kind() Kind
	Key() Key
	Value() Value
}

// Callback - callback function that is passed in Each.
type Callback func(node Node)

// Tree - delineate adaptive radix tree entity.
type Tree interface {
	Insert(key Key, value Value)
	Search(key Key) (value Value)
	Delete(key Key) (deleted bool)
	Each(cb Callback, options ...int)
	Size() int
}

// New - creates a new instace of adaptive radix tree.
func New() Tree {
	return newArt()
}
