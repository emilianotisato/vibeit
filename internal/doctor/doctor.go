package doctor

import (
	"fmt"
	"os/exec"
	"strings"
)

type Dependency struct {
	Name     string
	Command  string
	Required bool
	MinVer   string
}

var dependencies = []Dependency{
	{Name: "git", Command: "git", Required: true, MinVer: "2.0"},
	{Name: "zellij", Command: "zellij", Required: true, MinVer: "0.40"},
	{Name: "neovim", Command: "nvim", Required: true, MinVer: "0.9"},
	{Name: "lazygit", Command: "lazygit", Required: false, MinVer: "0.40"},
}

func Run() int {
	fmt.Println("vibeit doctor")
	fmt.Println("=============")
	fmt.Println()

	allOk := true

	for _, dep := range dependencies {
		status, version := checkDependency(dep)
		printStatus(dep, status, version)
		if !status && dep.Required {
			allOk = false
		}
	}

	fmt.Println()

	if allOk {
		fmt.Println("All required dependencies are installed!")
		return 0
	}

	fmt.Println("Some required dependencies are missing. Please install them:")
	fmt.Println()
	printInstallInstructions()
	return 1
}

func checkDependency(dep Dependency) (bool, string) {
	path, err := exec.LookPath(dep.Command)
	if err != nil {
		return false, ""
	}

	// Try to get version
	version := getVersion(dep.Command)
	if version == "" {
		// Command exists but couldn't get version
		return path != "", "unknown"
	}

	return true, version
}

func getVersion(command string) string {
	var cmd *exec.Cmd

	switch command {
	case "git":
		cmd = exec.Command("git", "--version")
	case "zellij":
		cmd = exec.Command("zellij", "--version")
	case "nvim":
		cmd = exec.Command("nvim", "--version")
	case "lazygit":
		cmd = exec.Command("lazygit", "--version")
	default:
		return ""
	}

	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	return parseVersion(command, string(out))
}

func parseVersion(command, output string) string {
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return ""
	}

	line := lines[0]

	switch command {
	case "git":
		// "git version 2.47.1"
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			return parts[2]
		}
	case "zellij":
		// "zellij 0.41.2"
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			return parts[1]
		}
	case "nvim":
		// "NVIM v0.10.2"
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			return strings.TrimPrefix(parts[1], "v")
		}
	case "lazygit":
		// Various formats, try to extract version
		if strings.Contains(line, "version=") {
			for _, part := range strings.Split(line, ",") {
				if strings.HasPrefix(strings.TrimSpace(part), "version=") {
					return strings.TrimPrefix(strings.TrimSpace(part), "version=")
				}
			}
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			return parts[1]
		}
	}

	return strings.TrimSpace(line)
}

func printStatus(dep Dependency, installed bool, version string) {
	reqStr := ""
	if !dep.Required {
		reqStr = " (optional)"
	}

	if installed {
		fmt.Printf("  ✓ %s%s: %s\n", dep.Name, reqStr, version)
	} else {
		if dep.Required {
			fmt.Printf("  ✗ %s: NOT FOUND\n", dep.Name)
		} else {
			fmt.Printf("  ⚠ %s%s: NOT FOUND\n", dep.Name, reqStr)
		}
	}
}

func printInstallInstructions() {
	fmt.Println("Arch Linux:")
	fmt.Println("  sudo pacman -S git neovim zellij")
	fmt.Println("  yay -S lazygit  # or paru -S lazygit")
	fmt.Println()
	fmt.Println("Ubuntu/Debian:")
	fmt.Println("  sudo apt install git neovim")
	fmt.Println("  # zellij: https://zellij.dev/documentation/installation")
	fmt.Println("  # lazygit: https://github.com/jesseduffield/lazygit#installation")
	fmt.Println()
	fmt.Println("For more info, visit: https://github.com/emilianotisato/vibeit")
}
