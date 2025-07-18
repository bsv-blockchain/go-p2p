package template_test

import (
	"testing"

	"github.com/bsv-blockchain/go-template"
)

// BenchmarkGreet benchmarks the Greet function.
func BenchmarkGreet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = template.Greet("BenchmarkUser")
	}
}
