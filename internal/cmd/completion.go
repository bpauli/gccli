package cmd

import (
	"fmt"
	"strings"

	"github.com/alecthomas/kong"
)

// CompletionCmd generates shell completion scripts.
type CompletionCmd struct {
	Shell string `arg:"" enum:"bash,zsh,fish,powershell" help:"Shell type (bash, zsh, fish, powershell)."`
}

func (c *CompletionCmd) Run(g *Globals) error {
	var script string
	switch c.Shell {
	case "bash":
		script = generateBash(g.Parser)
	case "zsh":
		script = generateZsh(g.Parser)
	case "fish":
		script = generateFish(g.Parser)
	case "powershell":
		script = generatePowerShell(g.Parser)
	default:
		return fmt.Errorf("unsupported shell: %s", c.Shell)
	}
	_, err := fmt.Fprint(g.Stdout, script)
	return err
}

// commandInfo represents a command or subcommand for completion generation.
type commandInfo struct {
	Path []string
	Help string
}

// flagInfo represents a flag for completion generation.
type flagInfo struct {
	Long   string
	Short  rune
	Help   string
	Enum   []string
	IsBool bool
}

// collectCommands recursively walks the kong node tree and collects non-hidden commands.
func collectCommands(node *kong.Node, path []string) []commandInfo {
	var cmds []commandInfo
	for _, child := range node.Children {
		if child.Hidden {
			continue
		}
		if child.Type != kong.CommandNode {
			continue
		}
		childPath := append(append([]string{}, path...), child.Name)
		cmds = append(cmds, commandInfo{
			Path: childPath,
			Help: child.Help,
		})
		cmds = append(cmds, collectCommands(child, childPath)...)
	}
	return cmds
}

// collectFlags returns non-hidden flags for the given node.
func collectFlags(node *kong.Node) []flagInfo {
	var flags []flagInfo
	for _, f := range node.Flags {
		if f.Hidden {
			continue
		}
		if f.Name == "help" {
			continue
		}
		fi := flagInfo{
			Long:   f.Name,
			Short:  f.Short,
			Help:   f.Help,
			IsBool: f.IsBool(),
		}
		if f.Enum != "" {
			fi.Enum = strings.Split(f.Enum, ",")
		}
		flags = append(flags, fi)
	}
	return flags
}

// escapeShellString escapes single quotes in a string for shell scripts.
func escapeShellString(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}
