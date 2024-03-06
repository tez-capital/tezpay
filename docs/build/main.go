package main

import (
	"log"

	"github.com/alis-is/tezpay/cmd"

	"github.com/spf13/cobra/doc"
)

func main() {
	err := doc.GenMarkdownTreeCustom(cmd.RootCmd, "./docs/cmd",
		func(p string) string { return p },
		func(s string) string { return "/tezpay/reference/cmd/" + s[:len(s)-3] },
	)
	if err != nil {
		log.Fatal(err)
	}

	GenerateDefaultHJson()
	GenerateSampleHJson()
	GenerateStarterHJson()
	GenerateHookSampleData()
}
