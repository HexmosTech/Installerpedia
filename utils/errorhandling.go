package utils

import (
	"ipm/types"
	"regexp"
	"strings"
)
var gitCloneRegex = regexp.MustCompile(`\bgit\s+clone\b`)
func IsExternallyManagedError(output string) bool {
    return strings.Contains(output, "externally-managed-environment") || 
           strings.Contains(output, "PEP 668")
}

func IsPermissionError(output string) bool {
    lower := strings.ToLower(output)
    return strings.Contains(lower, "permission denied") ||
        strings.Contains(lower, "operation not permitted") ||
        strings.Contains(lower, "eacces") ||
        strings.Contains(lower, "access to the path") && strings.Contains(lower, "denied") ||
        strings.Contains(lower, "unauthorizedaccessexception")
}

func IsGitCloneDestExistsError(output string) (bool, string) {
	lower := strings.ToLower(output)

	if strings.Contains(lower, "fatal: destination path") &&
		strings.Contains(lower, "already exists") &&
		strings.Contains(lower, "not an empty directory") {

		re := regexp.MustCompile(`destination path '([^']+)'`)
		m := re.FindStringSubmatch(output)
		if len(m) == 2 {
			return true, m[1]
		}
		return true, ""
	}

	return false, ""
}

func RemoveGitCloneCommands(cmds []types.Instruction) []types.Instruction {
	out := make([]types.Instruction, 0, len(cmds))
	for _, c := range cmds {
		if !gitCloneRegex.MatchString(c.Command) {
			out = append(out, c)
		}
	}
	return out
}

