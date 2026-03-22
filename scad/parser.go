package scad

type Parser struct {
	lx   *Lexer
	cur  Token
	peek Token
}

type namedExprBind struct {
	Name string
	Expr Expr
	P    Pos
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
		case "for":
			return p.parseForStmt(false)
		case "intersection_for":
			return p.parseForStmt(true)
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

	return nil, PosErrorf(p.cur.Pos, "expected statement")
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

func (p *Parser) parseForStmt(intersection bool) (Stmt, error) {
	pos := p.cur.Pos
	if intersection {
		if err := p.expectIdent("intersection_for"); err != nil {
			return nil, err
		}
	} else {
		if err := p.expectIdent("for"); err != nil {
			return nil, err
		}
	}
	binds, err := p.parseForBinds()
	if err != nil {
		return nil, err
	}
	body, err := p.parseStmt()
	if err != nil {
		return nil, err
	}
	return &ForStmt{Binds: binds, Body: body, Intersection: intersection, P: pos}, nil
}

func (p *Parser) parseModuleDef() (Stmt, error) {
	pos := p.cur.Pos
	if err := p.expectIdent("module"); err != nil {
		return nil, err
	}
	if p.cur.Kind != TokIdent {
		return nil, PosErrorf(p.cur.Pos, "expected module name")
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
		return nil, PosErrorf(p.cur.Pos, "expected function name")
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
				return nil, PosErrorf(p.cur.Pos, "expected parameter name")
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
	args, err := p.parseArgList()
	if err != nil {
		return Call{}, err
	}
	return Call{Name: name, Args: args, P: pos}, nil
}

func (p *Parser) parseArgList() ([]Arg, error) {
	if err := p.expect(TokLParen, "expected '(' in call"); err != nil {
		return nil, err
	}
	var args []Arg
	if p.cur.Kind != TokRParen {
		for {
			argPos := p.cur.Pos
			if p.cur.Kind == TokIdent && p.peek.Kind == TokAssign {
				an := p.cur.Lexeme
				p.advance()
				p.advance()
				ex, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				args = append(args, Arg{Name: an, Expr: ex, P: argPos})
			} else {
				ex, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				args = append(args, Arg{Expr: ex, P: argPos})
			}
			if p.cur.Kind == TokComma {
				p.advance()
				continue
			}
			break
		}
	}
	if err := p.expect(TokRParen, "expected ')' after args"); err != nil {
		return nil, err
	}
	return args, nil
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
	expr, err := p.parseAtom()
	if err != nil {
		return nil, err
	}
	for {
		if p.cur.Kind == TokLBrack {
			pos := p.cur.Pos
			p.advance()
			idx, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if err := p.expect(TokRBrack, "expected ']' after index"); err != nil {
				return nil, err
			}
			expr = &IndexExpr{X: expr, Index: idx, P: pos}
			continue
		}
		if p.cur.Kind == TokDot {
			pos := p.cur.Pos
			p.advance()
			if p.cur.Kind != TokIdent {
				return nil, PosErrorf(p.cur.Pos, "expected identifier after '.'")
			}
			expr = &DotExpr{X: expr, Name: p.cur.Lexeme, P: pos}
			p.advance()
			continue
		}
		if p.cur.Kind == TokLParen {
			pos := p.cur.Pos
			args, err := p.parseArgList()
			if err != nil {
				return nil, err
			}
			if v, ok := expr.(*VarExpr); ok {
				expr = &CallExpr{
					Call: Call{Name: v.Name, Args: args, P: pos},
					P:    pos,
				}
			} else {
				expr = &InvokeExpr{Fn: expr, Args: args, P: pos}
			}
			continue
		}
		break
	}
	return expr, nil
}

func (p *Parser) parseAtom() (Expr, error) {
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
		switch p.cur.Lexeme {
		case "for":
			return p.parseForExpr()
		case "let":
			return p.parseLetExpr()
		case "each":
			return p.parseEachExpr()
		case "function":
			return p.parseAnonFuncExpr()
		}
		// true/false
		if p.cur.Lexeme == "true" || p.cur.Lexeme == "false" {
			e := &BoolLit{V: p.cur.Lexeme == "true", P: p.cur.Pos}
			p.advance()
			return e, nil
		}
		e := &VarExpr{Name: p.cur.Lexeme, P: p.cur.Pos}
		p.advance()
		return e, nil
	case TokLBrack:
		pos := p.cur.Pos
		p.advance()
		var elems []Expr
		if p.cur.Kind != TokRBrack {
			first, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if p.cur.Kind == TokColon {
				p.advance()
				second, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				end := second
				var step Expr
				if p.cur.Kind == TokColon {
					p.advance()
					end, err = p.parseExpr()
					if err != nil {
						return nil, err
					}
					// OpenSCAD range syntax is [start:step:end].
					step = second
				}
				if err := p.expect(TokRBrack, "expected ']'"); err != nil {
					return nil, err
				}
				return &RangeLit{Start: first, End: end, Step: step, P: pos}, nil
			}
			elems = append(elems, first)
			for p.cur.Kind == TokComma {
				p.advance()
				if p.cur.Kind == TokRBrack {
					break
				}
				ex, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				elems = append(elems, ex)
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
		return nil, PosErrorf(p.cur.Pos, "expected expression")
	}
}

func (p *Parser) parseAnonFuncExpr() (Expr, error) {
	pos := p.cur.Pos
	if err := p.expectIdent("function"); err != nil {
		return nil, err
	}
	params, err := p.parseParamList()
	if err != nil {
		return nil, err
	}
	body, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return &FuncLitExpr{Params: params, Body: body, P: pos}, nil
}

func (p *Parser) parseForExpr() (Expr, error) {
	pos := p.cur.Pos
	if err := p.expectIdent("for"); err != nil {
		return nil, err
	}
	binds, err := p.parseForBinds()
	if err != nil {
		return nil, err
	}
	body, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return &ForExpr{Binds: binds, Body: body, P: pos}, nil
}

func (p *Parser) parseLetExpr() (Expr, error) {
	pos := p.cur.Pos
	if err := p.expectIdent("let"); err != nil {
		return nil, err
	}
	binds, err := p.parseLetBinds()
	if err != nil {
		return nil, err
	}
	body, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return &LetExpr{Binds: binds, Body: body, P: pos}, nil
}

func (p *Parser) parseEachExpr() (Expr, error) {
	pos := p.cur.Pos
	if err := p.expectIdent("each"); err != nil {
		return nil, err
	}
	x, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return &EachExpr{X: x, P: pos}, nil
}

func (p *Parser) parseForBinds() ([]ForBind, error) {
	rawBinds, err := p.parseNamedExprBinds(
		"expected '(' after for",
		"expected loop variable name",
		"expected '=' in for binding",
		"expected ')' after for bindings",
	)
	if err != nil {
		return nil, err
	}
	binds := make([]ForBind, 0, len(rawBinds))
	for _, b := range rawBinds {
		binds = append(binds, ForBind{Name: b.Name, Expr: b.Expr, P: b.P})
	}
	return binds, nil
}

func (p *Parser) parseLetBinds() ([]LetBind, error) {
	rawBinds, err := p.parseNamedExprBinds(
		"expected '(' after let",
		"expected let variable name",
		"expected '=' in let binding",
		"expected ')' after let bindings",
	)
	if err != nil {
		return nil, err
	}
	binds := make([]LetBind, 0, len(rawBinds))
	for _, b := range rawBinds {
		binds = append(binds, LetBind{Name: b.Name, Expr: b.Expr, P: b.P})
	}
	return binds, nil
}

func (p *Parser) parseNamedExprBinds(openErr, nameErr, assignErr, closeErr string) ([]namedExprBind, error) {
	if err := p.expect(TokLParen, openErr); err != nil {
		return nil, err
	}
	var binds []namedExprBind
	if p.cur.Kind != TokRParen {
		for {
			if p.cur.Kind != TokIdent {
				return nil, PosErrorf(p.cur.Pos, "%s", nameErr)
			}
			name := p.cur.Lexeme
			pos := p.cur.Pos
			p.advance()
			if err := p.expect(TokAssign, assignErr); err != nil {
				return nil, err
			}
			ex, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			binds = append(binds, namedExprBind{Name: name, Expr: ex, P: pos})
			if p.cur.Kind != TokComma {
				break
			}
			p.advance()
		}
	}
	if err := p.expect(TokRParen, closeErr); err != nil {
		return nil, err
	}
	return binds, nil
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
		return PosErrorf(p.cur.Pos, "%s", msg)
	}
	return p.advance()
}

func (p *Parser) expectIdent(want string) error {
	if p.cur.Kind != TokIdent || p.cur.Lexeme != want {
		return PosErrorf(p.cur.Pos, "expected %q", want)
	}
	return p.advance()
}
