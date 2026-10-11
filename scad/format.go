package scad

import "strings"

// Format formats source using two-space indentation. A trailing comma requests
// one item per line. Comments, literal spellings, and explicit parentheses are
// preserved. Invalid source returns an error and no formatted output.
func Format(src string) (string, error) {
	return FormatFile("", src)
}

// FormatFile is Format with a filename for syntax error messages. It does not
// read the file or evaluate the program (including use statements).
func FormatFile(filename, src string) (string, error) {
	if _, err := ParseFile(filename, src); err != nil {
		return "", err
	}
	f := sourceFormatter{src: src, lineStart: true}
	lx := NewLexerFile(filename, src)
	lx.KeepComments = true
	var comments []Token
	var stack []int
	for {
		t, err := lx.Next()
		if err != nil {
			return "", err
		}
		if t.Kind == TokComment {
			comments = append(comments, t)
			continue
		}
		i := len(f.tokens)
		f.tokens = append(f.tokens, formatToken{Token: t, comments: comments})
		comments = nil
		switch t.Kind {
		case TokLParen, TokLBrack, TokLBrace:
			stack = append(stack, i)
		case TokRParen, TokRBrack, TokRBrace:
			open := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			f.tokens[open].close = i
		}
		if t.Kind == TokEOF {
			break
		}
	}
	for f.tokens[f.next].Kind != TokEOF {
		f.stmt(0, false, "\n")
	}
	f.emit(f.next, 0, "\n")
	if f.out.Len() != 0 {
		f.newline(1)
	}
	return f.out.String(), nil
}

// The parser validates grammar; this delimiter tree retains the concrete syntax
// that the evaluation AST intentionally omits. Printing only changes whitespace
// between tokens, never their order or spelling.
type formatToken struct {
	Token
	comments   []Token
	close      int
	rangeColon bool
}

type sourceFormatter struct {
	src       string
	tokens    []formatToken
	next      int
	out       strings.Builder
	lineStart bool
	newlines  int
	last      Pos
}

func (f *sourceFormatter) raw(s string, indent int) {
	if s == "" {
		return
	}
	if f.lineStart {
		f.out.WriteString(strings.Repeat("  ", indent))
	}
	f.out.WriteString(s)
	f.newlines = len(s) - len(strings.TrimRight(s, "\n"))
	f.lineStart = f.newlines > 0
}

func (f *sourceFormatter) newline(n int) {
	if f.out.Len() == 0 {
		return
	}
	for f.newlines < n {
		f.out.WriteByte('\n')
		f.newlines++
	}
	f.lineStart = true
}

func (f *sourceFormatter) separator(s string, line int) {
	if s == "\n" {
		n := 1
		if line > f.last.Line+1 {
			n = 2
		}
		f.newline(n)
	} else if s == " " && !f.lineStart {
		f.out.WriteByte(' ')
	}
}

func (f *sourceFormatter) comments(i, indent int) bool {
	comments := f.tokens[i].comments
	for _, c := range comments {
		sep := "\n"
		if c.Pos.Line == f.last.Line && !f.lineStart {
			sep = " "
		}
		f.separator(sep, c.Pos.Line)
		f.raw(c.Lexeme, indent)
		f.last = c.End
		if strings.HasPrefix(c.Lexeme, "//") {
			f.newline(1)
		}
	}
	f.tokens[i].comments = nil
	return len(comments) != 0
}

func (f *sourceFormatter) emit(i, indent int, before string) {
	if f.comments(i, indent) {
		if f.tokens[i].Pos.Line > f.last.Line {
			before = "\n"
		} else if before == "" {
			before = " "
		}
	}
	t := f.tokens[i]
	// EOF carries final comments, but must not introduce trailing blank lines.
	if t.Kind == TokEOF {
		return
	}
	f.separator(before, t.Pos.Line)
	f.raw(f.src[t.Pos.Offset:t.End.Offset], indent)
	f.last = t.End
}

func (f *sourceFormatter) take(indent int, before string) {
	f.emit(f.next, indent, before)
	f.next++
}

func (f *sourceFormatter) block(indent int, before string) {
	f.take(indent, before)
	empty := f.tokens[f.next].Kind == TokRBrace && len(f.tokens[f.next].comments) == 0
	for f.tokens[f.next].Kind != TokRBrace {
		f.stmt(indent+1, false, "\n")
	}
	f.comments(f.next, indent+1)
	sep := "\n"
	if empty {
		sep = ""
	}
	f.take(indent, sep)
}

func (f *sourceFormatter) stmt(indent int, inChain bool, before string) {
	start := f.next
	t := f.tokens[start]
	if t.Kind == TokLBrace {
		f.block(indent, before)
		return
	}
	switch t.Lexeme {
	case "use":
		f.take(indent, before)
		f.take(indent, " ")
		if f.tokens[f.next].Kind == TokSemi {
			f.take(indent, "")
		}
		return
	case "module", "cache", "function":
		f.take(indent, before)
		f.take(indent, " ")
		f.statementGroup(indent, "", false)
		if t.Lexeme != "function" {
			f.block(indent, " ")
			return
		}
		f.take(indent, " ") // =
		f.throughSemi(indent, " ")
		return
	case "for", "intersection_for", "if":
		f.take(indent, before)
		f.statementGroup(indent, " ", true)
		thenBlock := f.tokens[f.next].Kind == TokLBrace
		if thenBlock {
			f.block(indent, " ")
		} else {
			f.stmt(indent+1, false, "\n")
		}
		if t.Lexeme == "if" && f.tokens[f.next].Lexeme == "else" {
			sep := "\n"
			if thenBlock {
				sep = " "
			}
			f.take(indent, sep)
			switch {
			case f.tokens[f.next].Kind == TokLBrace:
				f.block(indent, " ")
			case f.tokens[f.next].Lexeme == "if":
				f.stmt(indent, false, " ")
			default:
				f.stmt(indent+1, false, "\n")
			}
		}
		return
	}
	f.take(indent, before)
	if f.tokens[f.next].Kind == TokAssign {
		f.take(indent, " ")
		f.throughSemi(indent, " ")
		return
	}
	f.statementGroup(indent, "", false)
	switch f.tokens[f.next].Kind {
	case TokSemi:
		f.take(indent, "")
	case TokLBrace:
		f.block(indent, " ")
	default:
		// Consecutive calls form one modifier chain. A loop or conditional
		// starts a new nesting level, as does each block inside the chain.
		childIndent := indent
		if !inChain {
			childIndent++
		}
		f.stmt(childIndent, true, "\n")
	}
}

func (f *sourceFormatter) statementGroup(indent int, before string, binds bool) {
	end := f.tokens[f.next].close
	f.group(f.next, indent, before, false, binds)
	f.next = end + 1
}

func (f *sourceFormatter) throughSemi(indent int, before string) {
	end := f.next
	for f.tokens[end].Kind != TokSemi {
		end = f.after(end)
	}
	f.expr(f.next, end, indent, before, false, false)
	f.next = end
	f.take(indent, "")
}

func (f *sourceFormatter) after(i int) int {
	if f.tokens[i].close != 0 {
		return f.tokens[i].close + 1
	}
	return i + 1
}

func (f *sourceFormatter) group(open, indent int, before string, compact, binds bool) {
	close := f.tokens[open].close
	f.emit(open, indent, before)
	multiline := close > open+1 && f.tokens[close-1].Kind == TokComma
	for i := open + 1; i <= close; i = f.after(i) {
		for _, c := range f.tokens[i].comments {
			if strings.HasPrefix(c.Lexeme, "//") || strings.Contains(c.Lexeme, "\n") ||
				c.Pos.Line > f.tokens[i-1].End.Line {
				multiline = true
			}
		}
	}
	if open+1 < close && f.tokens[open+1].Lexeme == "for" &&
		f.tokens[open].Pos.Line != f.tokens[close].Pos.Line {
		multiline = true
	}
	// Colons belonging to ternaries aren't range separators.
	rangeExpr, ternaries := false, 0
	for i := open + 1; i < close; i = f.after(i) {
		switch f.tokens[i].Kind {
		case TokQuestion:
			ternaries++
		case TokColon:
			if ternaries == 0 {
				rangeExpr = true
				f.tokens[i].rangeColon = true
			} else {
				ternaries--
			}
		}
	}
	if rangeExpr {
		compact = true
	}
	if multiline {
		start := open + 1
		for i := start; i < close; i = f.after(i) {
			if f.tokens[i].Kind == TokComma {
				f.expr(start, i, indent+1, "\n", compact, binds)
				f.emit(i, indent+1, "")
				start = i + 1
			}
		}
		f.expr(start, close, indent+1, "\n", compact, binds)
		f.comments(close, indent+1)
		f.emit(close, indent, "\n")
	} else {
		f.expr(open+1, close, indent, "", compact, binds)
		f.emit(close, indent, "")
	}
}

func arithmeticPrecedence(k TokenKind) int {
	switch k {
	case TokPlus, TokMinus:
		return 1
	case TokStar, TokSlash, TokPercent:
		return 2
	case TokCaret:
		return 3
	}
	return 0
}

func endsOperand(k TokenKind) bool {
	return k == TokIdent || k == TokNumber || k == TokString || k == TokRParen || k == TokRBrack
}

// expr spaces a validated expression without reconstructing it from the AST:
// this retains redundant parentheses and avoids changing operator associativity.
func (f *sourceFormatter) expr(start, end, indent int, before string, compact, binds bool) {
	if start >= end {
		return
	}
	// Each comma-separated expression chooses its own arithmetic spacing.
	for i := start; i < end; i = f.after(i) {
		if f.tokens[i].Kind == TokComma {
			f.expr(start, i, indent, before, compact, binds)
			f.emit(i, indent, "")
			f.expr(i+1, end, indent, " ", compact, binds)
			return
		}
	}
	lowest := 4
	prev := TokEOF
	for i := start; i < end; i = f.after(i) {
		t := f.tokens[i]
		if p := arithmeticPrecedence(t.Kind); p != 0 && endsOperand(prev) && p < lowest {
			lowest = p
		}
		prev = f.tokens[f.after(i)-1].Kind
		if t.Lexeme == "each" {
			prev = TokEOF
		}
	}
	sep := before
	prev = TokEOF
	for i := start; i < end; i = f.after(i) {
		t := f.tokens[i]
		if t.close != 0 {
			isIndex := t.Kind == TokLBrack && endsOperand(prev)
			if i > start && f.tokens[i-1].Lexeme == "each" {
				isIndex = false
				sep = " "
			}
			groupBinds := i > start && (f.tokens[i-1].Lexeme == "for" || f.tokens[i-1].Lexeme == "let")
			functionBody := i > start && f.tokens[i-1].Lexeme == "function"
			if t.Kind == TokLParen && endsOperand(prev) {
				sep = ""
				if groupBinds || functionBody || f.tokens[i-1].Lexeme == "each" {
					sep = " "
				}
			} else if isIndex {
				sep = ""
			}
			f.group(i, indent, sep, isIndex, groupBinds)
			prev = f.tokens[t.close].Kind
			sep = ""
			if (groupBinds || functionBody) && t.close+1 < end {
				bodySep, bodyIndent := " ", indent
				if f.tokens[t.close+1].Pos.Line > f.tokens[t.close].End.Line {
					bodySep, bodyIndent = "\n", indent+1
				}
				f.expr(t.close+1, end, bodyIndent, bodySep, compact, binds)
				return
			}
			continue
		}
		after := ""
		switch t.Kind {
		case TokAssign:
			sep = ""
			if binds {
				sep = " "
			}
			after = sep
		case TokDot:
			sep = ""
			if prev == TokNumber {
				sep = " " // 1.x would lex as the number 1. followed by x.
			}
		case TokNot:
		case TokPlus, TokMinus, TokStar, TokSlash, TokPercent, TokCaret:
			if endsOperand(prev) && (i == start || f.tokens[i-1].Lexeme != "each") {
				sep = ""
				if !compact && arithmeticPrecedence(t.Kind) == lowest {
					sep = " "
				}
				after = sep
			} else if i > start && t.Kind == prev {
				// Keep adjacent unary + or - distinct, like gofmt.
				sep = " "
			}
		case TokColon:
			sep = " "
			if t.rangeColon && lowest == 4 {
				sep = ""
			}
			after = sep
		case TokQuestion, TokEq, TokNeq, TokLt, TokLte, TokGt, TokGte, TokAnd, TokOr:
			sep, after = " ", " "
		default:
			if endsOperand(prev) {
				sep = " "
			}
			if t.Lexeme == "each" {
				after = " "
			}
		}
		f.emit(i, indent, sep)
		sep, prev = after, t.Kind
	}
}
