package template_test

import (
	"fmt"

	"github.com/bsv-blockchain/go-p2p"
)

// ExampleGreet demonstrates the usage of the Greet function.
func ExampleGreet() {
	msg := template.Greet("Alice")
	fmt.Println(msg)
	// Output: Hello Alice
}
