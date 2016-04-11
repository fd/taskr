package router

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/net/context"
)

type compiler struct {
	nextHandlerID int
	root          node
}

type node struct {
	kind     int
	children []node
	handlers []handler
	id       int
	size     int
	depth    int

	// tEPSILON
	epsilon rune

	// tLITERAL
	literal []byte

	// tVARIABLE
	names      []string
	patternSrc string
	pattern    *regexp.Regexp
	min, max   int
}

type handler struct {
	id int
	Handler
}

type Handler interface {
	ServeHTTP(ctx context.Context, rw http.ResponseWriter, req *http.Request) error
}

type HandlerFunc func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error

func (h HandlerFunc) ServeHTTP(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
	return h(ctx, rw, req)
}

func (c *compiler) Insert(pattern string, h Handler) error {
	tokens, err := parse(pattern)
	if err != nil {
		return err
	}

	id := c.nextHandlerID
	c.nextHandlerID++

	c.insert(tokens, handler{id, h})
	return nil
}

func (c *compiler) Optimize() {
	c.root.Optimize()
	c.root.identify(0, 0)
}

func (c *compiler) Compile() []instruction {
	onErr := jumpPointer{inst: -1, keepFrames: 0}

	prog := c.root.Compile(onErr, nil)
	for _, inst := range prog {
		jump := inst.Jump()
		if jump.inst >= 0 {
			jump.keepFrames = prog[jump.inst].Frames() - 1
		} else {
			jump.keepFrames = 0
		}
	}

	return prog
}

func (c *compiler) insert(tokens []token, handler handler) {
	if c.root.kind == 0 {
		c.root = node{
			kind:    tEPSILON,
			epsilon: '/',
		}
	}

	c.root.Insert(tokens, handler)
}

func (n *node) Insert(tokens []token, handler handler) {
	if len(tokens) == 0 {
		panic("should not happen")
	}

	token := tokens[0]

	if len(tokens) == 1 {
		if token.kind != tEPSILON || token.epsilon != rEND {
			panic("should not happen")
		}

		n.handlers = append(n.handlers, handler)
		return
	}

	switch n.kind {

	case tEPSILON:
		// continue
		tokens = tokens[1:]

	case tVARIABLE:
		found := false
		for _, name := range n.names {
			if name == token.name {
				found = true
				break
			}
		}
		if !found {
			n.names = append(n.names, token.name)
		}
		tokens = tokens[1:]

	case tLITERAL:
		idx, equal, split := n.findLiteralSplit(token.literal)

		if split {
			n.children = []node{{
				kind:     tLITERAL,
				literal:  n.literal[idx:],
				children: n.children,
				handlers: n.handlers,
			}}
			n.literal = n.literal[:idx]
			n.handlers = nil

			token.literal = token.literal[idx:]
			tokens[0] = token

		} else if equal {
			tokens = tokens[1:]

		} else {
			token.literal = token.literal[idx:]
			tokens[0] = token

		}

	default:
		panic("should not happen")

	}

	token = tokens[0]
	for idx, child := range n.children {
		if child.canInsertToken(&token) {
			child.Insert(tokens, handler)
			n.children[idx] = child
			return
		}
	}

	switch token.kind {

	case tEPSILON:
		child := node{
			kind:    tEPSILON,
			epsilon: token.epsilon,
		}
		child.Insert(tokens, handler)
		n.children = append(n.children, child)

	case tLITERAL:
		child := node{
			kind:    tLITERAL,
			literal: token.literal,
		}
		child.Insert(tokens, handler)
		n.children = append(n.children, child)

	case tVARIABLE:
		child := node{
			kind:       tVARIABLE,
			names:      []string{token.name},
			pattern:    token.pattern,
			patternSrc: token.patternSrc,
			min:        token.min,
			max:        token.max,
		}
		child.Insert(tokens, handler)
		n.children = append(n.children, child)

	default:
		panic("should not happen")

	}
}

func (n *node) canInsertToken(t *token) bool {
	if n.kind != t.kind {
		return false
	}

	switch n.kind {

	case tEPSILON:
		return n.epsilon == t.epsilon

	case tLITERAL:
		return n.literal[0] == t.literal[0]

	case tVARIABLE:
		return n.pattern == t.pattern &&
			n.min == t.min &&
			n.max == t.max

	}

	panic("should not happen")
}

func (n *node) findLiteralSplit(lit []byte) (int, bool, bool) {
	var (
		lenNode  = len(n.literal)
		lenToken = len(lit)
		idx      int
	)

	for ; idx < lenNode && idx < lenToken; idx++ {
		if n.literal[idx] != lit[idx] {
			return idx, false, true
		}
	}

	return idx, lenNode == lenToken, false
}

func (n *node) String() string {
	var str string

	if len(n.children) > 0 || len(n.handlers) > 0 {
		str = "┬╴"
	} else {
		str = "─╴"
	}

	switch n.kind {
	case tEPSILON:
		if n.epsilon == rEND {
			str += "eps(end)"
		} else {
			str += fmt.Sprintf("eps(%d, %q)", n.size, n.epsilon)
		}
	case tLITERAL:
		str += fmt.Sprintf("lit(%d, %q)", n.size, string(n.literal))
	case tVARIABLE:
		if n.patternSrc == "" {
			str += fmt.Sprintf("var(%d, %v, none, %d, %d)", n.size, n.names, n.min, n.max)
		} else {
			str += fmt.Sprintf("var(%d, %v, %q, %d, %d)", n.size, n.names, n.patternSrc, n.min, n.max)
		}
	default:
		panic("oops")
	}

	for i, h := range n.handlers {
		if i == len(n.handlers)-1 && len(n.children) == 0 {
			str += fmt.Sprintf("\n└─╴handler(%d): %#v", h.id, h.Handler)
		} else {
			str += fmt.Sprintf("\n├─╴handler(%d): %#v", h.id, h.Handler)
		}
	}

	for i, c := range n.children {
		if i == 0 && 1 == len(n.children) {
			str += "\n└" + strings.Replace(c.String(), "\n", "\n ", -1)
		} else if i == 0 {
			str += "\n├" + strings.Replace(c.String(), "\n", "\n│", -1)
		} else if i == len(n.children)-1 {
			str += "\n└" + strings.Replace(c.String(), "\n", "\n ", -1)
		} else {
			str += "\n├" + strings.Replace(c.String(), "\n", "\n│", -1)
		}
	}

	return str
}

func (n *node) Optimize() {

	n.size = 0

	for idx, c := range n.children {
		c.Optimize()
		n.size += c.size + 1
		n.children[idx] = c
	}

	sort.Sort(sizeSortedNodes(n.children))
}

func (n *node) identify(id, depth int) int {
	n.id = id
	n.depth = depth

	for idx, c := range n.children {
		id = c.identify(id+1, depth+1)
		n.children[idx] = c
	}

	return id
}

func (n *node) Compile(onErr jumpPointer, prog []instruction) []instruction {
	switch n.kind {

	case tEPSILON:
		prog = n.compileEpsilon(onErr, prog)

	case tLITERAL:
		prog = n.compileLiteral(onErr, prog)

	case tVARIABLE:
		prog = n.compileVariable(onErr, prog)

	}

	l := len(n.children) - 1
	for idx, c := range n.children {
		nextErr := onErr
		if idx < l {
			nextErr.inst = n.children[idx+1].id
		}
		prog = c.Compile(nextErr, prog)
	}

	return prog
}

func (n *node) compileEpsilon(onErr jumpPointer, prog []instruction) []instruction {

	if n.epsilon == rEND {
		inst := &instMatchEnd{frameCount: n.depth + 1, jump: onErr, handlers: n.handlers}
		return append(prog, inst)
	}

	inst := &instMatchEpsilon{frameCount: n.depth + 1, onErr: onErr}
	return append(prog, inst)
}

func (n *node) compileLiteral(onErr jumpPointer, prog []instruction) []instruction {

	if len(n.literal) == 1 {
		inst := &instMatchByte{frameCount: n.depth + 1, onErr: onErr, b: n.literal[0]}
		return append(prog, inst)
	}

	inst := &instMatchBytes{frameCount: n.depth + 1, onErr: onErr, b: n.literal}
	return append(prog, inst)
}

func (n *node) compileVariable(onErr jumpPointer, prog []instruction) []instruction {
	inst := &instMatchVariable{
		names: n.names, pattern: n.pattern, min: n.min, max: n.max,
		frameCount: n.depth + 1, onErr: onErr}
	return append(prog, inst)
}

type sizeSortedNodes []node

func (s sizeSortedNodes) Len() int           { return len(s) }
func (s sizeSortedNodes) Less(i, j int) bool { return s[i].size > s[j].size }
func (s sizeSortedNodes) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
