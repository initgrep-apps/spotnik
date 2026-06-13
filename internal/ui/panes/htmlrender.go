package panes

import (
	"html"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/glamour"
)

func renderMarkdown(md string, width int) (string, error) {
	if md == "" {
		return "", nil
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	out, err := r.Render(md)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func htmlToMarkdown(htmlText string) string {
	if htmlText == "" {
		return ""
	}

	var out strings.Builder
	inTag := false
	var tagName strings.Builder
	hasContent := false

	for i := 0; i < len(htmlText); {
		if htmlText[i] == '<' {
			inTag = true
			tagName.Reset()
			i++
			continue
		}

		if inTag {
			if htmlText[i] == '>' {
				inTag = false
				tag := strings.ToLower(strings.TrimSpace(tagName.String()))

				if strings.HasPrefix(tag, "/") {
					closing := strings.TrimPrefix(tag, "/")
					switch closing {
					case "p":
						out.WriteString("\n\n")
						hasContent = true
					case "li":
						out.WriteString("\n")
					case "a":
						out.WriteString("]")
					case "b", "strong":
						out.WriteString("**")
					case "i", "em":
						out.WriteString("*")
					}
					i++
					continue
				}

				base := tag
				if idx := strings.IndexAny(base, " \t\n\r"); idx >= 0 {
					base = base[:idx]
				}

				switch base {
				case "br":
					out.WriteString("  \n")
					hasContent = true
				case "p":
					if hasContent {
						out.WriteString("\n\n")
					}
				case "li":
					out.WriteString("\n- ")
				case "b", "strong":
					out.WriteString("**")
				case "i", "em":
					out.WriteString("*")
				case "a":
					out.WriteString("[")
				}
				i++
				continue
			}

			if htmlText[i] == '/' && i+1 < len(htmlText) && htmlText[i+1] == '>' {
				base := strings.ToLower(strings.TrimSpace(tagName.String()))
				if base == "br" {
					out.WriteString("  \n")
					hasContent = true
				}
				inTag = false
				i += 2
				continue
			}

			tagName.WriteByte(htmlText[i])
			i++
			continue
		}

		if htmlText[i] == '&' {
			var entity strings.Builder
			entity.WriteByte('&')
			j := i + 1
			for j < len(htmlText) && htmlText[j] != ';' && j-i < 20 {
				entity.WriteByte(htmlText[j])
				j++
			}
			if j < len(htmlText) && htmlText[j] == ';' {
				entity.WriteByte(';')
				decoded := html.UnescapeString(entity.String())
				out.WriteString(decoded)
				i = j + 1
				continue
			}
		}

		r, size := utf8.DecodeRuneInString(htmlText[i:])
		if r != utf8.RuneError {
			if unicode.IsControl(r) && r != '\n' && r != '\t' {
				i += size
				continue
			}
			out.WriteRune(r)
			hasContent = true
		} else {
			out.WriteByte(htmlText[i])
		}
		i += size
	}

	result := out.String()
	// Collapse runs of 3+ newlines into double-newline paragraph breaks.
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(result)
}
