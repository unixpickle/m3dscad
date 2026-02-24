package scad

import "fmt"

type TokenKind int

const (
	TokEOF TokenKind = iota
	TokIdent
	TokNumber
	TokString

	// Punctuation
	TokLParen
	TokRParen
	TokLBrace
	TokRBrace
	TokLBrack
	TokRBrack
	TokComma
	TokSemi

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
	Offset int
	Line   int
	Col    int
}

func (p Pos) String() string {
	if p.Line == 0 && p.Col == 0 {
		return fmt.Sprintf("offset %d", p.Offset)
	}
	return fmt.Sprintf("%d:%d", p.Line, p.Col)
}

type Token struct {
	Kind   TokenKind
	Lexeme string
	Num    float64
	Pos    Pos
}
