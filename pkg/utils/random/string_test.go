package random

import "testing"

const n = 16

func BenchmarkBytesMaskImprSrcUnsafe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateString(n)
	}
}
