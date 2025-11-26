// Package main implements OpenAPI specification coverage analysis. It loads the API
// spec, compares registered handlers vs. documented endpoints, and prints a coverage
// summary plus detail lists for missing/extra paths and description gaps.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mauriciomferz/Gauth_go/internal/specgen"
)

func main() {
	var specPath string
	var out string
	flag.StringVar(&specPath, "spec", "docs/openapi.yaml", "Path to existing OpenAPI spec YAML")
	flag.StringVar(&out, "out", "docs/openapi.coverage.json", "Output JSON report path")
	flag.Parse()
	rep, err := specgen.GenerateCoverage(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "spec coverage error: %v\n", err)
		os.Exit(1)
	}
	if err := specgen.WriteReport(rep, out); err != nil {
		fmt.Fprintf(os.Stderr, "write report error: %v\n", err)
		os.Exit(1)
	}
	rel, _ := filepath.Rel(".", out)
	fmt.Printf("OpenAPI coverage report written: %s (missing=%d extra=%d coverage=%.2f "+
		"path_param_desc=%.2f query_param_desc=%.2f schema_prop_desc=%.2f op_example=%.2f err_example=%.2f)\n",
		rel, len(rep.MissingPaths), len(rep.ExtraSpecPaths), rep.CoveredRatio,
		rep.ParamDescriptionCoverage, rep.QueryParamDescriptionCoverage,
		rep.SchemaPropDescriptionCoverage, rep.OperationExampleCoverage, rep.ErrorResponseExampleCoverage)
	if len(rep.MissingPaths) > 0 {
		fmt.Println("Missing paths:")
		for _, p := range rep.MissingPaths {
			fmt.Printf("  - %s\n", p)
		}
	}
	if len(rep.ExtraSpecPaths) > 0 {
		fmt.Println("Extra spec paths (not registered):")
		for _, p := range rep.ExtraSpecPaths {
			fmt.Printf("  - %s\n", p)
		}
	}
	if len(rep.MissingParamDescriptions) > 0 {
		fmt.Println("Missing path parameter descriptions:")
		for _, mp := range rep.MissingParamDescriptions {
			fmt.Printf("  - %s\n", mp)
		}
	}
	if len(rep.MissingQueryParamDescriptions) > 0 {
		fmt.Println("Missing query parameter descriptions:")
		for _, mq := range rep.MissingQueryParamDescriptions {
			fmt.Printf("  - %s\n", mq)
		}
	}
	if len(rep.MissingSchemaPropDescriptions) > 0 {
		fmt.Println("Missing schema property descriptions:")
		for _, sp := range rep.MissingSchemaPropDescriptions {
			fmt.Printf("  - %s\n", sp)
		}
	}
	if len(rep.MissingOperationExamples) > 0 {
		fmt.Println("Missing operation examples:")
		for _, op := range rep.MissingOperationExamples {
			fmt.Printf("  - %s\n", op)
		}
	}
	if len(rep.MissingErrorResponseExamples) > 0 {
		fmt.Println("Missing error response examples:")
		for _, e := range rep.MissingErrorResponseExamples {
			fmt.Printf("  - %s\n", e)
		}
	}
}
