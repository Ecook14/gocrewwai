package utils_test

import (
	"testing"
	"github.com/Ecook14/gocrewwai/pkg/utils"
)

func TestFindMatchingBlock(t *testing.T) {
	fullText := `package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}`

	tests := []struct {
		name    string
		search  string
		found   bool
		mode    string
	}{
		{"Exact Match", "fmt.Println(\"Hello, World!\")", true, "exact"},
		{"Rstrip Match", "fmt.Println(\"Hello, World!\")  ", true, "rstrip"},
		{"Trim Match", "   fmt.Println(\"Hello, World!\")   ", true, "trim"},
		{"Multiline Match", "func main() {\n    fmt.Println(\"Hello, World!\")\n}", true, "exact"},
		{"Not Found", "fmt.Println(\"Goodbye\")", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, ok := utils.FindMatchingBlock(fullText, tt.search)
			if ok != tt.found {
				t.Errorf("FindMatchingBlock() ok = %v, want %v", ok, tt.found)
			}
		})
	}
}
