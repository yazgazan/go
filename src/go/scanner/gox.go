package scanner

import (
	"fmt"
	"go/token"
)

type goxMode int

const (
	GO goxMode = iota
	GOX_TAG
	BARE_WORDS
)

type StackState struct {
	mode       goxMode
	braceDepth int
}

type goxStack struct {
	*StackState
	stack []StackState
}

func (s *goxStack) push(mode goxMode) {
	s.stack = append(s.stack, StackState{mode: mode, braceDepth: 0})
	s.StackState = &s.stack[len(s.stack)-1]
}

func (s *goxStack) pop() error {
	if len(s.stack) <= 1 {
		return fmt.Errorf("Unable to pop empty gox stack")
	}
	s.stack = s.stack[:len(s.stack)-1]
	s.StackState = &s.stack[len(s.stack)-1]
	return nil
}

func (s *Scanner) scanGoxIdentifier() string {
	offs := s.offset
	for isLetter(s.ch) || isDigit(s.ch) || s.ch == '.' {
		s.next()
	}
	return string(s.src[offs:s.offset])
}

// GOX closing tag
func (s *Scanner) scanCTag() string {
	// '<' opening already consumed, and we know a '/' is here
	offs := s.offset - 1
	for {
		ch := s.ch
		if ch == '\n' || ch < 0 {
			s.error(offs, "tag literal not terminated")
			break
		}
		s.next()
		if ch == '>' {
			break
		}
	}

	return string(s.src[offs+2 : s.offset-1])
}

func (s *Scanner) Scan() (pos token.Pos, tok token.Token, lit string) {
	var f func() (token.Pos, token.Token, string)
	switch s.goxState.mode {
	case GO:
		f = s.scanGoMode
	case GOX_TAG:
		f = s.scanGoxTagMode
	case BARE_WORDS:
		f = s.scanBareWordsMode
	}

	pos, tok, lit = f()
	return
}

func (s *Scanner) scanGoxTagMode() (pos token.Pos, tok token.Token, lit string) {
	s.insertSemi = false
	s.skipWhitespace()

	// current token start
	pos = s.file.Pos(s.offset)

	switch ch := s.ch; {
	case isLetter(ch):
		// lit = s.scanGoxIdentifier()
		lit = s.scanIdentifier()
		tok = token.IDENT
	default:
		s.next()
		switch ch {
		case -1:
			s.error(s.offset, "reached illegal EOF in gox tag")
		case '=':
			tok = token.ASSIGN
		case '{':
			tok = token.LBRACE
			// push Go mode onto the stack
			s.goxState.push(GO)
		case '(':
			tok = token.LPAREN
			s.goxState.push(GO)

		case '"':
			tok = token.STRING
			// TODO(danny) Escape gox strings with XML rules
			lit = s.scanString()

		// case '[':
		// 	tok = token.LBRACK
		// case ']':
		// 	tok = token.RBRACK

		case '>':
			tok = token.OTAG_END
			// Pop stack and push bare words onto stack
			err := s.goxState.pop()
			s.goxState.push(BARE_WORDS)
			if err != nil {
				s.error(s.offset, err.Error())
			}

		case '.':
			// fractions starting with a '.' are handled by outer switch
			tok = token.PERIOD
			if s.ch == '.' && s.peek() == '.' {
				s.next()
				s.next() // consume last '.'
				tok = token.ELLIPSIS
			}

		case '/':
			if s.ch == '>' {
				s.next()
				// TODO make them supported
				tok = token.OTAG_SELF_CLOSE
				err := s.goxState.pop()
				if err != nil {
					s.error(s.offset, err.Error())
				}
				s.insertSemi = true
			}
		default:
			// next reports unexpected BOMs - don't repeat
			if ch != bom {
				s.error(s.file.Offset(pos), fmt.Sprintf("illegal character %#U", ch))
			}
			tok = token.ILLEGAL
			lit = string(ch)
		}
	}

	// Save the last token for gox
	s.lastToken = tok

	return
}

func (s *Scanner) scanBareWordsMode() (pos token.Pos, tok token.Token, lit string) {
	s.insertSemi = false
	// Whitespace matters in Bare Words mode, so do not call s.skipWhitespace()

	// current token start
	pos = s.file.Pos(s.offset)

	switch s.ch {
	case '{':
		s.next()
		tok = token.LBRACE
		// push Go mode onto the stack
		s.goxState.push(GO)
	case '<':
		s.next()
		switch {
		case s.ch == '/':
			tok = token.CTAG
			lit = s.scanCTag()
			// pop state
			err := s.goxState.pop()
			if err != nil {
				s.error(s.offset, err.Error())
				tok = token.ILLEGAL
			}
			// insert semi if we've popped into go mode
			// (needed for "asdf := <div></div>" lines)
			if s.goxState.mode == GO {
				s.insertSemi = true
			}
		case isLetter(s.ch) == true:
			tok = token.OTAG
			// push gox-tag
			s.goxState.push(GOX_TAG)
		}
	default:
		// Parse bare words
		offs := s.offset
		for {
			if s.ch < 0 {
				s.error(offs, "end of file during gox tag")
				tok = token.ILLEGAL
				return
			}
			if s.ch == '{' || s.ch == '<' {
				break
			}
			s.next()
		}

		lit = string(s.src[offs:s.offset])
		tok = token.BARE_WORDS
	}

	// Save the last token for gox
	s.lastToken = tok

	return
}

// GoxLegal returns whether a gox tag (<XML syntax>) can follow the given token.
// Otherwise, a "<" sign is interpreted as a less than.
func goxLegal(tok token.Token) bool {
	switch tok {

	case token.ASSIGN, token.EQL, token.NEQ, token.DEFINE,
		token.LPAREN, token.LBRACE, token.COMMA, token.COLON,
		token.RETURN, token.IF, token.SWITCH, token.CASE:

		return true
	}

	return false
}
