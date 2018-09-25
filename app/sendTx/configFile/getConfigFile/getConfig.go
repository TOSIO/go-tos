package main

import (
	"fmt"

	"github.com/TOSIO/go-tos/node"
	"github.com/naoina/toml"
)

func main() {
	data, err := toml.Marshal(node.DefaultConfig)

	if err != nil {
		return
	}

	fmt.Printf("%s\n", data)
}
