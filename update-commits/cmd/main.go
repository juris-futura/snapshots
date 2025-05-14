package main

import (
	"fmt"
	"os"

	processor "update-commits/yaml"

	"github.com/fatih/color"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: update-commits <yaml-file>")
		os.Exit(1)
	}

	yamlFile := os.Args[1]
	err := processor.ProcessYAML(yamlFile)
	if err != nil {
		red := color.New(color.FgRed).SprintFunc()
		fmt.Printf("Error processing YAML file: %v\n", red(err))
		os.Exit(1)
	}
}
