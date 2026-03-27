package version

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

//go:embed version.json
var versionData []byte

type Info struct {
    Version string `json:"version"`
}

// GetVersion returns the version string from version.json (embedded).
func GetVersion() string {
    var v Info
    if err := json.Unmarshal(versionData, &v); err != nil {
        return "unknown"
    }
    return v.Version
}



// PrintVersion neatly prints the version and suggests how to update.
func PrintVersion() {
    v := GetVersion()
    fmt.Printf("ipm version: %s\n", v)
    fmt.Println("Run 'ipm update' to check for updates.")
}

func Normalize(v string) string {
    v = strings.TrimSpace(v)
    v = strings.TrimPrefix(v, "v")
    return v
}

func CompareVersions(local, remote string) int {
    // remove leading v if present
    local = strings.TrimPrefix(local, "v")
    remote = strings.TrimPrefix(remote, "v")

    lParts := strings.Split(local, ".")
    rParts := strings.Split(remote, ".")

    for len(lParts) < 3 {
        lParts = append(lParts, "0")
    }
    for len(rParts) < 3 {
        rParts = append(rParts, "0")
    }

    for i := 0; i < 3; i++ {
        l, _ := strconv.Atoi(lParts[i])
        r, _ := strconv.Atoi(rParts[i])

        if l > r {
            return 1  // local is newer
        }
        if l < r {
            return -1 // remote is newer
        }
    }

    return 0 // same
}