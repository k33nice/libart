## Adaptive Radix Tree

The library provides an implementation of Adaptive Radix Tree (ART) based on ["The Adaptive Radix Tree: ARTful Indexing for Main-Memory Databases"](https://db.in.tum.de/~leis/papers/ART.pdf).

#### Overview

- `O(k)` get/put/remove operations where `k` is key lenght.

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
