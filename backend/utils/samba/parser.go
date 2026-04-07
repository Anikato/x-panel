package samba

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Section struct {
	Name    string
	Params  map[string]string
	Lines   []Line // preserves original order, comments, blanks
}

type Line struct {
	Type    string // "comment", "blank", "param"
	Raw     string // original text (for comments/blanks)
	Key     string
	Value   string
}

type Config struct {
	Sections []*Section
}

func Parse(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := &Config{}
	var current *Section

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)

		if trimmed == "" {
			if current != nil {
				current.Lines = append(current.Lines, Line{Type: "blank", Raw: raw})
			}
			continue
		}

		if trimmed[0] == '#' || trimmed[0] == ';' {
			if current != nil {
				current.Lines = append(current.Lines, Line{Type: "comment", Raw: raw})
			}
			continue
		}

		if trimmed[0] == '[' && trimmed[len(trimmed)-1] == ']' {
			name := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			current = &Section{
				Name:   name,
				Params: make(map[string]string),
			}
			cfg.Sections = append(cfg.Sections, current)
			continue
		}

		if current != nil {
			if idx := strings.IndexByte(trimmed, '='); idx > 0 {
				key := strings.TrimSpace(trimmed[:idx])
				val := strings.TrimSpace(trimmed[idx+1:])
				current.Params[key] = val
				current.Lines = append(current.Lines, Line{Type: "param", Raw: raw, Key: key, Value: val})
			} else {
				current.Lines = append(current.Lines, Line{Type: "comment", Raw: raw})
			}
		}
	}
	return cfg, scanner.Err()
}

func (c *Config) GetSection(name string) *Section {
	for _, s := range c.Sections {
		if strings.EqualFold(s.Name, name) {
			return s
		}
	}
	return nil
}

func (c *Config) GetGlobal() *Section {
	return c.GetSection("global")
}

func (c *Config) GetShares() []*Section {
	skip := map[string]bool{"global": true, "homes": true, "printers": true, "print$": true}
	var shares []*Section
	for _, s := range c.Sections {
		if !skip[strings.ToLower(s.Name)] {
			shares = append(shares, s)
		}
	}
	return shares
}

func (c *Config) RemoveSection(name string) {
	for i, s := range c.Sections {
		if strings.EqualFold(s.Name, name) {
			c.Sections = append(c.Sections[:i], c.Sections[i+1:]...)
			return
		}
	}
}

func (c *Config) AddSection(sec *Section) {
	c.RemoveSection(sec.Name)
	c.Sections = append(c.Sections, sec)
}

func NewShareSection(name, path, comment string, writable, guestOK bool, validUsers string) *Section {
	sec := &Section{
		Name:   name,
		Params: make(map[string]string),
	}
	sec.Params["path"] = path
	if comment != "" {
		sec.Params["comment"] = comment
	}
	sec.Params["browseable"] = "yes"
	if writable {
		sec.Params["writable"] = "yes"
	} else {
		sec.Params["read only"] = "yes"
	}
	if guestOK {
		sec.Params["guest ok"] = "yes"
	} else {
		sec.Params["guest ok"] = "no"
	}
	if validUsers != "" {
		sec.Params["valid users"] = validUsers
	}

	for k, v := range sec.Params {
		sec.Lines = append(sec.Lines, Line{Type: "param", Key: k, Value: v})
	}
	return sec
}

func (c *Config) Write(path string) error {
	var b strings.Builder
	for i, sec := range c.Sections {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("[%s]\n", sec.Name))

		if len(sec.Lines) > 0 {
			written := make(map[string]bool)
			for _, line := range sec.Lines {
				switch line.Type {
				case "comment", "blank":
					b.WriteString(line.Raw + "\n")
				case "param":
					val, exists := sec.Params[line.Key]
					if !exists {
						continue
					}
					if written[line.Key] {
						continue
					}
					b.WriteString(fmt.Sprintf("   %s = %s\n", line.Key, val))
					written[line.Key] = true
				}
			}
			for k, v := range sec.Params {
				if !written[k] {
					b.WriteString(fmt.Sprintf("   %s = %s\n", k, v))
				}
			}
		} else {
			for k, v := range sec.Params {
				b.WriteString(fmt.Sprintf("   %s = %s\n", k, v))
			}
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0644)
}
