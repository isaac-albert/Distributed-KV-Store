/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"log"

	"www.github.com/isaac-albert/Distributed-KV-Store/cmd/flag"
)

func main() {
	flag.StartFlags()
	err := flag.StartProgramOnCommand()
	if err != nil {
		log.Panic(err)
	}
}
