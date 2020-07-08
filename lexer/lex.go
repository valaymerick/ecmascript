package lexer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"unicode"
)

// Scanner holds the state of the scanner.
type Scanner struct {
	r         io.RuneReader // input reader
	peekRunes []rune        // peek runes queue
	buf       bytes.Buffer  // input buffer to hold current lexeme
}

// New creates a new Scanner.
func New(r *io.RuneReader) *Scanner {
	return &Scanner{
		r: *r,
	}
}

// nextRune reads the next rune from the input.
func (l *Scanner) nextRune() rune {
	r, _, err := l.r.ReadRune()
	if err != nil {
		if err != io.EOF {
			fmt.Fprintln(os.Stderr)
		}
		r = -1 // EOF rune
	}
	return r
}

// read consumes the peekRunes queue then calls nextRune.
func (l *Scanner) read() rune {
	if len(l.peekRunes) > 0 {
		r := l.peekRunes[0]
		l.peekRunes = l.peekRunes[1:]
		return r
	}
	return l.nextRune()
}

// peek returns but does not consume the next n rune in the input.
func (l *Scanner) peek(n int) rune {
	if len(l.peekRunes) >= n {
		return l.peekRunes[n-1]
	}

	p := l.nextRune()
	l.peekRunes = append(l.peekRunes, p)

	return p
}

// resetPeek resets the peekRunes queue and calls mkToken
func (l *Scanner) tok(typ Type, text string) *Token {
	l.peekRunes = nil
	return mkToken(typ, text)
}

// next returns the next token.
func (l *Scanner) next() *Token {
	for {
		r := l.read()
		switch {
		case r == '@':
			return mkToken(tokAt, "@")
		case isSpace(r):
		case isIdentifierStart(r):
			// names and keywords
			return l.alphanum(tokIdentifier, r)
		case isNumber(r):
		case isPunctuator(r):
			return l.lexPunctuator(r)
		}
	}
}

// lexPunctuator returns the next punctuator token
func (l *Scanner) lexPunctuator(r rune) *Token {
	switch r {
	case '(':
		return mkToken(tokOpenParen, "(")
	case ')':
		return mkToken(tokCloseParen, ")")
	case '{':
		return mkToken(tokOpenBrace, "{")
	case '}':
		return mkToken(tokCloseBrace, "}")
	case '[':
		return mkToken(tokOpenBracket, "[")
	case ']':
		return mkToken(tokCloseBracket, "]")
	case ',':
		return mkToken(tokComma, ",")
	case ':':
		return mkToken(tokColon, ":")
	case ';':
		return mkToken(tokSemicolon, ";")
	case '~':
		return mkToken(tokTilde, "~")

	case '=':
		// '=' or '=>' or '==' or '==='
		switch l.peek(1) {
		case '=':
			if l.peek(2) == '=' {
				return l.tok(tokEqualsEqualsEquals, "===")
			}
			return l.tok(tokEqualsEquals, "==")
		case '>':
			return l.tok(tokEqualsGreaterThan, "=>")
		}
		return l.tok(tokEquals, "=")

	case '+':
		// '+' or '+=' or '++'
		switch l.peek(1) {
		case '=':
			return l.tok(tokPlusEquals, "+=")
		case '+':
			return l.tok(tokPlusPlus, "++")
		}
		return l.tok(tokPlus, "+")

	case '-':
		// '-' or '-=' or '--'
		switch l.peek(1) {
		case '=':
			return l.tok(tokMinusEquals, "-=")
		case '-':
			return l.tok(tokMinusMinus, "--")
		}
		return l.tok(tokMinus, "-")

	case '*':
		// '*' or '*=' or '**' or '**='
		switch l.peek(1) {
		case '=':
			return l.tok(tokAsteriskEquals, "*=")
		case '*':
			if l.peek(2) == '=' {
				return l.tok(tokAsteriskAsteriskEquals, "**=")
			}
			return l.tok(tokAsteriskAsterisk, "**")
		}
		return l.tok(tokAsterisk, "*")

	case '/':
		// '/' or '/=' or '//' or '/* ... */'
		switch l.peek(1) {
		case '=':
			return l.tok(tokSlashEquals, "/=")
		case '/':
			// Single line comment
		case '*':
			// Multi line comment
		}
		return l.tok(tokSlash, "/")

	case '>':
		// '>' or '>>' or '>>>' or '>=' or '>>=' or '>>>='
		switch l.peek(1) {
		case '>':
			switch l.peek(2) {
			case '>':
				if l.peek(3) == '=' {
					return l.tok(tokGreaterThanGreaterThanGreaterThanEquals, ">>>=")
				}
				return l.tok(tokGreaterThanGreaterThanGreaterThan, ">>>")
			case '=':
				return l.tok(tokGreaterThanGreaterThanEquals, ">>=")
			}
			return l.tok(tokGreaterThanGreaterThan, ">>")
		case '=':
			return l.tok(tokGreaterThanEquals, ">=")
		}
		return l.tok(tokGreaterThan, ">")

	case '<':
		// '<' or '<<' or '<=' or '<<='
		switch l.peek(1) {
		case '<':
			if l.peek(2) == '=' {
				return l.tok(tokLessThanLessThanEquals, "<<=")
			}
			return l.tok(tokLessThanLessThan, "<<")
		case '=':
			return l.tok(tokLessThanEquals, "<=")
		}
		return l.tok(tokLessThan, "<")

	case '!':
		// '!' or '!=' or '!=='
		if l.peek(1) == '=' {
			if l.peek(2) == '=' {
				return l.tok(tokExclamationEqualsEquals, "!==")
			}
			return l.tok(tokExclamationEquals, "!=")
		}
		return l.tok(tokExclamation, "!")

	case '^':
		// '^' or '^='
		if l.peek(1) == '=' {
			return l.tok(tokCaretEquals, "^=")
		}
		return l.tok(tokCaret, "^")

	case '|':
		// '|' or '|=' or '||' or '||='
		switch l.peek(1) {
		case '=':
			return l.tok(tokBarEquals, "|=")
		case '|':
			if l.peek(2) == '=' {
				return l.tok(tokBarBarEquals, "||=")
			}
			return l.tok(tokBarBar, "||")
		}
		return l.tok(tokBar, "|")

	case '&':
		// '&' or '&=' or '&&' or '&&='
		switch l.peek(1) {
		case '=':
			return l.tok(tokAmpersandEquals, "&=")
		case '&':
			if l.peek(2) == '=' {
				return l.tok(tokAmpersandAmpersandEquals, "&&=")
			}
			return l.tok(tokAmpersandAmpersand, "&&")
		}
		return l.tok(tokAmpersand, "&")

	case '%':
		// '%' or '%='
		if l.peek(1) == '=' {
			return l.tok(tokPercentEquals, "%=")
		}
		return l.tok(tokPercent, "%")

	case '?':
		// '?' or '?.' or '??' or '??='
		switch l.peek(1) {
		case '?':
			if l.peek(2) == '=' {
				return l.tok(tokQuestionQuestionEquals, "??=")
			}
			return l.tok(tokQuestionQuestion, "??")
		case '.':
			// Differentiate optional chaining punctuators (?.id) from conditional operators (? :)
			if !isNumber(l.peek(2)) {
				return l.tok(tokQuestionDot, "?.")
			}
		}
		return l.tok(tokQuestion, "?")
	default:
		return l.tok(tokSyntaxError, "")

	}
}

// accum appends the current rune to the buffer until
// the valid function returns false
func (l *Scanner) accum(r rune, valid func(rune) bool) {
	l.buf.Reset()
	for {
		l.buf.WriteRune(r)
		r = l.read()
		if r == -1 {
			return
		}
		if !valid(r) {
			return
		}
	}
}

// alphanum creates a keyword or identifier token using the buffer.
func (l *Scanner) alphanum(typ Type, r rune) *Token {
	l.accum(r, isIdentifierContinue)
	return mkToken(typ, l.buf.String())
}

// alphanum creates a numeric literal token using the buffer.
// func (l *Scanner) number(r rune) *Token {

// }

// isAlphaNumeric reports whether r is a letter, digit, or underscore.
func isAlphanum(r rune) bool {
	return r == '_' || r == '$' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// isNumber reports whether r is a numeric literal.
func isNumber(r rune) bool {
	return '0' <= r && r <= '9'
}

// isPunctuator reports whether r is a punctuator
func isPunctuator(r rune) bool {
	switch r {
	case '{', '}', '(', ')', '[', ']', '.', ';', ',', '<', '>', '=', '!', '+', '-', '*', '%', '&', '|', '^', '~', '?', ':', '/':
		return true
	default:
		return false
	}
}

// isSpace checks whether r is a space as defined
// in the Unicode standard or the ECMAScript specification.
func isSpace(r rune) bool {
	switch {
	case r == 0x85:
		return false
	case
		unicode.IsSpace(r),
		r == '\uFEFF': // zero width non-breaking space
		return true

	default:
		return false
	}
}
