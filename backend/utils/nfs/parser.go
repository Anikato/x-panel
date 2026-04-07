package nfs

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Export struct {
	Path    string
	Clients []Client
	Comment string
}

type Client struct {
	Host    string
	Options string
}

var clientRe = regexp.MustCompile(`(\S+?)\(([^)]*)\)`)

func Parse(path string) ([]Export, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var exports []Export
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] == '#' {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		export := Export{Path: parts[0]}
		rest := strings.Join(parts[1:], " ")
		matches := clientRe.FindAllStringSubmatch(rest, -1)
		for _, m := range matches {
			export.Clients = append(export.Clients, Client{
				Host:    m[1],
				Options: m[2],
			})
		}
		exports = append(exports, export)
	}
	return exports, scanner.Err()
}

func Write(path string, exports []Export) error {
	var b strings.Builder
	b.WriteString("# /etc/exports - NFS server exports\n")
	b.WriteString("# Managed by X-Panel. Manual edits are preserved on next panel write.\n\n")

	for _, e := range exports {
		if e.Comment != "" {
			b.WriteString(fmt.Sprintf("# %s\n", e.Comment))
		}
		b.WriteString(e.Path)
		for _, c := range e.Clients {
			b.WriteString(fmt.Sprintf(" %s(%s)", c.Host, c.Options))
		}
		b.WriteString("\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0644)
}

func AddExport(path string, export Export) error {
	exports, err := Parse(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for i, e := range exports {
		if e.Path == export.Path {
			exports[i] = export
			return Write(path, exports)
		}
	}
	exports = append(exports, export)
	return Write(path, exports)
}

func RemoveExport(configPath, exportPath string) error {
	exports, err := Parse(configPath)
	if err != nil {
		return err
	}
	var filtered []Export
	for _, e := range exports {
		if e.Path != exportPath {
			filtered = append(filtered, e)
		}
	}
	return Write(configPath, filtered)
}
