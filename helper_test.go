package main

import "testing"

func TestGetRootDomain(t *testing.T) {
	t.Parallel() // marks TLog as capable of running in parallel with other tests
	tests := []struct {
		input    string
		expected string
	}{
		{"1234.567.89.com", "89.com"},
		{"asd.digital", "asd.digital"},
		{"kndafs.adshg.andgjkSdg.jdnfdsa", "andgjkSdg.jdnfdsa"},
		{"sub.root.gv.at", "root.gv.at"},
	}

	for _, tt := range tests {
		tt := tt // NOTE: https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel() // marks each test case as capable of running in parallel with each other
			result, err := getRootDomain(tt.input)
			if err != nil {
				t.Fatalf("%s should not error out. error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Fatalf("wrong root domain detected. Input: %s Output: %s Expected: %s", tt.input, result, tt.expected)
			}
		})
	}
}
