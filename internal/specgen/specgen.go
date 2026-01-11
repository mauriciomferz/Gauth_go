package specgen

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	webpkg "github.com/mauriciomferz/AgentAuth/web"
	"gopkg.in/yaml.v3"
)

// CoverageReport captures comparison between live registered routes and OpenAPI spec paths.
type CoverageReport struct {
	GeneratedAt    time.Time `json:"generated_at"`
	SpecPath       string    `json:"spec_path"`
	TotalRoutes    int       `json:"total_routes"`
	TotalSpecPaths int       `json:"total_spec_paths"`
	MissingPaths   []string  `json:"missing_paths"`
	ExtraSpecPaths []string  `json:"extra_spec_paths"`
	CoveredRatio   float64   `json:"covered_ratio"`
	// Path parameter description coverage (only considers parameters with in: path)
	PathParamsTotal           int      `json:"path_params_total"`
	PathParamsWithDescription int      `json:"path_params_with_description"`
	MissingParamDescriptions  []string `json:"missing_param_descriptions"`
	ParamDescriptionCoverage  float64  `json:"parameter_description_coverage"`
	// Query parameter description coverage (in: query)
	QueryParamsTotal              int      `json:"query_params_total"`
	QueryParamsWithDescription    int      `json:"query_params_with_description"`
	MissingQueryParamDescriptions []string `json:"missing_query_param_descriptions"`
	QueryParamDescriptionCoverage float64  `json:"query_parameter_description_coverage"`
	// Component schema property description coverage
	SchemaPropsTotal              int      `json:"schema_props_total"`
	SchemaPropsWithDescription    int      `json:"schema_props_with_description"`
	MissingSchemaPropDescriptions []string `json:"missing_schema_prop_descriptions"`
	SchemaPropDescriptionCoverage float64  `json:"schema_property_description_coverage"`
	// Operation example coverage (at least one example in request or any response content)
	OperationsTotal          int      `json:"operations_total"`
	OperationsWithExample    int      `json:"operations_with_example"`
	MissingOperationExamples []string `json:"missing_operation_examples"`
	OperationExampleCoverage float64  `json:"operation_example_coverage"`
	// Error response example coverage (4xx/5xx responses must have at least one example)
	ErrorResponsesTotal          int      `json:"error_responses_total"`
	ErrorResponsesWithExample    int      `json:"error_responses_with_example"`
	MissingErrorResponseExamples []string `json:"missing_error_response_examples"`
	ErrorResponseExampleCoverage float64  `json:"error_response_example_coverage"`
}

// GenerateCoverage loads the existing OpenAPI spec and computes coverage against the live server's routes.
//
//nolint:gocyclo // Spec coverage report generation
func GenerateCoverage(specPath string) (*CoverageReport, error) {
	// Ensure JWT route inclusion before server creation
	if err := os.Setenv("AGENTAUTH_USE_JWT_LIB", "1"); err != nil {
		return nil, fmt.Errorf("set AGENTAUTH_USE_JWT_LIB: %w", err)
	}
	bs := webpkg.NewBetaServer("0")
	// Read spec file
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("yaml parse: %w", err)
	}
	pathsObj, ok := doc["paths"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("spec missing paths object")
	}
	// Build spec path set and also collect path parameter description coverage
	specPaths := map[string]struct{}{}
	var pathParamsTotal, pathParamsWithDesc int
	var missingParamDescs []string
	var queryParamsTotal, queryParamsWithDesc int
	var missingQueryParamDescs []string
	// Operation example counters (locals)
	var operationsTotal, operationsWithExample int
	var missingOperationExamples []string
	// Error response example counters
	var errorResponsesTotal, errorResponsesWithExample int
	var missingErrorResponseExamples []string
	for p, raw := range pathsObj {
		specPaths[p] = struct{}{}
		// Each raw should be a map of operations (get, post, etc.)
		opsMap, _ := raw.(map[string]any)
		for method, v := range opsMap {
			opObj, ok := v.(map[string]any)
			if !ok {
				continue
			}
			// Operation example detection
			hasExample := false
			// requestBody examples
			if rb, ok := opObj["requestBody"].(map[string]any); ok {
				if cMap, ok := rb["content"].(map[string]any); ok {
					for _, mtRaw := range cMap {
						mtObj, ok := mtRaw.(map[string]any)
						if !ok {
							continue
						}
						if _, ex := mtObj["examples"]; ex {
							hasExample = true
							break
						}
						if _, ex := mtObj["example"]; ex {
							hasExample = true
							break
						}
					}
				}
			}
			// responses examples (if not already found) & error response coverage
			if respMap, ok := opObj["responses"].(map[string]any); ok {
				for code, rr := range respMap {
					rObj, ok := rr.(map[string]any)
					if !ok {
						continue
					}
					// Error response coverage: consider 4xx / 5xx numeric codes only
					if len(code) == 3 && (strings.HasPrefix(code, "4") || strings.HasPrefix(code, "5")) {
						errorResponsesTotal++
						hasErrExample := false
						if cMap, ok := rObj["content"].(map[string]any); ok {
							for _, mtRaw := range cMap {
								mtObj, ok := mtRaw.(map[string]any)
								if !ok {
									continue
								}
								if _, ex := mtObj["examples"]; ex {
									hasErrExample = true
									break
								}
								if _, ex := mtObj["example"]; ex {
									hasErrExample = true
									break
								}
							}
						}
						if hasErrExample {
							errorResponsesWithExample++
						} else {
							// operation identifier plus status code
							oid := fmt.Sprint(opObj["operationId"])
							if oid == "" || oid == "<nil>" {
								oid = strings.ToLower(method) + ":" + p
							}
							missingErrorResponseExamples = append(missingErrorResponseExamples, fmt.Sprintf("%s:%s", oid, code))
						}
					}
					// Aggregate operation-level example detection from any response content if not already flagged
					if !hasExample {
						if cMap, ok := rObj["content"].(map[string]any); ok {
							for _, mtRaw := range cMap {
								mtObj, ok := mtRaw.(map[string]any)
								if !ok {
									continue
								}
								if _, ex := mtObj["examples"]; ex {
									hasExample = true
									break
								}
								if _, ex := mtObj["example"]; ex {
									hasExample = true
									break
								}
							}
						}
					}
				}
			}
			repOperationsIdent := func() string {
				// Prefer operationId if present
				if oid, ok := opObj["operationId"].(string); ok && oid != "" {
					return oid
				}
				return strings.ToLower(method) + ":" + p
			}
			if hasExample {
				// will count later after loop; accumulate counters now
				_ = hasExample // will count later
				// We'll increment inside loop.
				// operationsTotal++ operationsWithExample++ etc.
				// We'll use local counters defined above.
			}
			paramsSlice, hasParams := opObj["parameters"].([]any)
			if !hasParams {
				continue
			}
			for _, prm := range paramsSlice {
				prmObj, ok := prm.(map[string]any)
				if !ok {
					continue
				}
				inLoc := strings.ToLower(fmt.Sprint(prmObj["in"]))
				if inLoc == "path" {
					pathParamsTotal++
					name := fmt.Sprint(prmObj["name"])
					desc, _ := prmObj["description"].(string)
					if strings.TrimSpace(desc) != "" {
						pathParamsWithDesc++
					} else {
						// record path + param name for clarity
						missingParamDescs = append(missingParamDescs, fmt.Sprintf("%s:%s", p, name))
					}
				} else if inLoc == "query" {
					queryParamsTotal++
					name := fmt.Sprint(prmObj["name"])
					desc, _ := prmObj["description"].(string)
					if strings.TrimSpace(desc) != "" {
						queryParamsWithDesc++
					} else {
						missingQueryParamDescs = append(missingQueryParamDescs, fmt.Sprintf("%s:%s", p, name))
					}
				}
			}
			// Increment operation counters
			operationsTotal++
			if hasExample {
				operationsWithExample++
			} else {
				missingOperationExamples = append(missingOperationExamples, repOperationsIdent())
			}
		}
	}
	// Component schemas property description coverage counters
	var schemaPropsTotal, schemaPropsWithDesc int
	var missingSchemaPropDescs []string
	compsObj, _ := doc["components"].(map[string]any)
	if compsObj != nil {
		schemasObj, _ := compsObj["schemas"].(map[string]any)
		for sname, sraw := range schemasObj {
			sobj, ok := sraw.(map[string]any)
			if !ok {
				continue
			}
			props, _ := sobj["properties"].(map[string]any)
			for pname, praw := range props {
				propObj, ok := praw.(map[string]any)
				if !ok {
					continue
				}
				schemaPropsTotal++
				desc := ""
				if d, ok := propObj["description"].(string); ok {
					desc = d
				}
				if strings.TrimSpace(desc) != "" {
					schemaPropsWithDesc++
				} else {
					missingSchemaPropDescs = append(missingSchemaPropDescs, fmt.Sprintf("%s.%s", sname, pname))
				}
			}
		}
	}
	// Collect live routes via accessor
	liveFiltered := map[string]struct{}{}
	// regex to find :param segments
	colonParamRe := regexp.MustCompile(`/:([A-Za-z0-9_]+)`) // e.g., /foo/:id -> /foo/{id}
	for _, r := range bs.Routes() {
		if strings.HasPrefix(r.Path, "/api/") ||
			strings.HasPrefix(r.Path, "/.well-known/") ||
			r.Path == "/openapi.yaml" ||
			r.Path == "/api/v1/openapi" {
			norm := colonParamRe.ReplaceAllString(r.Path, "/{$1}")
			liveFiltered[norm] = struct{}{}
		}
	}
	// Compute missing and extra
	var missing []string
	for p := range liveFiltered {
		if _, ok := specPaths[p]; !ok {
			missing = append(missing, p)
		}
	}
	var extra []string
	for p := range specPaths {
		if _, ok := liveFiltered[p]; !ok {
			extra = append(extra, p)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	covered := float64(len(liveFiltered)-len(missing)) / float64(len(liveFiltered))
	var paramCoverage float64
	if pathParamsTotal > 0 {
		paramCoverage = float64(pathParamsWithDesc) / float64(pathParamsTotal)
	} else {
		paramCoverage = 1
	}
	var queryParamCoverage float64
	if queryParamsTotal > 0 {
		queryParamCoverage = float64(queryParamsWithDesc) / float64(queryParamsTotal)
	} else {
		queryParamCoverage = 1
	}
	rep := &CoverageReport{
		GeneratedAt:                   time.Now().UTC(),
		SpecPath:                      specPath,
		TotalRoutes:                   len(liveFiltered),
		TotalSpecPaths:                len(specPaths),
		MissingPaths:                  missing,
		ExtraSpecPaths:                extra,
		CoveredRatio:                  covered,
		PathParamsTotal:               pathParamsTotal,
		PathParamsWithDescription:     pathParamsWithDesc,
		MissingParamDescriptions:      missingParamDescs,
		ParamDescriptionCoverage:      paramCoverage,
		QueryParamsTotal:              queryParamsTotal,
		QueryParamsWithDescription:    queryParamsWithDesc,
		MissingQueryParamDescriptions: missingQueryParamDescs,
		QueryParamDescriptionCoverage: queryParamCoverage,
		SchemaPropsTotal:              schemaPropsTotal,
		SchemaPropsWithDescription:    schemaPropsWithDesc,
		MissingSchemaPropDescriptions: missingSchemaPropDescs,
		OperationsTotal:               operationsTotal,
		OperationsWithExample:         operationsWithExample,
		MissingOperationExamples:      missingOperationExamples,
		ErrorResponsesTotal:           errorResponsesTotal,
		ErrorResponsesWithExample:     errorResponsesWithExample,
		MissingErrorResponseExamples:  missingErrorResponseExamples,
	}
	if schemaPropsTotal > 0 {
		rep.SchemaPropDescriptionCoverage = float64(schemaPropsWithDesc) / float64(schemaPropsTotal)
	} else {
		rep.SchemaPropDescriptionCoverage = 1
	}
	if operationsTotal > 0 {
		rep.OperationExampleCoverage = float64(operationsWithExample) / float64(operationsTotal)
	} else {
		rep.OperationExampleCoverage = 1
	}
	if errorResponsesTotal > 0 {
		rep.ErrorResponseExampleCoverage = float64(errorResponsesWithExample) / float64(errorResponsesTotal)
	} else {
		rep.ErrorResponseExampleCoverage = 1
	}
	return rep, nil
}

// WriteReport writes coverage report as prettified JSON to the given output path.
func WriteReport(rep *CoverageReport, outPath string) error {
	b, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, b, 0o600)
}
