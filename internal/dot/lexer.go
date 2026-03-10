package dot

import (
	"fmt"
	"strings"
	"unicode"
)

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokDigraph
	tokGraph
	tokNode
	tokEdge
	tokSubgraph
	tokIdent
	tokString
	tokNumber
	tokArrow  // ->
	tokEquals // =
	tokComma
	tokSemicolon
	tokLBrace
	tokRBrace
	tokLBrack
	tokRBrack
)

type token struct {
	kind tokenKind
	val  string
	line int
	col  int
}

func (t token) String() string {
	return fmt.Sprintf("<%d:%q line=%d>", t.kind, t.val, t.line)
}

type lexer struct {
	input  []rune
	pos    int
	line   int
	col    int
	tokens []token
}

func lex(input string) ([]token, error) {
	cleaned := stripComments(input)
	l := &lexer{input: []rune(cleaned), line: 1, col: 1}
	if err := l.run(); err != nil {
		return nil, err
	}
	return l.tokens, nil
}

func stripComments(s string) string {
	var b strings.Builder
	runes := []rune(s)
	i := 0
	inString := false

	for i < len(runes) {
		if inString {
			if runes[i] == '\\' && i+1 < len(runes) {
				b.WriteRune(runes[i])
				b.WriteRune(runes[i+1])
				i += 2
				continue
			}
			if runes[i] == '"' {
				inString = false
			}
			b.WriteRune(runes[i])
			i++
			continue
		}

		if runes[i] == '"' {
			inString = true
			b.WriteRune(runes[i])
			i++
			continue
		}

		if i+1 < len(runes) && runes[i] == '/' && runes[i+1] == '/' {
			for i < len(runes) && runes[i] != '\n' {
				i++
			}
			continue
		}

		if i+1 < len(runes) && runes[i] == '/' && runes[i+1] == '*' {
			i += 2
			for i+1 < len(runes) && !(runes[i] == '*' && runes[i+1] == '/') {
				i++
			}
			if i+1 < len(runes) {
				i += 2
			}
			b.WriteRune(' ')
			continue
		}

		b.WriteRune(runes[i])
		i++
	}
	return b.String()
}

func (l *lexer) run() error {
	for {
		l.skipWhitespace()
		if l.pos >= len(l.input) {
			l.emit(tokEOF, "")
			return nil
		}

		ch := l.input[l.pos]

		switch {
		case ch == '{':
			l.emit(tokLBrace, "{")
			l.advance()
		case ch == '}':
			l.emit(tokRBrace, "}")
			l.advance()
		case ch == '[':
			l.emit(tokLBrack, "[")
			l.advance()
		case ch == ']':
			l.emit(tokRBrack, "]")
			l.advance()
		case ch == '=':
			l.emit(tokEquals, "=")
			l.advance()
		case ch == ',':
			l.emit(tokComma, ",")
			l.advance()
		case ch == ';':
			l.emit(tokSemicolon, ";")
			l.advance()
		case ch == '-' && l.peek() == '>':
			l.emit(tokArrow, "->")
			l.advance()
			l.advance()
		case ch == '"':
			s, err := l.readString()
			if err != nil {
				return err
			}
			l.emit(tokString, s)
		case ch == '-' || unicode.IsDigit(ch):
			n := l.readNumber()
			if n == "-" {
				return fmt.Errorf("line %d: unexpected '-'", l.line)
			}
			l.emit(tokNumber, n)
		case isIdentStart(ch):
			id := l.readIdent()
			switch id {
			case "digraph":
				l.emit(tokDigraph, id)
			case "graph":
				l.emit(tokGraph, id)
			case "node":
				l.emit(tokNode, id)
			case "edge":
				l.emit(tokEdge, id)
			case "subgraph":
				l.emit(tokSubgraph, id)
			default:
				l.emit(tokIdent, id)
			}
		default:
			return fmt.Errorf("line %d col %d: unexpected character %q", l.line, l.col, ch)
		}
	}
}

func (l *lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == '\n' {
			l.line++
			l.col = 0
		}
		if !unicode.IsSpace(ch) {
			break
		}
		l.pos++
		l.col++
	}
}

func (l *lexer) advance() {
	l.pos++
	l.col++
}

func (l *lexer) peek() rune {
	if l.pos+1 < len(l.input) {
		return l.input[l.pos+1]
	}
	return 0
}

func (l *lexer) emit(kind tokenKind, val string) {
	l.tokens = append(l.tokens, token{kind: kind, val: val, line: l.line, col: l.col})
}

func (l *lexer) readString() (string, error) {
	l.advance() // skip opening quote
	var b strings.Builder
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == '\\' && l.pos+1 < len(l.input) {
			next := l.input[l.pos+1]
			switch next {
			case '"':
				b.WriteRune('"')
			case 'n':
				b.WriteRune('\n')
			case 't':
				b.WriteRune('\t')
			case '\\':
				b.WriteRune('\\')
			default:
				b.WriteRune('\\')
				b.WriteRune(next)
			}
			l.pos += 2
			l.col += 2
			continue
		}
		if ch == '"' {
			l.advance()
			return b.String(), nil
		}
		if ch == '\n' {
			l.line++
			l.col = 0
		}
		b.WriteRune(ch)
		l.advance()
	}
	return "", fmt.Errorf("line %d: unterminated string", l.line)
}

func (l *lexer) readNumber() string {
	start := l.pos
	if l.input[l.pos] == '-' {
		l.advance()
	}
	for l.pos < len(l.input) && (unicode.IsDigit(l.input[l.pos]) || l.input[l.pos] == '.') {
		l.advance()
	}
	// Duration suffix
	if l.pos < len(l.input) {
		rest := string(l.input[l.pos:])
		for _, suffix := range []string{"ms", "s", "m", "h", "d"} {
			if strings.HasPrefix(rest, suffix) {
				for range suffix {
					l.advance()
				}
				break
			}
		}
	}
	return string(l.input[start:l.pos])
}

func (l *lexer) readIdent() string {
	start := l.pos
	for l.pos < len(l.input) && isIdentPart(l.input[l.pos]) {
		l.advance()
	}
	return string(l.input[start:l.pos])
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentPart(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
