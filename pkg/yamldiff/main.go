package yamldiff

import (
	"fmt"
	"os"
	"strings"
)

// Update the signature to indicate it returns a string
func Diff(desired, current string) string {
	yamls1, err := Load(desired)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v", err)
		return ""
	}

	yamls2, err := Load(current)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v", err)
		return ""
	}

	var diffs []string
	for _, diff := range Do(yamls1, yamls2) {
		diffs = append(diffs, colorizeDiff(diff.Dump()))
	}

	return strings.Join(diffs, "\n\n") // separate multiple diffs with double newline
}

func colorizeDiff(diff string) string {
	// Assuming ANSI escape codes for coloring:
	// 31 is red (for deletions), 32 is green (for additions)
	lines := strings.Split(diff, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "-") {
			lines[i] = fmt.Sprintf("\033[31m%s\033[0m", line)
		} else if strings.HasPrefix(line, "+") {
			lines[i] = fmt.Sprintf("\033[32m%s\033[0m", line)
		}
	}
	return strings.Join(lines, "\n")
}
