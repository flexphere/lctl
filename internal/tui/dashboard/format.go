package dashboard

import (
	"strings"
)

// TokenKind differentiates the two shapes a parsed format token can
// take: a literal run of bytes, or a variable reference like $name.
type TokenKind int

const (
	// TokLiteral is unchanged text copied straight to the output.
	TokLiteral TokenKind = iota
	// TokVar names a variable whose value is substituted at render time.
	TokVar
)

// Token is one piece of a parsed format string. Exactly one of
// Literal/VarName is populated depending on Kind.
type Token struct {
	Kind    TokenKind
	Literal string
	VarName string
}

// ParseFormat breaks a format string into a flat token list. The
// accepted syntax is intentionally small:
//
//   - $name   substitutes a variable whose identifier matches the regex
//     [a-z_][a-z0-9_]* (case-insensitive).
//   - ${name} same as $name but with explicit delimiters, useful when
//     the identifier is immediately followed by more word characters
//     (e.g. "${label}beta").
//   - \$     a literal '$' character (escape).
//
// Any other run of characters is kept as a literal, including lone
// dollar signs whose following character is not a valid identifier
// start — those are passed through so typos stay visible to the user.
func ParseFormat(s string) []Token {
	var tokens []Token
	var lit strings.Builder
	flush := func() {
		if lit.Len() == 0 {
			return
		}
		tokens = append(tokens, Token{Kind: TokLiteral, Literal: lit.String()})
		lit.Reset()
	}
	i := 0
	for i < len(s) {
		c := s[i]
		// Escape: \$ emits a literal '$' and skips the backslash.
		if c == '\\' && i+1 < len(s) && s[i+1] == '$' {
			lit.WriteByte('$')
			i += 2
			continue
		}
		if c != '$' {
			lit.WriteByte(c)
			i++
			continue
		}
		// Possible variable: ${name} or $name.
		if i+1 < len(s) && s[i+1] == '{' {
			end := strings.IndexByte(s[i+2:], '}')
			if end >= 0 {
				name := s[i+2 : i+2+end]
				if isValidName(name) {
					flush()
					tokens = append(tokens, Token{Kind: TokVar, VarName: name})
					i += 2 + end + 1
					continue
				}
			}
			// Malformed ${...} — treat the $ as a literal so the user
			// can see what they typed.
			lit.WriteByte('$')
			i++
			continue
		}
		name, n := readName(s[i+1:])
		if n == 0 {
			lit.WriteByte('$')
			i++
			continue
		}
		flush()
		tokens = append(tokens, Token{Kind: TokVar, VarName: name})
		i += 1 + n
	}
	flush()
	return tokens
}

// RenderFormat expands a parsed token list with the provided variable
// values. Unknown variables are re-emitted as "$name" (or "${name}"
// for the brace form) so they remain visible instead of silently
// vanishing when the user makes a typo. Since the parser always uses
// the short form for parsing, we re-emit "$name" unconditionally.
func RenderFormat(tokens []Token, vars map[string]string) string {
	var b strings.Builder
	for _, t := range tokens {
		switch t.Kind {
		case TokLiteral:
			b.WriteString(t.Literal)
		case TokVar:
			if v, ok := vars[t.VarName]; ok {
				b.WriteString(v)
			} else {
				b.WriteByte('$')
				b.WriteString(t.VarName)
			}
		}
	}
	return b.String()
}

// readName scans a variable name at the start of s and returns it
// along with the number of bytes consumed. An empty name (return
// value 0) means the caller should treat the leading '$' as literal.
func readName(s string) (string, int) {
	if len(s) == 0 {
		return "", 0
	}
	if !isNameStart(s[0]) {
		return "", 0
	}
	n := 1
	for n < len(s) && isNameCont(s[n]) {
		n++
	}
	return s[:n], n
}

func isValidName(s string) bool {
	if len(s) == 0 || !isNameStart(s[0]) {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isNameCont(s[i]) {
			return false
		}
	}
	return true
}

func isNameStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isNameCont(c byte) bool {
	return isNameStart(c) || (c >= '0' && c <= '9')
}
