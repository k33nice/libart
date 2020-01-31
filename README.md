## Adaptive Radix Tree

The library provides an implementation of Adaptive Radix Tree (ART) based on ["The Adaptive Radix Tree: ARTful Indexing for Main-Memory Databases"](https://db.in.tum.de/~leis/papers/ART.pdf).

#### Overview

As this library implement a radix tree, it provides the following features:

* `O(k)` get/put/remove operations where `k` is key length.
* Minimum / Maximum value lookups
* Prefix compression
* Ordered iteration
* Prefix based iteration

#### Performance

This library is implemented following to the specification and use optimizations and avoiding memory allocations.
It relays on the `bytes` library that provides access to the SIMD instructions.

#### Usage

```go
package main

import (
    "github.com/k33nice/libart"
)

func main() {
    tree := art.New()

    tree.Insert([]byte("Some key"), "Some value")
    value := tree.Search([]byte("Some key"))
    if value != nil {
        // Do something with the value
    }

    tree.Each(func(n Node) {
        // n.Key() - key of the node
        // n.Value() - value of the node
    })
}
```
