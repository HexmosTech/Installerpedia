package utils

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/denisbrodbeck/machineid"
)

var (
	OS   = runtime.GOOS
	ARCH = runtime.GOARCH
)

var machineDistinctID string

func GetMachineDistinctID() string {
	if machineDistinctID != "" {
		return machineDistinctID
	}

	// App-specific hash so you don’t collide with other apps
	id, err := machineid.ProtectedID("ipm")
	if err != nil {
		// absolute fallback – still stable per run
		id = fmt.Sprintf("ipm-fallback-%s-%s", OS, ARCH)
	}

	machineDistinctID = id
	return machineDistinctID
}


func GetSupplementedOS() string {
    if runtime.GOOS == "linux" {
        if detailed := GetLinuxFamily(true); detailed != "" {
            return fmt.Sprintf("linux (%s)", detailed)
        }
    } else if runtime.GOOS == "darwin" {
        if detailed := GetMacDetailed(); detailed != "" {
            return fmt.Sprintf("darwin (%s)", detailed)
        }
    }
    return runtime.GOOS
}
// 2. Helper to detect the current system's family
func GetLinuxFamily(detailed bool) string {
	if runtime.GOOS != "linux" {
		return ""
	}
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	content := string(data)

	// Helper logic for internal sorting/base detection
	lowerContent := strings.ToLower(content)
	family := ""
	// 1. Debian Family (DEB)
	if strings.Contains(lowerContent, "debian") || 
	strings.Contains(lowerContent, "ubuntu") || 
	strings.Contains(lowerContent, "kali") || 
	strings.Contains(lowerContent, "pop") ||
	strings.Contains(lowerContent, "deepin") ||
	strings.Contains(lowerContent, "mint") ||     
	strings.Contains(lowerContent, "zorin") ||   
	strings.Contains(lowerContent, "elementary") || 
	strings.Contains(lowerContent, "pureos") ||  
	strings.Contains(lowerContent, "raspbian") || 
	strings.Contains(lowerContent, "uos") {
	family = "debian"

	// 2. Red Hat/SUSE Family (RPM)
	} else if strings.Contains(lowerContent, "fedora") || 
		strings.Contains(lowerContent, "rhel") || 
		strings.Contains(lowerContent, "centos") || 
		strings.Contains(lowerContent, "suse") || 
		strings.Contains(lowerContent, "sles") ||
		strings.Contains(lowerContent, "amzn") || 
		strings.Contains(lowerContent, "rocky") || 
		strings.Contains(lowerContent, "alma") || 
		strings.Contains(lowerContent, "oracle") ||
        strings.Contains(lowerContent, "nobara") || 
        strings.Contains(lowerContent, "mageia") { 
	family = "rpm"

	// 3. Arch Family (Pacman)
	} else if strings.Contains(lowerContent, "arch") || 
		strings.Contains(lowerContent, "manjaro") ||
		strings.Contains(lowerContent, "endeavour") {
	family = "arch"
	}
	if detailed {
		prettyName := ""
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				name := strings.TrimPrefix(line, "PRETTY_NAME=")
				prettyName = strings.Trim(name, "\"")
				break
			}
		}

		// Combine them: "Linux Mint 21.3, Debian based"
		if prettyName != "" && family != "" {
			return fmt.Sprintf("%s, %s based", prettyName, family)
		}
		if prettyName != "" {
			return prettyName
		}
		return family
	}

	return family
}


func GetMacDetailed() string {
    // 1. Read the system plist
    data, err := os.ReadFile("/System/Library/CoreServices/SystemVersion.plist")
    if err != nil {
        return "macOS"
    }
    content := string(data)

    // 2. Extract Version (e.g., 14.4.1)
    version := "Unknown"
    if strings.Contains(content, "ProductUserVisibleVersion") {
        parts := strings.Split(content, "ProductUserVisibleVersion")
        if len(parts) > 1 {
            subParts := strings.Split(parts[1], "<string>")
            if len(subParts) > 1 {
                version = strings.Split(subParts[1], "</string>")[0]
            }
        }
    }

    // 3. Map Version to Marketing Name (The "Family" equivalent)
    // macOS versions 11+ use the first number for the major release
    major := strings.Split(version, ".")[0]
    name := "macOS"
    
    switch major {
    case "15": name = "Sequoia"
    case "14": name = "Sonoma"
    case "13": name = "Ventura"
    case "12": name = "Monterey"
    case "11": name = "Big Sur"
    case "10":
        // For 10.x, we look at the second number
        if segments := strings.Split(version, "."); len(segments) > 1 {
            switch segments[1] {
            case "15": name = "Catalina"
            case "14": name = "Mojave"
            case "13": name = "High Sierra"
            default: name = "Mac OS X"
            }
        }
    }

    return fmt.Sprintf("%s %s", name, version)
}
