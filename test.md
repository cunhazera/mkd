# Hello, Markdown!

This is a **bold** statement and this is *italic*.

## Code Example

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, world!")
}
```

## Lists

- Item one
- Item two
  - Nested item
- Item three

1. First
2. Second
3. Third

## Blockquote

> The best way to predict the future is to invent it.
> — Alan Kay

## Table

| Name  | Age | Role       |
|-------|-----|------------|
| Alice | 30  | Engineer   |
| Bob   | 25  | Designer   |
| Carol | 35  | PM         |

## Link

Here is an example `single snippet code block`

Heres a [link](https://github.com/cunhazera)

Check out [Charmbracelet](https://charm.sh) for more TUI tools.

```go
    // before
    quantities.forEach { quantity ->
        buildPatchesFromQuantity(quantity, index + 1)
    }

    // after
    quantities.forEachIndexed { quantityIndex, quantity ->
        buildPatchesFromQuantity(quantity, agentIndex = index + 1, quantityIndex = quantityIndex)
    }
```