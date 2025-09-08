package main

import (
	"log"
	"os"

	"github.com/tez-capital/tezpay/cmd"

	"github.com/spf13/cobra/doc"
)

func main() {
	docsDirectory := "./docs/cmd"
	os.RemoveAll(docsDirectory)
	os.MkdirAll(docsDirectory, os.ModePerm)

	err := doc.GenMarkdownTreeCustom(cmd.RootCmd, docsDirectory,
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
