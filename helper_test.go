package main

import "testing"

// https://github.com/golang/go/wiki/TableDrivenTests
func TestParseUnknownDate(t *testing.T) {
	t.Parallel() // marks TLog as capable of running in parallel with other tests
	tests := []struct {
		date string
	}{
		{"09-Mar-2023"},
		{"31-Jul-2022"},
		{"2022-12-12T11:01:02Z"},
		{"2022-12-03"},
		{"2022. 12. 01."},
		{"2022-12-12 11:40:12"},
		{"28/06/2022 23:59:59"},
		{"24.10.2022"},
		{"2022-06-29 14:08:21+03"},
		{"31.8.2025 00:00:00"},
		{"01-10-2025"},
		{"20-Apr-2023 03:28:40"},
		{"2022-12-08 14:00:00 CLST"},
		{"December  2 2022"},
		{"02/28/2025"},
		{"April 10 2023"},
		{"2025-Dec-11"},
		{"2025-Dec-11."},
		{"2024-06-05 00:00:00 (UTC+8)"},
	}

	for _, tt := range tests {
		tt := tt // NOTE: https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		t.Run(tt.date, func(t *testing.T) {
			t.Parallel() // marks each test case as capable of running in parallel with each other
			_, err := parseUnknownDate(tt.date)
			if err != nil {
				t.Fatalf("%s should not error out. error: %v", tt.date, err)
			}
		})
	}
}

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
