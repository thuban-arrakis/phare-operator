package util

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

// GetDiff calculates and returns the differences between two resources.
func GetDiff(existing, desired interface{}) string {
	// Convert objects to YAML
	// existingYAML := toYAML(existing)
	// desiredYAML := toYAML(desired)
	ex := existing
	de := desired
	diff := cmp.Diff(de, ex)
	// print("existingYAML: ", existingYAML)
	// print("desiredYAML: ", desiredYAML)
	// print(diff)
	return diff
}

// func colorizeDiff(diff string) string {
// 	// Assuming ANSI escape codes for coloring:
// 	// 31 is red (for deletions), 32 is green (for additions)
// 	lines := strings.Split(diff, "\n")
// 	for i, line := range lines {
// 		if strings.HasPrefix(line, "-") {
// 			lines[i] = fmt.Sprintf("\033[31m%s\033[0m", line)
// 		} else if strings.HasPrefix(line, "+") {
// 			lines[i] = fmt.Sprintf("\033[32m%s\033[0m", line)
// 		}
// 	}
// 	return strings.Join(lines, "\n")
// }

func toYAML(obj interface{}) string {
	data, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Sprintf("Error marshaling to YAML: %s", err)
	}
	return string(data)
}
