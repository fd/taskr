package router

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/juju/errors"
)

type Variable struct {
	Name     string
	MinCount int
	MaxCount int
	Pattern  string
}

func ExtractVariables(pattern string) ([]Variable, error) {
	tokens, err := parse(pattern)
	if err != nil {
		return nil, err
	}

	vars := make([]Variable, 0, 4)
	for _, token := range tokens {
		if token.kind != tVARIABLE {
			continue
		}

		vars = append(vars, Variable{
			Name:     token.name,
			MinCount: token.min,
			MaxCount: token.max,
			Pattern:  token.patternSrc,
		})
	}

	return vars, nil
}

func parse(pattern string) ([]token, error) {
	bytes := []byte(pattern)

	p := &parser{
		bytes:  bytes,
		length: len(bytes),
	}

	return p.Parse()
}

const (
	rEND = rune(-1)
	rERR = rune(-2)
)

const (
	tEPSILON = 1 + iota
	tLITERAL
	tVARIABLE
)

type parser struct {
	bytes  []byte
	cur    rune
	offset int
	length int
	err    error

	tokens           []token
	paramNames       map[string]bool
	nextNumericParam int
}

type token struct {
	kind int

	// tEPSILON
	epsilon rune

	// tLITERAL
	literal []byte

	// tVARIABLE
	name       string
	patternSrc string
	pattern    *regexp.Regexp
	min, max   int
}

func (p *parser) pushToken(t token) {
	p.tokens = append(p.tokens, t)
}

func (p *parser) setError(err error) {
	if p.cur != rERR {
		p.cur = rERR
		p.err = err
	}
}

func (p *parser) setOffset(offset int) bool {
	if p.cur == rERR {
		return false
	}

	if offset < 0 {
		offset = p.length + offset
	}

	if offset < p.length {
		p.offset = offset
		p.cur = rune(p.bytes[p.offset])
		return true
	}

	p.offset = p.length
	p.cur = rEND
	return false
}

func (p *parser) seek(steps int) bool {
	return p.setOffset(p.offset + steps)
}

func (p *parser) step() bool {
	return p.seek(1)
}

func (p *parser) Parse() ([]token, error) {
	p.setOffset(0)
	p.tokens = nil
	p.paramNames = make(map[string]bool)
	p.nextNumericParam = 1

	if p.cur != '/' {
		p.setError(errors.Errorf("expected a '/'"))
	}

LOOP:
	for {
		switch p.cur {

		case rERR:
			return nil, p.err

		case rEND:
			break LOOP

		case '/':
			p.parseEpsilon()

		case '{':
			p.parseVariable()

		default:
			p.parseLiteral()

		}
	}

	for {
		l := len(p.tokens)
		if l == 0 {
			break
		}
		l--

		t := p.tokens[l]
		if t.kind != tEPSILON {
			break
		}

		p.tokens = p.tokens[:l]
	}

	if len(p.tokens) == 0 {
		p.pushToken(token{kind: tEPSILON, epsilon: '/'})
	}

	p.pushToken(token{kind: tEPSILON, epsilon: rEND})
	return p.tokens, nil
}

func (p *parser) parseEpsilon() {
	switch p.cur {

	case rERR, rEND:
		return

	case '/':
		for p.cur == '/' {
			p.step()
		}
		p.pushToken(token{kind: tEPSILON, epsilon: '/'})

	default:
		p.setError(errors.Errorf("unexpected character %q", p.cur))

	}
}

func (p *parser) parseLiteral() {
	beg := p.offset

LOOP:
	for {
		switch p.cur {

		case rERR:
			return

		case rEND, '/':
			break LOOP

		default:
			p.step()

		}
	}

	end := p.offset

	p.pushToken(token{kind: tLITERAL, literal: p.bytes[beg:end]})
}

// name(regexp){1,5}
// name(regexp){1,}
// name(regexp){,5}
// name(regexp){5}
// name(regexp)+
// name(regexp)*
// name(regexp)?
func (p *parser) parseVariable() {
	p.step() // '{'

	var (
		name, ok1       = p.parseVariableName()
		patternSrc, ok2 = p.parseVariablePattern()
		min, max, ok3   = p.parseVariableRepeat()
		pattern         *regexp.Regexp
		err             error
	)

	if !(ok1 && ok2 && ok3) {
		return
	}

	if p.cur != '}' {
		p.setError(errors.Errorf("expected '}'"))
		return
	}
	p.step()

	if name == "" {
		name = strconv.Itoa(p.nextNumericParam)
		p.nextNumericParam++
		for p.paramNames[name] {
			name = strconv.Itoa(p.nextNumericParam)
			p.nextNumericParam++
		}
	}

	if p.paramNames[name] {
		p.setError(errors.Errorf("param name is already used: %q", name))
		return
	}

	if patternSrc != "" {
		pattern, err = regexp.Compile("\\A" + patternSrc + "\\z")
		if err != nil {
			p.setError(err)
			return
		}
	}

	p.paramNames[name] = true

	p.pushToken(token{kind: tVARIABLE, name: name, patternSrc: patternSrc, pattern: pattern, min: min, max: max})
}

func (p *parser) parseVariableName() (string, bool) {
	var (
		beg = p.offset
		end int
	)

LOOP:
	for {
		switch p.cur {

		case '(', '{', '}', '?', '+', '*':
			break LOOP

		case rERR:
			return "", false

		case rEND:
			p.setError(errors.Errorf("invallid pattern"))
			return "", false

		default:
			p.step()

		}
	}

	end = p.offset
	return string(p.bytes[beg:end]), true
}

func (p *parser) parseVariablePattern() (string, bool) {
	var (
		beg   = p.offset
		end   int
		count = 0
	)

	if p.cur != '(' {
		return "", true
	}

LOOP:
	for {
		switch p.cur {

		case '\\':
			p.step()
			p.step()

		case '(':
			p.step()
			count++

		case ')':
			p.step()
			count--
			if count == 0 {
				break LOOP
			}

		case rERR:
			return "", false

		case rEND:
			p.setError(errors.Errorf("invallid pattern"))
			return "", false

		default:
			p.step()

		}
	}

	end = p.offset
	return string(p.bytes[beg:end]), true
}

func (p *parser) parseVariableRepeat() (min, max int, ok bool) {

	switch p.cur {

	case '?':
		p.step()
		return 0, 1, true

	case '+':
		p.step()
		return 1, -1, true

	case '*':
		p.step()
		return 0, -1, true

	case '{':
		p.step()

		if p.cur == ',' {
			// {,max}
			p.step()
			min = 0
		} else {
			min, ok = p.parseInt()
			if !ok {
				p.setError(errors.Errorf("expected a number"))
				return 0, 0, false
			}

			// {n}
			if p.cur == '}' {
				p.step()
				return min, min, true
			}

			// {min,xxx}
			if p.cur == ',' {
				p.step()
			}
		}

		if p.cur == '}' {
			// {min,}
			p.step()
			return min, -1, true
		}

		{
			// {min,max}
			max, ok = p.parseInt()
			if !ok {
				p.setError(errors.Errorf("expected a number"))
				return 0, 0, false
			}

			if p.cur == '}' {
				p.step()
				return min, max, true
			}

			p.setError(errors.Errorf("invalid repeat pattern"))
			return 0, 0, false
		}

	case '}':
		return 1, 1, true

	case rERR:
		return 0, 0, false

	}

	p.setError(errors.Errorf("invalid repeat pattern"))
	return 0, 0, false
}

func (p *parser) parseInt() (int, bool) {
	var (
		beg = p.offset
		end int
	)

	for '0' <= p.cur && p.cur <= '9' {
		p.step()
	}

	end = p.offset

	n, err := strconv.Atoi(string(p.bytes[beg:end]))
	if err != nil {
		p.setError(err)
		return 0, false
	}

	return n, true
}

func (t token) String() string {
	switch t.kind {
	case tEPSILON:
		if t.epsilon == rEND {
			return "eps(end)"
		}
		return fmt.Sprintf("eps(%q)", t.epsilon)
	case tLITERAL:
		return fmt.Sprintf("lit(%q)", string(t.literal))
	case tVARIABLE:
		if t.patternSrc == "" {
			return fmt.Sprintf("var(%q, none, %d, %d)", t.name, t.min, t.max)
		}
		return fmt.Sprintf("var(%q, %q, %d, %d)", t.name, t.patternSrc, t.min, t.max)
	}
	panic("oops")
}
