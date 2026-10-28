package component

import (
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

const startConditionalTag = "<!--[if mso | IE]>"
const startMsoConditionalTag = "<!--[if mso]>"
const endConditionalTag = "<![endif]-->"
const startNegationConditionalTag = "<!--[if !mso | IE]><!-->"
const startMsoNegationConditionalTag = "<!--[if !mso]><!-->"
const endNegationConditionalTag = "<!--<![endif]-->"

func conditionalTag(content string, isNegation bool) string {
	bd := strings.Builder{}
	if isNegation {
		_, _ = io.WriteString(&bd, startNegationConditionalTag)
	} else {
		_, _ = io.WriteString(&bd, startConditionalTag)
	}

	_, _ = io.WriteString(&bd, content)

	if isNegation {
		_, _ = io.WriteString(&bd, endNegationConditionalTag)
	} else {
		_, _ = io.WriteString(&bd, endConditionalTag)
	}

	return bd.String()
}

func msoConditionalTag(content string, isNegation bool) string {
	bd := strings.Builder{}
	if isNegation {
		_, _ = io.WriteString(&bd, startMsoNegationConditionalTag)
	} else {
		_, _ = io.WriteString(&bd, startMsoConditionalTag)
	}

	_, _ = io.WriteString(&bd, content)

	if isNegation {
		_, _ = io.WriteString(&bd, endNegationConditionalTag)
	} else {
		_, _ = io.WriteString(&bd, endConditionalTag)
	}

	return bd.String()
}

// MergeOutlookConditionals joins adjacent Outlook conditional blocks exactly
// like MJML's mergeOutlookConditionnals helper: the closing marker of a block,
// the whitespace after it and the opening marker of the next "mso | IE" block
// are dropped, so contiguous Outlook fragments form one conditional comment.
//
// It is implemented as a single scan instead of a regular expression over the
// whole document, which keeps it allocation-free apart from the result buffer.
func MergeOutlookConditionals(content string) string {
	first := strings.Index(content, endConditionalTag)
	if first < 0 {
		return content
	}

	var b strings.Builder
	b.Grow(len(content))

	rest := content

	for {
		i := strings.Index(rest, endConditionalTag)
		if i < 0 {
			b.WriteString(rest)
			return b.String()
		}

		afterEnd := i + len(endConditionalTag)

		j := afterEnd
		for j < len(rest) {
			r, size := utf8.DecodeRuneInString(rest[j:])
			if !unicode.IsSpace(r) {
				break
			}

			j += size
		}

		if strings.HasPrefix(rest[j:], startConditionalTag) {
			b.WriteString(rest[:i])
			rest = rest[j+len(startConditionalTag):]

			continue
		}

		b.WriteString(rest[:afterEnd])
		rest = rest[afterEnd:]
	}
}
