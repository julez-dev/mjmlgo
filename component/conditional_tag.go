package component

import (
	"io"
	"strings"
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
