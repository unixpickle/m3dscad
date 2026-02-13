package scad

import (
	"fmt"
	"strconv"
	"unicode"
)

type Lexer struct {
	s   string
	i   int
	pos Pos
}

func NewLexer(s string) *Lexer {
	return &Lexer{s: s, pos: Pos{Line: 1, Col: 1}}
}

func (l *Lexer) Next() (Token, error) {
	l.skipSpaceAndComments()

	if l.i >= len(l.s) {
		return Token{Kind: TokEOF, Pos: l.pos}, nil
	}

	startPos := l.pos
	ch := l.peek()

	// Ident / keyword
	if isIdentStart(ch) {
		start := l.i
		l.advance()
		for l.i < len(l.s) && isIdentContinue(l.peek()) {
			l.advance()
		}
		lex := l.s[start:l.i]
		return Token{Kind: TokIdent, Lexeme: lex, Pos: startPos}, nil
	}

	// Number (simple float)
	if unicode.IsDigit(rune(ch)) || (ch == '.' && l.i+1 < len(l.s) && unicode.IsDigit(rune(l.s[l.i+1]))) {
		start := l.i
		l.advance()
		for l.i < len(l.s) {
			c := l.peek()
			if unicode.IsDigit(rune(c)) || c == '.' || c == 'e' || c == 'E' || c == '+' || c == '-' {
				// permissive; ParseFloat will validate
				l.advance()
				continue
			}
			break
		}
		txt := l.s[start:l.i]
		f, err := strconv.ParseFloat(txt, 64)
		if err != nil {
			return Token{}, fmt.Errorf("%v: invalid number %q: %w", startPos, txt, err)
		}
		return Token{Kind: TokNumber, Lexeme: txt, Num: f, Pos: startPos}, nil
	}

	// String
	if ch == '"' {
		l.advance() // opening quote
		start := l.i
		for l.i < len(l.s) && l.peek() != '"' {
			if l.peek() == '\\' && l.i+1 < len(l.s) {
				l.advance()
			}
			l.advance()
		}
		if l.i >= len(l.s) {
			return Token{}, fmt.Errorf("%v: unterminated string", startPos)
		}
		txt := l.s[start:l.i]
		l.advance() // closing quote
		return Token{Kind: TokString, Lexeme: txt, Pos: startPos}, nil
	}

	// Two-char ops
	if l.match("==") {
		return Token{Kind: TokEq, Lexeme: "==", Pos: startPos}, nil
	}
	if l.match("!=") {
		return Token{Kind: TokNeq, Lexeme: "!=", Pos: startPos}, nil
	}
	if l.match("<=") {
		return Token{Kind: TokLte, Lexeme: "<=", Pos: startPos}, nil
	}
	if l.match(">=") {
		return Token{Kind: TokGte, Lexeme: ">=", Pos: startPos}, nil
	}
	if l.match("&&") {
		return Token{Kind: TokAnd, Lexeme: "&&", Pos: startPos}, nil
	}
	if l.match("||") {
		return Token{Kind: TokOr, Lexeme: "||", Pos: startPos}, nil
	}

	// Single-char tokens
	l.advance()
	switch ch {
	case '(':
		return Token{Kind: TokLParen, Lexeme: "(", Pos: startPos}, nil
	case ')':
		return Token{Kind: TokRParen, Lexeme: ")", Pos: startPos}, nil
	case '{':
		return Token{Kind: TokLBrace, Lexeme: "{", Pos: startPos}, nil
	case '}':
		return Token{Kind: TokRBrace, Lexeme: "}", Pos: startPos}, nil
	case '[':
		return Token{Kind: TokLBrack, Lexeme: "[", Pos: startPos}, nil
	case ']':
		return Token{Kind: TokRBrack, Lexeme: "]", Pos: startPos}, nil
	case ',':
		return Token{Kind: TokComma, Lexeme: ",", Pos: startPos}, nil
	case ';':
		return Token{Kind: TokSemi, Lexeme: ";", Pos: startPos}, nil
	case '=':
		return Token{Kind: TokAssign, Lexeme: "=", Pos: startPos}, nil
	case '+':
		return Token{Kind: TokPlus, Lexeme: "+", Pos: startPos}, nil
	case '-':
		return Token{Kind: TokMinus, Lexeme: "-", Pos: startPos}, nil
	case '*':
		return Token{Kind: TokStar, Lexeme: "*", Pos: startPos}, nil
	case '/':
		return Token{Kind: TokSlash, Lexeme: "/", Pos: startPos}, nil
	case '%':
		return Token{Kind: TokPercent, Lexeme: "%", Pos: startPos}, nil
	case '^':
		return Token{Kind: TokCaret, Lexeme: "^", Pos: startPos}, nil
	case '!':
		return Token{Kind: TokNot, Lexeme: "!", Pos: startPos}, nil
	case '<':
		return Token{Kind: TokLt, Lexeme: "<", Pos: startPos}, nil
	case '>':
		return Token{Kind: TokGt, Lexeme: ">", Pos: startPos}, nil
	case '?':
		return Token{Kind: TokQuestion, Lexeme: "?", Pos: startPos}, nil
	case ':':
		return Token{Kind: TokColon, Lexeme: ":", Pos: startPos}, nil
	default:
		return Token{}, fmt.Errorf("%v: unexpected character %q", startPos, ch)
	}
}

func (l *Lexer) skipSpaceAndComments() {
	for {
		// whitespace
		for l.i < len(l.s) {
			c := l.peek()
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
				l.advance()
			} else {
				break
			}
		}
		// line comment //
		if l.i+1 < len(l.s) && l.s[l.i] == '/' && l.s[l.i+1] == '/' {
			for l.i < len(l.s) && l.peek() != '\n' {
				l.advance()
			}
			continue
		}
		// block comment /* */
		if l.i+1 < len(l.s) && l.s[l.i] == '/' && l.s[l.i+1] == '*' {
			l.advance()
			l.advance()
			for l.i+1 < len(l.s) && !(l.s[l.i] == '*' && l.s[l.i+1] == '/') {
				l.advance()
			}
			if l.i+1 < len(l.s) {
				l.advance()
				l.advance()
			}
			continue
		}
		return
	}
}

func (l *Lexer) peek() byte { return l.s[l.i] }

func (l *Lexer) advance() {
	if l.i >= len(l.s) {
		return
	}
	ch := l.s[l.i]
	l.i++
	l.pos.Offset++
	if ch == '\n' {
		l.pos.Line++
		l.pos.Col = 1
	} else {
		l.pos.Col++
	}
}

func (l *Lexer) match(s string) bool {
	if l.i+len(s) > len(l.s) {
		return false
	}
	if l.s[l.i:l.i+len(s)] != s {
		return false
	}
	for range s {
		l.advance()
	}
	return true
}

func isIdentStart(b byte) bool {
	return b == '_' || unicode.IsLetter(rune(b))
}
func isIdentContinue(b byte) bool {
	return isIdentStart(b) || unicode.IsDigit(rune(b))
}
