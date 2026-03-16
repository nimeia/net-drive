package benchgate

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Threshold struct {
	MaxNsPerOp     float64 `json:"max_ns_per_op"`
	MaxBytesPerOp  float64 `json:"max_bytes_per_op,omitempty"`
	MaxAllocsPerOp float64 `json:"max_allocs_per_op,omitempty"`
}

type Result struct {
	Name          string
	NsPerOp       float64
	BytesPerOp    float64
	AllocsPerOp   float64
	RawLine       string
	HasNsPerOp    bool
	HasBytesPerOp bool
	HasAllocs     bool
}

type Violation struct {
	Name   string
	Metric string
	Got    float64
	Limit  float64
}

var benchmarkNameSuffix = regexp.MustCompile(`^(Benchmark\S+?)(?:-\d+)?$`)

func LoadThresholds(r io.Reader) (map[string]Threshold, error) {
	var thresholds map[string]Threshold
	if err := json.NewDecoder(r).Decode(&thresholds); err != nil {
		return nil, err
	}
	return thresholds, nil
}

func ParseOutput(r io.Reader) (map[string]Result, error) {
	results := make(map[string]Result)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		name := normalizeBenchmarkName(fields[0])
		res := Result{Name: name, RawLine: line}
		for i := 0; i < len(fields)-1; i++ {
			value, err := strconv.ParseFloat(strings.TrimSpace(fields[i]), 64)
			if err != nil {
				continue
			}
			switch fields[i+1] {
			case "ns/op":
				res.NsPerOp = value
				res.HasNsPerOp = true
			case "B/op":
				res.BytesPerOp = value
				res.HasBytesPerOp = true
			case "allocs/op":
				res.AllocsPerOp = value
				res.HasAllocs = true
			}
		}
		if res.HasNsPerOp {
			results[name] = res
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func Evaluate(results map[string]Result, thresholds map[string]Threshold) ([]string, []Violation) {
	missing := make([]string, 0)
	violations := make([]Violation, 0)
	for name, limit := range thresholds {
		res, ok := results[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		if limit.MaxNsPerOp > 0 && res.NsPerOp > limit.MaxNsPerOp {
			violations = append(violations, Violation{Name: name, Metric: "ns/op", Got: res.NsPerOp, Limit: limit.MaxNsPerOp})
		}
		if limit.MaxBytesPerOp > 0 && res.HasBytesPerOp && res.BytesPerOp > limit.MaxBytesPerOp {
			violations = append(violations, Violation{Name: name, Metric: "B/op", Got: res.BytesPerOp, Limit: limit.MaxBytesPerOp})
		}
		if limit.MaxAllocsPerOp > 0 && res.HasAllocs && res.AllocsPerOp > limit.MaxAllocsPerOp {
			violations = append(violations, Violation{Name: name, Metric: "allocs/op", Got: res.AllocsPerOp, Limit: limit.MaxAllocsPerOp})
		}
	}
	sort.Strings(missing)
	sort.Slice(violations, func(i, j int) bool {
		if violations[i].Name == violations[j].Name {
			return violations[i].Metric < violations[j].Metric
		}
		return violations[i].Name < violations[j].Name
	})
	return missing, violations
}

func FormatReport(missing []string, violations []Violation) string {
	var b strings.Builder
	if len(missing) == 0 && len(violations) == 0 {
		return "benchmark gate passed"
	}
	if len(missing) > 0 {
		b.WriteString("missing benchmarks:\n")
		for _, name := range missing {
			fmt.Fprintf(&b, "- %s\n", name)
		}
	}
	if len(violations) > 0 {
		b.WriteString("benchmark threshold violations:\n")
		for _, v := range violations {
			fmt.Fprintf(&b, "- %s %s got=%.2f limit=%.2f\n", v.Name, v.Metric, v.Got, v.Limit)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func normalizeBenchmarkName(name string) string {
	match := benchmarkNameSuffix.FindStringSubmatch(name)
	if len(match) == 2 {
		return match[1]
	}
	return name
}
