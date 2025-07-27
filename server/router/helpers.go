package router

import (
	"fmt"
	"net/http"
	"strings"
)

// getPathValues extracts all path variables from a request based on a route pattern
func getPathValues(req *http.Request, pattern string) map[string]string {
	result := make(map[string]string)

	// Extract variable names from the pattern
	parts := strings.Split(pattern, "/")
	for _, part := range parts {
		// Check if this part is a variable (starts with { and ends with })
		if len(part) > 2 && part[0] == '{' && part[len(part)-1] == '}' {
			// Extract variable name without braces
			varName := part[1 : len(part)-1]

			// Skip any suffix like {$}
			if varName == "$" {
				continue
			}

			// Get the value from the request
			value := req.PathValue(varName)
			if value != "" {
				result[varName] = value
			}
		}
	}

	return result
}

// cleanPatternString removes methods and trailing {$} from a pattern string
func cleanPatternString(pattern string) string {
	// Split on whitespace to remove any method
	// Use Fields for whitespace on split
	parts := strings.Fields(pattern)
	if len(parts) > 1 {
		// If there's a method, we only care about the path part
		pattern = parts[1]
	}
	// Remove trailing /{$}
	clean := strings.TrimSuffix(pattern, "/{$}")
	// Remove trailing {$} without slash
	clean = strings.TrimSuffix(clean, "{$}")
	return clean
}

func interpolatePattern(pattern string, params map[string]string) string {
	result := pattern
	for key, value := range params {
		placeholder := fmt.Sprintf("{%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}
