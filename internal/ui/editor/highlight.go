// Package editor provides the SQL editor component for gotermsql.
package editor

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/lipgloss"
	"github.com/sadopc/gotermsql/internal/theme"
)

// Highlighter tokenises SQL text using chroma and renders it with lipgloss
// styles from the active theme.
type Highlighter struct {
	lexer chroma.Lexer
}

// NewHighlighter creates a Highlighter that uses the PostgreSQL lexer. If the
// PostgreSQL lexer is unavailable it falls back to the generic SQL lexer.
func NewHighlighter() *Highlighter {
	l := lexers.Get("PostgreSQL")
	if l == nil {
		l = lexers.Get("SQL")
	}
	if l == nil {
		l = lexers.Fallback
	}
	// Coalesce runs of identical token types so the loop below processes
	// fewer, larger chunks.
	l = chroma.Coalesce(l)

	return &Highlighter{lexer: l}
}

// Highlight tokenises sql and returns a string where each token is styled
// with the corresponding lipgloss style from the provided theme. Newlines are
// preserved so multi-line SQL renders correctly.
func (h *Highlighter) Highlight(sql string, th *theme.Theme) string {
	if th == nil {
		return sql
	}

	iter, err := h.lexer.Tokenise(nil, sql)
	if err != nil {
		return sql
	}

	var b strings.Builder
	b.Grow(len(sql) * 2) // rough estimate

	for _, tok := range iter.Tokens() {
		value := tok.Value
		if value == "" {
			continue
		}

		style, ok := styleFor(tok.Type, th)
		if !ok {
			b.WriteString(value)
			continue
		}

		// Handle tokens that contain newlines: style each segment
		// individually so that a newline is always emitted as-is.
		if strings.Contains(value, "\n") {
			lines := strings.Split(value, "\n")
			for i, line := range lines {
				if line != "" {
					b.WriteString(style.Render(line))
				}
				if i < len(lines)-1 {
					b.WriteByte('\n')
				}
			}
		} else {
			b.WriteString(style.Render(value))
		}
	}

	return b.String()
}

// styleFor maps a chroma token type to the corresponding lipgloss.Style from
// the theme. The second return value is false when the token should pass
// through unstyled.
func styleFor(tt chroma.TokenType, th *theme.Theme) (lipgloss.Style, bool) {
	switch {
	// KeywordType is a subtype of Keyword, so check it first to give SQL
	// types (e.g. INT, VARCHAR) their own colour.
	case tt == chroma.KeywordType:
		return th.SQLType, true
	case tt == chroma.NameFunction:
		return th.SQLFunction, true
	case isKeyword(tt):
		return th.SQLKeyword, true
	case isString(tt):
		return th.SQLString, true
	case isNumber(tt):
		return th.SQLNumber, true
	case isComment(tt):
		return th.SQLComment, true
	case tt == chroma.Operator || tt == chroma.OperatorWord:
		return th.SQLOperator, true
	default:
		return lipgloss.Style{}, false
	}
}

// ---------------------------------------------------------------------------
// Token type helpers
// ---------------------------------------------------------------------------

func isKeyword(tt chroma.TokenType) bool {
	return tt == chroma.Keyword ||
		tt == chroma.KeywordConstant ||
		tt == chroma.KeywordDeclaration ||
		tt == chroma.KeywordNamespace ||
		tt == chroma.KeywordPseudo ||
		tt == chroma.KeywordReserved ||
		tt == chroma.KeywordType
}

func isString(tt chroma.TokenType) bool {
	return tt == chroma.LiteralString ||
		tt == chroma.LiteralStringAffix ||
		tt == chroma.LiteralStringBacktick ||
		tt == chroma.LiteralStringChar ||
		tt == chroma.LiteralStringDelimiter ||
		tt == chroma.LiteralStringDoc ||
		tt == chroma.LiteralStringDouble ||
		tt == chroma.LiteralStringEscape ||
		tt == chroma.LiteralStringHeredoc ||
		tt == chroma.LiteralStringInterpol ||
		tt == chroma.LiteralStringOther ||
		tt == chroma.LiteralStringRegex ||
		tt == chroma.LiteralStringSingle ||
		tt == chroma.LiteralStringSymbol
}

func isNumber(tt chroma.TokenType) bool {
	return tt == chroma.LiteralNumber ||
		tt == chroma.LiteralNumberBin ||
		tt == chroma.LiteralNumberFloat ||
		tt == chroma.LiteralNumberHex ||
		tt == chroma.LiteralNumberInteger ||
		tt == chroma.LiteralNumberIntegerLong ||
		tt == chroma.LiteralNumberOct
}

func isComment(tt chroma.TokenType) bool {
	return tt == chroma.Comment ||
		tt == chroma.CommentHashbang ||
		tt == chroma.CommentMultiline ||
		tt == chroma.CommentPreproc ||
		tt == chroma.CommentPreprocFile ||
		tt == chroma.CommentSingle ||
		tt == chroma.CommentSpecial
}
