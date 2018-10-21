package common

import (
	"testing"
)

func BenchmarkListDirectories(b *testing.B) {
	for i := 0; i < b.N; i++ {
		output := make(map[string]struct{})
		if err := ListDirectories("/sys/fs/cgroup", "", true, output); err != nil {
			b.Fatal(err)
		}
	}
}
