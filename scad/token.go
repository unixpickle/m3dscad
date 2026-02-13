package scad

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

type Token struct {
	Kind   TokenKind
	Lexeme string
	Num    float64
	Pos    Pos
}
