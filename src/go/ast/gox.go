package ast

import (
	"go/token"
	"unicode"
)

type (
	BareWordsExpr struct {
		ValuePos token.Pos
		Value    string
	}

	GoxExpr struct {
		Otag    token.Pos      // position of <
		TagName Expr           // div
		Attrs   []*GoxAttrStmt // props
		X       []Expr         // expression(s) inside GoxTag or none
		Ctag    *CtagExpr      // </asdf> or />
	}

	GoExpr struct {
		Lbrace token.Pos
		X      Expr
		Rbrace token.Pos
	}

	CtagExpr struct {
		Close token.Pos // position of "<" or "/"
		Value string
	}

	// GOX attribute
	GoxAttrStmt struct {
		Lhs *Ident
		Rhs Expr // can be nil
	}
)

func (x *BareWordsExpr) Pos() token.Pos { return x.ValuePos }
func (x *GoxExpr) Pos() token.Pos       { return x.Otag }
func (x *GoExpr) Pos() token.Pos        { return x.Lbrace }
func (x *CtagExpr) Pos() token.Pos      { return x.Close }
func (s *GoxAttrStmt) Pos() token.Pos   { return s.Lhs.Pos() }

func (x *GoxExpr) End() token.Pos       { return x.Ctag.End() }
func (x *GoExpr) End() token.Pos        { return x.Rbrace }
func (x *BareWordsExpr) End() token.Pos { return token.Pos(int(x.ValuePos) + len(x.Value)) }
func (x *CtagExpr) End() token.Pos      { return token.Pos(int(x.Close) + len(x.Value)) }
func (s *GoxAttrStmt) End() token.Pos {
	if s.Rhs == nil {
		return s.Lhs.End()
	} else {
		return s.Rhs.End()
	}
}

func (*GoxExpr) exprNode()       {}
func (*GoExpr) exprNode()        {}
func (*BareWordsExpr) exprNode() {}
func (*CtagExpr) exprNode()      {}

func (*GoxAttrStmt) stmtNode() {}

func IsGoxComponent(expr Expr) bool {
	switch t := expr.(type) {
	default:
		return false
	case *CallExpr, *SelectorExpr:
		return true
	case *Ident:
		return unicode.IsUpper(rune(t.Name[0]))
	}
}
