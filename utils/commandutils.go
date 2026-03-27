package utils

import (
	"ipm/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func GetCommands(commands []types.Instruction) []string {
	cmds := make([]string, len(commands))
	for i, instr := range commands {
		cmds[i] = instr.Command
	}
	return cmds
}

func GetVenvBinPath(baseDir string, binary string) string {
	// Windows uses 'Scripts', Unix uses 'bin'
	subDir := "bin"
	if runtime.GOOS == "windows" {
		subDir = "Scripts"
	}

	path := filepath.Join(baseDir, ".venv", subDir, binary)
	if runtime.GOOS == "windows" && binary == "python" {
		path += ".exe"
	}

	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

func TruncateDesc(s string) string {
	const max = 250

	s = strings.TrimSpace(s)
	r := []rune(s)

	if len(r) <= max {
		return s
	}

	return string(r[:max]) + "…"
}
