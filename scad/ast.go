package scad

type Program struct {
	Stmts []Stmt
}

type Stmt interface {
	stmtNode()
	pos() Pos
}

type Expr interface {
	exprNode()
	pos() Pos
}

// ---- statements ----

type AssignStmt struct {
	Name string
	Expr Expr
	P    Pos
}

func (*AssignStmt) stmtNode() {}
func (s *AssignStmt) pos() Pos { return s.P }

type BlockStmt struct {
	Stmts []Stmt
	P     Pos
}

func (*BlockStmt) stmtNode() {}
func (s *BlockStmt) pos() Pos { return s.P }

type IfStmt struct {
	Cond Expr
	Then Stmt
	Else Stmt // may be nil
	P    Pos
}

func (*IfStmt) stmtNode() {}
func (s *IfStmt) pos() Pos { return s.P }

type Param struct {
	Name    string
	Default Expr // may be nil
	P       Pos
}

type ModuleDefStmt struct {
	Name   string
	Params []Param
	Body   *BlockStmt
	P      Pos
}

func (*ModuleDefStmt) stmtNode() {}
func (s *ModuleDefStmt) pos() Pos { return s.P }

type FuncDefStmt struct {
	Name   string
	Params []Param
	Body   Expr
	P      Pos
}

func (*FuncDefStmt) stmtNode() {}
func (s *FuncDefStmt) pos() Pos { return s.P }

type Arg struct {
	Name string // optional
	Expr Expr
	P    Pos
}

type Call struct {
	Name string
	Args []Arg
	P    Pos
}

type CallStmt struct {
	Call     Call
	Children []Stmt // 0 = no children
	P        Pos
}

func (*CallStmt) stmtNode() {}
func (s *CallStmt) pos() Pos { return s.P }

// ---- expressions ----

type NumberLit struct {
	V float64
	P Pos
}

func (*NumberLit) exprNode() {}
func (e *NumberLit) pos() Pos { return e.P }

type BoolLit struct {
	V bool
	P Pos
}

func (*BoolLit) exprNode() {}
func (e *BoolLit) pos() Pos { return e.P }

type StringLit struct {
	V string
	P Pos
}

func (*StringLit) exprNode() {}
func (e *StringLit) pos() Pos { return e.P }

type VarExpr struct {
	Name string
	P    Pos
}

func (*VarExpr) exprNode() {}
func (e *VarExpr) pos() Pos { return e.P }

type ArrayLit struct {
	Elems []Expr
	P     Pos
}

func (*ArrayLit) exprNode() {}
func (e *ArrayLit) pos() Pos { return e.P }

type UnaryExpr struct {
	Op TokenKind
	X  Expr
	P  Pos
}

func (*UnaryExpr) exprNode() {}
func (e *UnaryExpr) pos() Pos { return e.P }

type BinaryExpr struct {
	Op TokenKind
	L  Expr
	R  Expr
	P  Pos
}

func (*BinaryExpr) exprNode() {}
func (e *BinaryExpr) pos() Pos { return e.P }

type TernaryExpr struct {
	Cond Expr
	Then Expr
	Else Expr
	P    Pos
}

func (*TernaryExpr) exprNode() {}
func (e *TernaryExpr) pos() Pos { return e.P }

type CallExpr struct {
	Call Call
	P    Pos
}

func (*CallExpr) exprNode() {}
func (e *CallExpr) pos() Pos { return e.P }
