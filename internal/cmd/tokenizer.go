package cmd

import (
	"fmt"
	"strings"
)

// Redirect holds redirect info from the command line.
type Redirect struct {
	Append bool   // >> vs >
	Path   string // target path
}

// Tokenize splits a command line into tokens, handling quotes and redirects.
// Returns the tokens, optional redirect info, and any error.
func Tokenize(line string) ([]string, *Redirect, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil, nil
	}

	var tokens []string
	var current strings.Builder
	var redirect *Redirect
	inSingle := false
	inDouble := false
	escaped := false

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if escaped {
			current.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' && !inSingle {
			escaped = true
			continue
		}

		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}

		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}

		if inSingle || inDouble {
			current.WriteByte(ch)
			continue
		}

		// Check for redirect operators
		if ch == '>' {
			// Save current token if any
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}

			append_ := false
			if i+1 < len(line) && line[i+1] == '>' {
				append_ = true
				i++
			}

			// Skip whitespace after redirect
			i++
			for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
				i++
			}

			// Read the path
			var pathBuilder strings.Builder
			pathInSingle := false
			pathInDouble := false
			for i < len(line) {
				pch := line[i]
				if pch == '\'' && !pathInDouble {
					pathInSingle = !pathInSingle
					i++
					continue
				}
				if pch == '"' && !pathInSingle {
					pathInDouble = !pathInDouble
					i++
					continue
				}
				if !pathInSingle && !pathInDouble && (pch == ' ' || pch == '\t') {
					break
				}
				pathBuilder.WriteByte(pch)
				i++
			}

			if pathBuilder.Len() == 0 {
				return nil, nil, fmt.Errorf("syntax error: redirect without target")
			}

			redirect = &Redirect{
				Append: append_,
				Path:   pathBuilder.String(),
			}
			continue
		}

		if ch == ' ' || ch == '\t' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteByte(ch)
	}

	if inSingle || inDouble {
		return nil, nil, fmt.Errorf("syntax error: unterminated quote")
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens, redirect, nil
}
