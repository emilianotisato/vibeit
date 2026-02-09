package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
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
	{Name: "tmux", Command: "tmux", Required: true, MinVer: "3.0"},
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

	printTmuxDetachStatus()

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
	case "tmux":
		cmd = exec.Command("tmux", "-V")
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
	case "tmux":
		// "tmux 3.4"
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
	fmt.Println("  sudo pacman -S git neovim tmux")
	fmt.Println("  yay -S lazygit  # or paru -S lazygit")
	fmt.Println()
	fmt.Println("Ubuntu/Debian:")
	fmt.Println("  sudo apt install git neovim tmux")
	fmt.Println("  # lazygit: https://github.com/jesseduffield/lazygit#installation")
	fmt.Println()
	fmt.Println("For more info, visit: https://github.com/emilianotisato/vibeit")
}

func printTmuxDetachStatus() {
	fmt.Println()
	fmt.Println("Tmux keybinding check:")

	ok, source := checkTmuxDetachBinding()
	if ok {
		fmt.Printf("  ✓ Ctrl+\\\\ detach binding found (%s)\n", source)
	} else {
		fmt.Println("  ⚠ Ctrl+\\ detach binding not detected")
		fmt.Println("    Add to ~/.tmux.conf: bind-key -n C-\\\\ detach-client")
		fmt.Println("    Then reload: tmux source-file ~/.tmux.conf")
	}

	fmt.Println("  Verify: tmux list-keys -T root | grep -F 'C-\\\\'")
}

func checkTmuxDetachBinding() (bool, string) {
	// First check the live tmux root table (if a server is running).
	cmd := exec.Command("tmux", "list-keys", "-T", "root")
	out, err := cmd.Output()
	if err == nil {
		if hasDetachBinding(string(out)) {
			return true, "running tmux server"
		}
		return false, "running tmux server"
	}

	// If no server is running, check persisted config for the canonical binding.
	home, err := os.UserHomeDir()
	if err != nil {
		return false, "unable to resolve home directory"
	}
	confPath := home + "/.tmux.conf"
	data, err := os.ReadFile(confPath)
	if err != nil {
		return false, "~/.tmux.conf not found"
	}
	if hasDetachBinding(string(data)) {
		return true, "~/.tmux.conf"
	}
	return false, "~/.tmux.conf"
}

func hasDetachBinding(text string) bool {
	// Accept both config lines (`bind-key -n C-\\ detach-client`) and
	// `tmux list-keys` output (`bind-key -T root C-\\ detach-client`).
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?m)^\s*bind(?:-key)?\s+-n\s+C-\\+\s+detach-client\b`),
		regexp.MustCompile(`(?m)^\s*bind-key\s+-T\s+root\s+C-\\+\s+detach-client\b`),
	}
	for _, re := range patterns {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}
