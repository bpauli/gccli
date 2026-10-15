package cmd

import (
	"errors"
	"fmt"
	"html"
	"slices"
	"strings"

	"github.com/alecthomas/kong"
)

// referenceGroup is one card on the docs site and one ### section in Markdown.
type referenceGroup struct {
	Title    string
	Commands []string
}

type refParam struct {
	Name string
	Help string
}

type refCommand struct {
	Synopsis string
	Help     string
	Alias    string
	Params   []refParam
}

type refGroup struct {
	Title    string
	Commands []refCommand
}

const (
	referenceBegin   = "<!-- gccli:commands:begin -->"
	referenceEnd     = "<!-- gccli:commands:end -->"
	referenceComment = "<!-- Generated from the command tree. Do not edit. Run: go test ./internal/cmd -run TestDocsUpToDate -update -->"
)

func buildReference(root *kong.Node, groups []referenceGroup) ([]refGroup, error) {
	leaves := map[string][]refCommand{}
	var topLevel []string
	for _, info := range collectCommands(root, nil) {
		if len(info.Path) == 1 {
			topLevel = append(topLevel, info.Path[0])
		}
		node := findNode(root, info.Path)
		if !node.Leaf() {
			continue
		}
		leaves[info.Path[0]] = append(leaves[info.Path[0]], referenceCommand(node, info.Path))
	}

	grouped := map[string]bool{}
	out := make([]refGroup, 0, len(groups))
	for _, g := range groups {
		rg := refGroup{Title: g.Title}
		for _, name := range g.Commands {
			if !slices.Contains(topLevel, name) {
				return nil, fmt.Errorf("reference group %q names command %q, which the command tree does not have", g.Title, name)
			}
			grouped[name] = true
			rg.Commands = append(rg.Commands, leaves[name]...)
		}
		out = append(out, rg)
	}
	for _, name := range topLevel {
		if !grouped[name] {
			return nil, fmt.Errorf("command %q is in no reference group: add it to referenceGroups in internal/cmd/docs_gen.go", name)
		}
	}
	return out, nil
}

func referenceCommand(node *kong.Node, path []string) refCommand {
	var flags []*kong.Flag
	for _, f := range node.Flags {
		if !f.Hidden && f.Name != "help" {
			flags = append(flags, f)
		}
	}

	var usage []string
	var cmd refCommand
	for _, f := range flags {
		if f.Required {
			usage = append(usage, f.Summary())
		}
	}
	for _, arg := range node.Positional {
		usage = append(usage, arg.Summary())
		cmd.Params = append(cmd.Params, refParam{Name: arg.Summary(), Help: withEnumNote(arg.Help, arg.Enum)})
	}
	for _, f := range flags {
		cmd.Params = append(cmd.Params, refParam{Name: f.String(), Help: withEnumNote(f.Help, f.Enum)})
	}
	if slices.ContainsFunc(flags, func(f *kong.Flag) bool { return !f.Required }) {
		usage = append(usage, "[flags]")
	}

	synopsis := func(path []string) string {
		return strings.Join(append(append([]string{"gccli"}, path...), usage...), " ")
	}
	cmd.Synopsis = synopsis(path)
	cmd.Help = node.Help
	if node.Parent != nil && node.Parent.DefaultCmd == node {
		cmd.Alias = synopsis(path[:len(path)-1])
	}
	return cmd
}

func withEnumNote(help, enum string) string {
	if enum == "" {
		return help
	}
	values := strings.Split(enum, ",")
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}
	if !slices.ContainsFunc(values, func(v string) bool { return !strings.Contains(help, v) }) {
		return help
	}
	return strings.TrimSpace(help + " One of: " + strings.Join(values, ", ") + ".")
}

func renderMarkdownReference(groups []refGroup) string {
	var b strings.Builder
	b.WriteString(referenceComment + "\n\n## Command Reference\n")
	for _, g := range groups {
		fmt.Fprintf(&b, "\n### %s\n\n", g.Title)
		for _, c := range g.Commands {
			fmt.Fprintf(&b, "- `%s`: %s", c.Synopsis, c.Help)
			if c.Alias != "" {
				fmt.Fprintf(&b, " Also runs as `%s`.", c.Alias)
			}
			b.WriteString("\n")
			for _, p := range c.Params {
				fmt.Fprintf(&b, "  - `%s`: %s\n", p.Name, p.Help)
			}
		}
	}
	return b.String()
}

func renderHTMLReference(groups []refGroup) string {
	const indent = "          "
	var b strings.Builder
	b.WriteString(indent + referenceComment + "\n")
	for i, g := range groups {
		class := "cmd-group fade-in"
		if i == 0 {
			class = "cmd-group is-open fade-in"
		}
		fmt.Fprintf(&b, "\n%s<div class=\"%s\">\n", indent, class)
		fmt.Fprintf(&b, "%s  <div class=\"cmd-header\"><h3>%s</h3><span class=\"cmd-arrow\">&#x25B6;</span></div>\n", indent, html.EscapeString(g.Title))
		fmt.Fprintf(&b, "%s  <div class=\"cmd-body\">\n", indent)
		for _, c := range g.Commands {
			desc := html.EscapeString(c.Help)
			if c.Alias != "" {
				desc += " Also runs as <code>" + html.EscapeString(c.Alias) + "</code>."
			}
			for _, p := range c.Params {
				desc += "<br><code>" + html.EscapeString(p.Name) + "</code> " + html.EscapeString(p.Help)
			}
			fmt.Fprintf(&b, "%s    <div class=\"cmd-row\"><div class=\"cmd\">%s</div><div class=\"desc\">%s</div></div>\n", indent, html.EscapeString(c.Synopsis), desc)
		}
		fmt.Fprintf(&b, "%s  </div>\n%s</div>\n", indent, indent)
	}
	return b.String()
}

func spliceReference(doc, block string) (string, error) {
	if strings.Count(doc, referenceBegin) != 1 || strings.Count(doc, referenceEnd) != 1 {
		return "", fmt.Errorf("want exactly one %s and one %s line", referenceBegin, referenceEnd)
	}
	begin := strings.Index(doc, referenceBegin)
	end := strings.Index(doc, referenceEnd)
	if end < begin {
		return "", errors.New(referenceEnd + " comes before " + referenceBegin)
	}
	nl := strings.IndexByte(doc[begin:], '\n')
	if nl < 0 || begin+nl > end {
		return "", errors.New(referenceBegin + " and " + referenceEnd + " must be on separate lines")
	}
	contentStart := begin + nl + 1
	contentEnd := strings.LastIndexByte(doc[:end], '\n') + 1
	return doc[:contentStart] + block + doc[contentEnd:], nil
}
