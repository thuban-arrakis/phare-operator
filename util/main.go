package util

import (
	"fmt"
	"os"
	"strings"

	"github.com/localcorp/phare-controller/util/yaml-diff/yamldiff"
)

// Update the signature to indicate it returns a string
func Diff(desired, current string) string {
	yamls1, err := yamldiff.Load(desired)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v", err)
		return ""
	}

	yamls2, err := yamldiff.Load(current)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v", err)
		return ""
	}

	var diffs []string
	for _, diff := range yamldiff.Do(yamls1, yamls2) {
		// The two lines below seem redundant as they do the same thing twice.
		// Keeping only the second line which appends to the diffs slice would suffice.
		fmt.Println(diff.Dump())
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
