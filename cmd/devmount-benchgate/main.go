package main

import (
	"flag"
	"fmt"
	"os"

	"developer-mount/internal/benchgate"
)

func main() {
	inputPath := flag.String("input", "", "benchmark output file")
	thresholdsPath := flag.String("thresholds", "", "benchmark threshold JSON file")
	flag.Parse()
	if *inputPath == "" || *thresholdsPath == "" {
		fmt.Fprintln(os.Stderr, "-input and -thresholds are required")
		os.Exit(2)
	}

	inputFile, err := os.Open(*inputPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer inputFile.Close()

	thresholdFile, err := os.Open(*thresholdsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer thresholdFile.Close()

	results, err := benchgate.ParseOutput(inputFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	thresholds, err := benchgate.LoadThresholds(thresholdFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	missing, violations := benchgate.Evaluate(results, thresholds)
	report := benchgate.FormatReport(missing, violations)
	fmt.Println(report)
	if len(missing) > 0 || len(violations) > 0 {
		os.Exit(1)
	}
}
