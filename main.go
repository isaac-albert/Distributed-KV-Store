/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"log"

	"www.github.com/isaac-albert/Distributed-KV-Store/cmd"
)

func main() {
	err := cmd.StartProgram()
	if err != nil {
		log.Fatal(err)
	}
}
