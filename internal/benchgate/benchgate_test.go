package benchgate

import (
	"strings"
	"testing"
)

func TestParseOutputAndEvaluate(t *testing.T) {
	output := strings.NewReader(`goos: linux
BenchmarkMetadataLookupHot-12          22000        3500 ns/op        64 B/op         2 allocs/op
BenchmarkEncodeDecodeFrame-12          18000        1200 ns/op        32 B/op         1 allocs/op
`)
	results, err := ParseOutput(output)
	if err != nil {
		t.Fatalf("ParseOutput() error = %v", err)
	}
	if got := results["BenchmarkMetadataLookupHot"].NsPerOp; got != 3500 {
		t.Fatalf("MetadataLookupHot ns/op = %.0f, want 3500", got)
	}
	thresholds := map[string]Threshold{
		"BenchmarkMetadataLookupHot": {MaxNsPerOp: 4000, MaxAllocsPerOp: 4},
		"BenchmarkEncodeDecodeFrame": {MaxNsPerOp: 1000},
		"BenchmarkMissing":           {MaxNsPerOp: 1},
	}
	missing, violations := Evaluate(results, thresholds)
	if len(missing) != 1 || missing[0] != "BenchmarkMissing" {
		t.Fatalf("missing = %+v, want [BenchmarkMissing]", missing)
	}
	if len(violations) != 1 {
		t.Fatalf("violations len = %d, want 1", len(violations))
	}
	if violations[0].Name != "BenchmarkEncodeDecodeFrame" || violations[0].Metric != "ns/op" {
		t.Fatalf("unexpected violation = %+v", violations[0])
	}
}

func TestFormatReport(t *testing.T) {
	report := FormatReport([]string{"BenchmarkMissing"}, []Violation{{Name: "BenchmarkHot", Metric: "ns/op", Got: 12, Limit: 10}})
	if !strings.Contains(report, "missing benchmarks") || !strings.Contains(report, "BenchmarkHot ns/op") {
		t.Fatalf("FormatReport() = %q", report)
	}
	if got := FormatReport(nil, nil); got != "benchmark gate passed" {
		t.Fatalf("FormatReport(nil,nil) = %q", got)
	}
}
