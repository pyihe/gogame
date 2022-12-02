package uuid

import (
	"fmt"
	"testing"
)

func TestNext(t *testing.T) {
	id := Next()
	fmt.Println(id, Sequence(id), MachineID(id), UnixMilli(id))
	id = Next()
	fmt.Println(id, Sequence(id), MachineID(id), UnixMilli(id))
}

func BenchmarkNext(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Next()
	}
}
