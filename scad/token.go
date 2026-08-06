package scad

import "fmt"

type TokenKind int

const (
	TokEOF TokenKind = iota
	TokIdent
	TokNumber
	TokString
	TokUsePath

	// Punctuation
	TokLParen
	TokRParen
	TokLBrace
	TokRBrace
	TokLBrack
	TokRBrack
	TokComma
	TokSemi
	TokDot

	// Operators
	TokAssign // =
	TokPlus   // +
	TokMinus  // -
	TokStar   // *
	TokSlash  // /
	TokPercent
	TokCaret // ^

	TokNot // !
	TokEq  // ==
	TokNeq // !=
	TokLt  // <
	TokLte // <=
	TokGt  // >
	TokGte // >=
	TokAnd // &&
	TokOr  // ||

	TokQuestion // ?
	TokColon    // :
)

type Pos struct {
	Filename string
	Offset   int
	Line     int
	Col      int
}

func (p Pos) String() string {
	prefix := ""
	if p.Filename != "" {
		prefix = p.Filename + ":"
	}
	if p.Line == 0 && p.Col == 0 {
		return fmt.Sprintf("%soffset %d", prefix, p.Offset)
	}
	return fmt.Sprintf("%s%d:%d", prefix, p.Line, p.Col)
}

type Token struct {
	Kind   TokenKind
	Lexeme string
	Num    float64
	Pos    Pos
}
