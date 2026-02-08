package markdown

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Frontmatter holds the YAML frontmatter fields parsed from a markdown file.
type Frontmatter struct {
	ID string
}

// Parse splits a markdown document into its frontmatter and body content.
// If no frontmatter is present, an empty Frontmatter is returned with the
// full text as body.
func Parse(raw string) (Frontmatter, string) {
	fm, body, ok := splitFrontmatter(raw)
	if !ok {
		return Frontmatter{}, raw
	}
	return parseFrontmatterFields(fm), body
}

// Serialize writes frontmatter and body back into a complete markdown document.
func Serialize(fm Frontmatter, body string) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("id: %s\n", fm.ID))
	b.WriteString("---\n")
	if body != "" {
		b.WriteString(body)
	}
	return b.String()
}

// EnsureID returns the frontmatter with an ID, generating a new UUID if empty.
func EnsureID(fm Frontmatter) Frontmatter {
	if fm.ID == "" {
		fm.ID = uuid.New().String()
	}
	return fm
}

// ExtractTitle returns the title from the first # heading in the body.
// If no heading is found, it returns an empty string.
func ExtractTitle(body string) string {
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
	}
	return ""
}

// splitFrontmatter splits raw content on the --- delimiters.
// Returns the frontmatter block, the body, and whether frontmatter was found.
func splitFrontmatter(raw string) (string, string, bool) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "---") {
		return "", "", false
	}

	// Find the closing ---
	rest := trimmed[3:] // skip opening ---
	rest = strings.TrimLeft(rest, " \t")
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	} else if len(rest) > 1 && rest[0] == '\r' && rest[1] == '\n' {
		rest = rest[2:]
	}

	idx := strings.Index(rest, "---")
	if idx == -1 {
		return "", "", false
	}

	fm := rest[:idx]
	body := rest[idx+3:]
	// Trim the leading newline from body
	if len(body) > 0 && body[0] == '\n' {
		body = body[1:]
	} else if len(body) > 1 && body[0] == '\r' && body[1] == '\n' {
		body = body[2:]
	}

	return fm, body, true
}

// parseFrontmatterFields extracts known fields from the frontmatter block.
// This is a simple line-based parser that handles our minimal frontmatter
// without pulling in a full YAML library.
func parseFrontmatterFields(raw string) Frontmatter {
	var fm Frontmatter
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		if key, val, ok := parseKV(line); ok {
			switch key {
			case "id":
				fm.ID = val
			}
		}
	}
	return fm
}

func parseKV(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx == -1 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	val := strings.TrimSpace(line[idx+1:])
	return key, val, true
}
