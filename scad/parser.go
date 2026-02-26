package scad

import "fmt"

type Parser struct {
	lx   *Lexer
	cur  Token
	peek Token
}

func NewParser(src string) (*Parser, error) {
	lx := NewLexer(src)
	t0, err := lx.Next()
	if err != nil {
		return nil, err
	}
	t1, err := lx.Next()
	if err != nil {
		return nil, err
	}
	return &Parser{lx: lx, cur: t0, peek: t1}, nil
}

func (p *Parser) ParseProgram() (*Program, error) {
	var stmts []Stmt
	for p.cur.Kind != TokEOF {
		s, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, s)
	}
	return &Program{Stmts: stmts}, nil
}

func (p *Parser) parseStmt() (Stmt, error) {
	// block
	if p.cur.Kind == TokLBrace {
		return p.parseBlock()
	}

	// keywords as identifiers
	if p.cur.Kind == TokIdent {
		switch p.cur.Lexeme {
		case "module":
			return p.parseModuleDef()
		case "function":
			return p.parseFuncDef()
		case "if":
			return p.parseIf()
		}
	}

	// assignment: ident '=' expr ';'
	if p.cur.Kind == TokIdent && p.peek.Kind == TokAssign {
		name := p.cur.Lexeme
		pos := p.cur.Pos
		p.advance() // ident
		p.advance() // =
		ex, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.expect(TokSemi, "expected ';' after assignment"); err != nil {
			return nil, err
		}
		return &AssignStmt{Name: name, Expr: ex, P: pos}, nil
	}

	// call statement (optionally with children)
	if p.cur.Kind == TokIdent && p.peek.Kind == TokLParen {
		call, err := p.parseCall()
		if err != nil {
			return nil, err
		}

		// children block
		if p.cur.Kind == TokLBrace {
			blk, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			return &CallStmt{Call: call, Children: blk.Stmts, P: call.P}, nil
		}

		// semicolon terminator
		if p.cur.Kind == TokSemi {
			p.advance()
			return &CallStmt{Call: call, Children: nil, P: call.P}, nil
		}

		// single-child form: translate(...) cube(...);
		child, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		return &CallStmt{Call: call, Children: []Stmt{child}, P: call.P}, nil
	}

	return nil, fmt.Errorf("%v: expected statement", p.cur.Pos)
}

func (p *Parser) parseBlock() (*BlockStmt, error) {
	pos := p.cur.Pos
	if err := p.expect(TokLBrace, "expected '{'"); err != nil {
		return nil, err
	}
	var stmts []Stmt
	for p.cur.Kind != TokRBrace && p.cur.Kind != TokEOF {
		s, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, s)
	}
	if err := p.expect(TokRBrace, "expected '}'"); err != nil {
		return nil, err
	}
	return &BlockStmt{Stmts: stmts, P: pos}, nil
}

func (p *Parser) parseIf() (Stmt, error) {
	pos := p.cur.Pos
	if err := p.expectIdent("if"); err != nil {
		return nil, err
	}
	if err := p.expect(TokLParen, "expected '(' after if"); err != nil {
		return nil, err
	}
	cond, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if err := p.expect(TokRParen, "expected ')' after if condition"); err != nil {
		return nil, err
	}
	thenStmt, err := p.parseStmt()
	if err != nil {
		return nil, err
	}
	var elseStmt Stmt
	if p.cur.Kind == TokIdent && p.cur.Lexeme == "else" {
		p.advance()
		elseStmt, err = p.parseStmt()
		if err != nil {
			return nil, err
		}
	}
	return &IfStmt{Cond: cond, Then: thenStmt, Else: elseStmt, P: pos}, nil
}

func (p *Parser) parseModuleDef() (Stmt, error) {
	pos := p.cur.Pos
	if err := p.expectIdent("module"); err != nil {
		return nil, err
	}
	if p.cur.Kind != TokIdent {
		return nil, fmt.Errorf("%v: expected module name", p.cur.Pos)
	}
	name := p.cur.Lexeme
	p.advance()

	params, err := p.parseParamList()
	if err != nil {
		return nil, err
	}

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ModuleDefStmt{Name: name, Params: params, Body: body, P: pos}, nil
}

func (p *Parser) parseFuncDef() (Stmt, error) {
	pos := p.cur.Pos
	if err := p.expectIdent("function"); err != nil {
		return nil, err
	}
	if p.cur.Kind != TokIdent {
		return nil, fmt.Errorf("%v: expected function name", p.cur.Pos)
	}
	name := p.cur.Lexeme
	p.advance()

	params, err := p.parseParamList()
	if err != nil {
		return nil, err
	}

	if err := p.expect(TokAssign, "expected '=' in function definition"); err != nil {
		return nil, err
	}
	body, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if err := p.expect(TokSemi, "expected ';' after function definition"); err != nil {
		return nil, err
	}
	return &FuncDefStmt{Name: name, Params: params, Body: body, P: pos}, nil
}

func (p *Parser) parseParamList() ([]Param, error) {
	if err := p.expect(TokLParen, "expected '('"); err != nil {
		return nil, err
	}
	var params []Param
	if p.cur.Kind != TokRParen {
		for {
			if p.cur.Kind != TokIdent {
				return nil, fmt.Errorf("%v: expected parameter name", p.cur.Pos)
			}
			paramPos := p.cur.Pos
			name := p.cur.Lexeme
			p.advance()
			var def Expr
			if p.cur.Kind == TokAssign {
				p.advance()
				ex, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				def = ex
			}
			params = append(params, Param{Name: name, Default: def, P: paramPos})
			if p.cur.Kind == TokComma {
				p.advance()
				continue
			}
			break
		}
	}
	if err := p.expect(TokRParen, "expected ')'"); err != nil {
		return nil, err
	}
	return params, nil
}

func (p *Parser) parseCall() (Call, error) {
	pos := p.cur.Pos
	name := p.cur.Lexeme
	p.advance() // ident
	if err := p.expect(TokLParen, "expected '(' in call"); err != nil {
		return Call{}, err
	}
	var args []Arg
	if p.cur.Kind != TokRParen {
		for {
			argPos := p.cur.Pos
			// name=expr form
			if p.cur.Kind == TokIdent && p.peek.Kind == TokAssign {
				an := p.cur.Lexeme
				p.advance() // ident
				p.advance() // =
				ex, err := p.parseExpr()
				if err != nil {
					return Call{}, err
				}
				args = append(args, Arg{Name: an, Expr: ex, P: argPos})
			} else {
				ex, err := p.parseExpr()
				if err != nil {
					return Call{}, err
				}
				args = append(args, Arg{Name: "", Expr: ex, P: argPos})
			}
			if p.cur.Kind == TokComma {
				p.advance()
				continue
			}
			break
		}
	}
	if err := p.expect(TokRParen, "expected ')' after args"); err != nil {
		return Call{}, err
	}
	return Call{Name: name, Args: args, P: pos}, nil
}

// ---- expressions (precedence climbing) ----

func (p *Parser) parseExpr() (Expr, error) { return p.parseTernary() }

func (p *Parser) parseTernary() (Expr, error) {
	cond, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.cur.Kind == TokQuestion {
		pos := p.cur.Pos
		p.advance()
		t, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.expect(TokColon, "expected ':' in ternary"); err != nil {
			return nil, err
		}
		e, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		return &TernaryExpr{Cond: cond, Then: t, Else: e, P: pos}, nil
	}
	return cond, nil
}

func (p *Parser) parseOr() (Expr, error) {
	x, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.cur.Kind == TokOr {
		op := p.cur.Kind
		pos := p.cur.Pos
		p.advance()
		r, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		x = &BinaryExpr{Op: op, L: x, R: r, P: pos}
	}
	return x, nil
}

func (p *Parser) parseAnd() (Expr, error) {
	x, err := p.parseEq()
	if err != nil {
		return nil, err
	}
	for p.cur.Kind == TokAnd {
		op := p.cur.Kind
		pos := p.cur.Pos
		p.advance()
		r, err := p.parseEq()
		if err != nil {
			return nil, err
		}
		x = &BinaryExpr{Op: op, L: x, R: r, P: pos}
	}
	return x, nil
}

func (p *Parser) parseEq() (Expr, error) {
	x, err := p.parseCmp()
	if err != nil {
		return nil, err
	}
	for p.cur.Kind == TokEq || p.cur.Kind == TokNeq {
		op := p.cur.Kind
		pos := p.cur.Pos
		p.advance()
		r, err := p.parseCmp()
		if err != nil {
			return nil, err
		}
		x = &BinaryExpr{Op: op, L: x, R: r, P: pos}
	}
	return x, nil
}

func (p *Parser) parseCmp() (Expr, error) {
	x, err := p.parseAdd()
	if err != nil {
		return nil, err
	}
	for p.cur.Kind == TokLt || p.cur.Kind == TokLte || p.cur.Kind == TokGt || p.cur.Kind == TokGte {
		op := p.cur.Kind
		pos := p.cur.Pos
		p.advance()
		r, err := p.parseAdd()
		if err != nil {
			return nil, err
		}
		x = &BinaryExpr{Op: op, L: x, R: r, P: pos}
	}
	return x, nil
}

func (p *Parser) parseAdd() (Expr, error) {
	x, err := p.parseMul()
	if err != nil {
		return nil, err
	}
	for p.cur.Kind == TokPlus || p.cur.Kind == TokMinus {
		op := p.cur.Kind
		pos := p.cur.Pos
		p.advance()
		r, err := p.parseMul()
		if err != nil {
			return nil, err
		}
		x = &BinaryExpr{Op: op, L: x, R: r, P: pos}
	}
	return x, nil
}

func (p *Parser) parseMul() (Expr, error) {
	x, err := p.parsePow()
	if err != nil {
		return nil, err
	}
	for p.cur.Kind == TokStar || p.cur.Kind == TokSlash || p.cur.Kind == TokPercent {
		op := p.cur.Kind
		pos := p.cur.Pos
		p.advance()
		r, err := p.parsePow()
		if err != nil {
			return nil, err
		}
		x = &BinaryExpr{Op: op, L: x, R: r, P: pos}
	}
	return x, nil
}

func (p *Parser) parsePow() (Expr, error) {
	x, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	if p.cur.Kind == TokCaret {
		op := p.cur.Kind
		pos := p.cur.Pos
		p.advance()
		r, err := p.parsePow() // right-assoc
		if err != nil {
			return nil, err
		}
		return &BinaryExpr{Op: op, L: x, R: r, P: pos}, nil
	}
	return x, nil
}

func (p *Parser) parseUnary() (Expr, error) {
	if p.cur.Kind == TokNot || p.cur.Kind == TokMinus || p.cur.Kind == TokPlus {
		op := p.cur.Kind
		pos := p.cur.Pos
		p.advance()
		x, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: op, X: x, P: pos}, nil
	}
	return p.parsePrimary()
}

func (p *Parser) parsePrimary() (Expr, error) {
	switch p.cur.Kind {
	case TokNumber:
		e := &NumberLit{V: p.cur.Num, P: p.cur.Pos}
		p.advance()
		return e, nil
	case TokString:
		e := &StringLit{V: p.cur.Lexeme, P: p.cur.Pos}
		p.advance()
		return e, nil
	case TokIdent:
		// true/false
		if p.cur.Lexeme == "true" || p.cur.Lexeme == "false" {
			e := &BoolLit{V: p.cur.Lexeme == "true", P: p.cur.Pos}
			p.advance()
			return e, nil
		}
		// call expr?
		if p.peek.Kind == TokLParen {
			call, err := p.parseCall()
			if err != nil {
				return nil, err
			}
			return &CallExpr{Call: call, P: call.P}, nil
		}
		e := &VarExpr{Name: p.cur.Lexeme, P: p.cur.Pos}
		p.advance()
		return e, nil
	case TokLBrack:
		pos := p.cur.Pos
		p.advance()
		var elems []Expr
		if p.cur.Kind != TokRBrack {
			for {
				ex, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				elems = append(elems, ex)
				if p.cur.Kind == TokComma {
					p.advance()
					if p.cur.Kind == TokRBrack {
						break
					}
					continue
				}
				break
			}
		}
		if err := p.expect(TokRBrack, "expected ']'"); err != nil {
			return nil, err
		}
		return &ArrayLit{Elems: elems, P: pos}, nil
	case TokLParen:
		p.advance()
		ex, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.expect(TokRParen, "expected ')'"); err != nil {
			return nil, err
		}
		return ex, nil
	default:
		return nil, fmt.Errorf("%v: expected expression", p.cur.Pos)
	}
}

func (p *Parser) advance() error {
	t, err := p.lx.Next()
	if err != nil {
		return err
	}
	p.cur, p.peek = p.peek, t
	return nil
}

func (p *Parser) expect(k TokenKind, msg string) error {
	if p.cur.Kind != k {
		return fmt.Errorf("%v: %s", p.cur.Pos, msg)
	}
	return p.advance()
}

func (p *Parser) expectIdent(want string) error {
	if p.cur.Kind != TokIdent || p.cur.Lexeme != want {
		return fmt.Errorf("%v: expected %q", p.cur.Pos, want)
	}
	return p.advance()
}
