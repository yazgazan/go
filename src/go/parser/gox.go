package parser

import (
	"fmt"
	"go/ast"
	"go/token"
)

func (p *parser) parseGoxTag() ast.Expr {
	if p.trace {
		defer un(trace(p, "GoxTag"))
	}

	otag := p.expect(token.OTAG)

	tagName := p.checkExpr(p.parseRhs())

	attrs := []*ast.GoxAttrStmt{}
	for p.tok != token.OTAG_END && p.tok != token.OTAG_SELF_CLOSE && p.tok != token.EOF {
		attr := p.parseGoxAttr()
		if attr == nil {
			return nil
		}
		attrs = append(attrs, attr)
	}
	if p.tok == token.EOF {
		p.error(p.pos, "Unexpected EOF")
		return nil
	}
	// if a self closing tag, close
	if p.tok == token.OTAG_SELF_CLOSE {
		lit := p.lit
		ctagpos := p.expect(token.OTAG_SELF_CLOSE)
		return &ast.GoxExpr{
			Otag: otag, TagName: tagName,
			Attrs: attrs, X: nil,
			Ctag: &ast.CtagExpr{
				Close: ctagpos,
				Value: lit,
			},
		}
	}
	p.expect(token.OTAG_END)

	p.exprLev++ // we're in the expression

	var content []ast.Expr // tag contents

	for p.tok != token.CTAG {
		switch p.tok {
		case token.LBRACE:
			content = append(content, p.parseGoExpr())
		case token.BARE_WORDS:
			content = append(content, p.parseBareWords())
		case token.OTAG:
			content = append(content, p.parseGoxTag())
		default:
			p.error(p.pos, "Unexpected token in gox tag")
			p.exprLev--
			p.next()
			return &ast.BadExpr{From: p.pos, To: p.pos + 1}
		}
	}

	lit := p.lit
	ctagpos := p.expect(token.CTAG)
	ctag := &ast.CtagExpr{Close: ctagpos, Value: lit}

	p.exprLev--

	if ast.GoxName(tagName) != lit {
		p.error(ctagpos, fmt.Sprintf("tag %q not closed", tagName))
		p.next()
		panic(fmt.Sprintf("tag %q not closed", tagName))
		return &ast.BadExpr{From: ctagpos, To: ctagpos + 1}
	}

	return &ast.GoxExpr{Otag: otag, TagName: tagName, Attrs: attrs, X: content, Ctag: ctag}
}

func (p *parser) parseGoxAttr() *ast.GoxAttrStmt {
	if p.trace {
		defer un(trace(p, "GoxAttrStmt"))
	}

	lhs := p.parseIdent()
	if p.tok != token.ASSIGN {
		return &ast.GoxAttrStmt{Lhs: lhs, Rhs: nil}
	}
	p.expect(token.ASSIGN)
	var rhs ast.Expr
	switch p.tok {
	case token.LBRACE:
		rhs = p.parseGoExpr()
	case token.STRING, token.CHAR:
		rhs = p.parseRhs() // yeaaaah
	default:
		p.error(p.pos, "Encountered illegal attribute value in gox tag")
		return nil
	}

	return &ast.GoxAttrStmt{Lhs: lhs, Rhs: rhs}
}

func (p *parser) parseBareWords() *ast.BareWordsExpr {
	if p.trace {
		defer un(trace(p, "BareWordsExpr"))
	}

	lit := p.lit
	pos := p.expect(token.BARE_WORDS)

	return &ast.BareWordsExpr{ValuePos: pos, Value: lit}
}

func (p *parser) parseGoExpr() *ast.GoExpr {
	lPos := p.expect(token.LBRACE)
	expr := p.parseRhs()
	rPos := p.expect(token.RBRACE)
	return &ast.GoExpr{Lbrace: lPos, X: expr, Rbrace: rPos}
}
