package output

import (
	"io"
	"testing"
)

// BenchmarkPrinterJSONSmall measures pretty-print of a small typical response
// (Account.Get shape). Every command that returns JSON goes through this path.
func BenchmarkPrinterJSONSmall(b *testing.B) {
	p := New(io.Discard, io.Discard)
	payload := map[string]any{
		"username": "1000000001",
		"name":     "Acme Co",
		"email":    "ops@acme.example",
		"cash":     142.18,
		"callerId": "2015551234",
		"timezone": "America/Chicago",
	}
	b.ReportAllocs()
	for range b.N {
		if err := p.JSON(payload); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPrinterJSONList measures pretty-print of a list response — the
// shape `numbers list` returns. Tests the marshal+indent path against a
// realistic payload size (~25 records).
func BenchmarkPrinterJSONList(b *testing.B) {
	p := New(io.Discard, io.Discard)
	numbers := make([]map[string]any, 25)
	for i := range numbers {
		numbers[i] = map[string]any{
			"number":     "2015551234",
			"route":      4,
			"cnam":       true,
			"smsEnabled": true,
			"forward":    "",
			"trunk":      "voicetel-east-1",
		}
	}
	payload := map[string]any{"numbers": numbers}
	b.ReportAllocs()
	for range b.N {
		if err := p.JSON(payload); err != nil {
			b.Fatal(err)
		}
	}
}
