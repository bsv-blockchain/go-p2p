// Package main is an example of how to use the go-template package
package main

import (
	"log"

	"github.com/bsv-blockchain/go-template"
)

func main() {
	// Greet the user with a custom name
	name := "Alice"
	greeting := template.Greet(name)
	log.Println(greeting)
}
