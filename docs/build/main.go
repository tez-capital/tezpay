package main

import (
	"log"

	"github.com/alis-is/tezpay/cmd"

	"github.com/spf13/cobra/doc"
)

func main() {
	err := doc.GenMarkdownTree(cmd.RootCmd, "./docs/cmd")
	if err != nil {
		log.Fatal(err)
	}

	GenerateDefault()
	GenerateSample()
}
