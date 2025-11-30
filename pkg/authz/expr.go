package authz

// Expression interpreter for advanced policy conditions (Task 6).
// Grammar (simplified):
//   Expr        := OrExpr
//   OrExpr      := AndExpr { '||' AndExpr }
//   AndExpr     := UnaryExpr { '&&' UnaryExpr }
//   UnaryExpr   := '!' UnaryExpr | Primary
//   Primary     := Comparison | '(' Expr ')'
//   Comparison  := Value ( (== | != | > | >= | < | <=) Value | InList )?
//   InList      := 'in' '[' ValueList ']'  // membership test
//   ValueList   := Value { ',' Value }
//   Value       := IDENT | STRING | NUMBER | BOOLEAN
// Identifiers: subject, resource, action, any context key (direct) or ctx.<key>.
// Strings use double quotes; numbers are decimal; booleans: true/false.
// Whitespace ignored. Errors fail closed (non-match).

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ExprLimits defines resource limits for parsing/evaluation.
type ExprLimits struct {
	MaxTokens           int
	MaxDepth            int
	MaxOps              int
	MaxIdentifierLength int
	MaxLiteralLength    int
}

// DefaultExprLimits fallback values.
var DefaultExprLimits = ExprLimits{MaxTokens: 256, MaxDepth: 32, MaxOps: 1024, MaxIdentifierLength: 64, MaxLiteralLength: 256}


// Boolean string constants for token comparison
const (
boolTrueString  = "true"
boolFalseString = "false"
)
// token types
const (
	tokEOF = iota
	tokIdent
	tokString
	tokNumber
	tokBool
	tokAnd
	tokOr
	tokNot
	tokLParen
	tokRParen
	tokLBracket
	tokRBracket
	tokComma
	tokEq
	tokNeq
	tokGT
	tokGTE
	tokLT
	tokLTE
	tokIn
)

type token struct {
	typ int
	lit string
}

// lexer
type lexer struct {
	src    string
	pos    int
	tokens []token
	limits ExprLimits
}

func lex(src string, limits ExprLimits) ([]token, error) {
	l := &lexer{src: src, limits: limits}
	for {
		l.skipWS()
		if l.pos >= len(l.src) {
			l.tokens = append(l.tokens, token{typ: tokEOF})
			break
		}
		r := l.src[l.pos]
		oldPos := l.pos // Track position before trying operators
		// Try multi-char operators first
		if err := l.tryMultiCharOps(r); err != nil {
			return nil, err
		} else if l.pos > oldPos {
			// Position advanced, token was consumed
			continue
		}
		// Try single-char tokens
		if err := l.trySingleCharTokens(r); err != nil {
			return nil, err
		}
		switch {
		case l.pos > oldPos:
			// Token was consumed in switch
		case isIdentStart(r):
			// Identifier or keyword
			if err := l.readIdentOrKeyword(); err != nil {
				return nil, err
			}
		case unicode.IsDigit(rune(r)):
			// Number
			if err := l.readNum(); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unexpected character: %c", r)
		}
		if len(l.tokens) > l.limits.MaxTokens {
			return nil, errors.New("token limit exceeded")
		}
	}
	return l.tokens, nil
}

// tryMultiCharOps handles && and ||
func (l *lexer) tryMultiCharOps(r byte) error {
	if r == '&' && l.peek("&&") {
		l.pos += 2
		l.add(tokAnd, "&&")
	} else if r == '|' && l.peek("||") {
		l.pos += 2
		l.add(tokOr, "||")
	}
	return nil
}

// trySingleCharTokens handles single-char and some two-char tokens
func (l *lexer) trySingleCharTokens(r byte) error {
	oldPos := l.pos
	switch r {
	case '!':
		// not or !=
		if l.pos+1 < len(l.src) && l.src[l.pos+1] == '=' {
			l.pos += 2
			l.add(tokNeq, "!=")
		} else {
			l.pos++
			l.add(tokNot, "!")
		}
	case '(':
		l.pos++
		l.add(tokLParen, "(")
	case ')':
		l.pos++
		l.add(tokRParen, ")")
	case '[':
		l.pos++
		l.add(tokLBracket, "[")
	case ']':
		l.pos++
		l.add(tokRBracket, "]")
	case ',':
		l.pos++
		l.add(tokComma, ",")
	case '=':
		// equality '=='
		if l.pos+1 < len(l.src) && l.src[l.pos+1] == '=' {
			l.pos += 2
			l.add(tokEq, "==")
		} else {
			return fmt.Errorf("unexpected '='; use '=='")
		}
	case '>':
		l.pos++
		if l.match('=') {
			l.add(tokGTE, ">=")
		} else {
			l.add(tokGT, ">")
		}
	case '<':
		l.pos++
		if l.match('=') {
			l.add(tokLTE, "<=")
		} else {
			l.add(tokLT, "<")
		}
	case '"':
		str, err := l.readString()
		if err != nil {
			return err
		}
		if len(str) > l.limits.MaxLiteralLength {
			return errors.New("string literal length limit exceeded")
		}
		l.add(tokString, str)
	}
	// Check if we consumed anything
	if l.pos == oldPos {
		// No single-char match, caller should try identifier or number
		return nil
	}
	return nil
}

// readIdentOrKeyword reads an identifier or keyword
func (l *lexer) readIdentOrKeyword() error {
	id := l.readIdent()
	if len(id) > l.limits.MaxIdentifierLength {
		return errors.New("identifier length limit exceeded")
	}
	low := strings.ToLower(id)
	switch low {
	case boolTrueString, boolFalseString:
		l.add(tokBool, low)
	case "in":
		l.add(tokIn, low)
	default:
		l.add(tokIdent, id)
	}
	return nil
}

// readNum reads a number literal
func (l *lexer) readNum() error {
	num := l.readNumber()
	if len(num) > l.limits.MaxLiteralLength {
		return errors.New("number literal length limit exceeded")
	}
	l.add(tokNumber, num)
	return nil
}

func (l *lexer) add(t int, lit string) { l.tokens = append(l.tokens, token{typ: t, lit: lit}) }
func (l *lexer) skipWS() {
	for l.pos < len(l.src) {
		if l.src[l.pos] == ' ' || l.src[l.pos] == '\n' || l.src[l.pos] == '\t' || l.src[l.pos] == '\r' {
			l.pos++
		} else {
			break
		}
	}
}
func (l *lexer) peek(two string) bool {
	if l.pos+len(two) <= len(l.src) && l.src[l.pos:l.pos+len(two)] == two {
		return true
	}
	return false
}
func (l *lexer) match(ch byte) bool {
	if l.pos < len(l.src) && l.src[l.pos] == ch {
		l.pos++
		return true
	}
	return false
}
func isIdentStart(b byte) bool { return unicode.IsLetter(rune(b)) || b == '_' }
func isIdentPart(b byte) bool  { return isIdentStart(b) || unicode.IsDigit(rune(b)) || b == '.' }
func (l *lexer) readIdent() string {
	start := l.pos
	for l.pos < len(l.src) && isIdentPart(l.src[l.pos]) {
		l.pos++
	}
	return l.src[start:l.pos]
}
func (l *lexer) readNumber() string {
	start := l.pos
	for l.pos < len(l.src) && (unicode.IsDigit(rune(l.src[l.pos])) || l.src[l.pos] == '.') {
		l.pos++
	}
	return l.src[start:l.pos]
}
func (l *lexer) readString() (string, error) {
	l.pos++
	start := l.pos
	for l.pos < len(l.src) {
		if l.src[l.pos] == '"' {
			s := l.src[start:l.pos]
			l.pos++
			return s, nil
		}
		l.pos++
	}
	return "", errors.New("unterminated string")
}

// AST nodes

type node interface {
	eval(ctx map[string]interface{}, limits ExprLimits, ops *int, depth int) (interface{}, error)
}

type binaryNode struct {
	op          int
	left, right node
}

func (b *binaryNode) eval(ctx map[string]interface{}, limits ExprLimits, ops *int, depth int) (interface{}, error) {
	if depth > limits.MaxDepth {
		return nil, errors.New("max depth exceeded")
	}
	*ops++
	if *ops > limits.MaxOps {
		return nil, errors.New("max ops exceeded")
	}
	// logical short-circuit for AND/OR
	if b.op == tokAnd {
		lv, err := b.left.eval(ctx, limits, ops, depth+1)
		if err != nil {
			return nil, err
		}
		lb, ok := lv.(bool)
		if !ok {
			return nil, errors.New("left operand not boolean")
		}
		if !lb {
			return false, nil
		}
		rv, err := b.right.eval(ctx, limits, ops, depth+1)
		if err != nil {
			return nil, err
		}
		rb, ok := rv.(bool)
		if !ok {
			return nil, errors.New("right operand not boolean")
		}
		return lb && rb, nil
	}
	if b.op == tokOr {
		lv, err := b.left.eval(ctx, limits, ops, depth+1)
		if err != nil {
			return nil, err
		}
		lb, ok := lv.(bool)
		if !ok {
			return nil, errors.New("left operand not boolean")
		}
		if lb {
			return true, nil
		}
		rv, err := b.right.eval(ctx, limits, ops, depth+1)
		if err != nil {
			return nil, err
		}
		rb, ok := rv.(bool)
		if !ok {
			return nil, errors.New("right operand not boolean")
		}
		return lb || rb, nil
	}
	// comparison
	lv, err := b.left.eval(ctx, limits, ops, depth+1)
	if err != nil {
		return nil, err
	}
	rv, err := b.right.eval(ctx, limits, ops, depth+1)
	if err != nil {
		return nil, err
	}
	return compareValues(b.op, lv, rv)
}

type unaryNode struct {
	op    int
	inner node
}

func (u *unaryNode) eval(ctx map[string]interface{}, limits ExprLimits, ops *int, depth int) (interface{}, error) {
	if depth > limits.MaxDepth {
		return nil, errors.New("max depth exceeded")
	}
	*ops++
	if *ops > limits.MaxOps {
		return nil, errors.New("max ops exceeded")
	}
	v, err := u.inner.eval(ctx, limits, ops, depth+1)
	if err != nil {
		return nil, err
	}
	b, ok := v.(bool)
	if !ok {
		return nil, errors.New("operand not boolean")
	}
	if u.op == tokNot {
		return !b, nil
	}
	return nil, errors.New("unknown unary op")
}

type literalNode struct{ val interface{} }

func (l *literalNode) eval(ctx map[string]interface{}, limits ExprLimits, ops *int, depth int) (interface{}, error) {
	*ops++
	if *ops > limits.MaxOps {
		return nil, errors.New("max ops exceeded")
	}
	return l.val, nil
}

type identNode struct{ name string }

func (i *identNode) eval(ctx map[string]interface{}, limits ExprLimits, ops *int, depth int) (interface{}, error) {
	*ops++
	if *ops > limits.MaxOps {
		return nil, errors.New("max ops exceeded")
	}
	// resolution: direct key, or ctx.<key>
	if strings.HasPrefix(i.name, "ctx.") {
		key := strings.TrimPrefix(i.name, "ctx.")
		if v, ok := ctx[key]; ok {
			return v, nil
		}
		return "", nil
	}
	if v, ok := ctx[i.name]; ok {
		return v, nil
	}
	return "", nil
}

type inListNode struct {
	left node
	list []node
}

func (n *inListNode) eval(ctx map[string]interface{}, limits ExprLimits, ops *int, depth int) (interface{}, error) {
	if depth > limits.MaxDepth {
		return nil, errors.New("max depth exceeded")
	}
	lv, err := n.left.eval(ctx, limits, ops, depth+1)
	if err != nil {
		return nil, err
	}
	for _, e := range n.list {
		v, err := e.eval(ctx, limits, ops, depth+1)
		if err != nil {
			return nil, err
		}
		if equalValues(lv, v) {
			return true, nil
		}
	}
	return false, nil
}

// comparison helper
func compareValues(op int, left, right interface{}) (bool, error) {
	// numeric attempt
	lf, lok := toFloat(left)
	rf, rok := toFloat(right)
	if lok && rok {
		switch op {
		case tokEq:
			return lf == rf, nil
		case tokNeq:
			return lf != rf, nil
		case tokGT:
			return lf > rf, nil
		case tokGTE:
			return lf >= rf, nil
		case tokLT:
			return lf < rf, nil
		case tokLTE:
			return lf <= rf, nil
		default:
			return false, fmt.Errorf("invalid numeric op")
		}
	}
	// fall back string compare
	ls := fmt.Sprintf("%v", left)
	rs := fmt.Sprintf("%v", right)
	switch op {
	case tokEq:
		return ls == rs, nil
	case tokNeq:
		return ls != rs, nil
	default:
		return false, fmt.Errorf("invalid string op")
	}
}

func toFloat(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, true
	case string:
		f, err := strconv.ParseFloat(x, 64)
		if err == nil {
			return f, true
		}
	}
	return 0, false
}

func equalValues(a, b interface{}) bool { return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b) }

// parser

type parser struct {
	tokens []token
	pos    int
	limits ExprLimits
}

func parse(tokens []token, limits ExprLimits) (node, error) {
	p := &parser{tokens: tokens, limits: limits}
	return p.parseExpr(0)
}

func (p *parser) current() token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return token{typ: tokEOF}
}
func (p *parser) consume() token { t := p.current(); p.pos++; return t }
func (p *parser) expect(tt int) (token, error) {
	t := p.consume()
	if t.typ != tt {
		return t, fmt.Errorf("expected token %d got %d", tt, t.typ)
	}
	return t, nil
}

func (p *parser) parseExpr(depth int) (node, error) { return p.parseOr(depth) }
func (p *parser) parseOr(depth int) (node, error) {
	n, err := p.parseAnd(depth + 1)
	if err != nil {
		return nil, err
	}
	for p.current().typ == tokOr {
		p.consume()
		r, err := p.parseAnd(depth + 1)
		if err != nil {
			return nil, err
		}
		n = &binaryNode{op: tokOr, left: n, right: r}
	}
	return n, nil
}
func (p *parser) parseAnd(depth int) (node, error) {
	n, err := p.parseUnary(depth + 1)
	if err != nil {
		return nil, err
	}
	for p.current().typ == tokAnd {
		p.consume()
		r, err := p.parseUnary(depth + 1)
		if err != nil {
			return nil, err
		}
		n = &binaryNode{op: tokAnd, left: n, right: r}
	}
	return n, nil
}
func (p *parser) parseUnary(depth int) (node, error) {
	if p.current().typ == tokNot {
		p.consume()
		inner, err := p.parseUnary(depth + 1)
		if err != nil {
			return nil, err
		}
		return &unaryNode{op: tokNot, inner: inner}, nil
	}
	return p.parsePrimary(depth + 1)
}
func (p *parser) parsePrimary(depth int) (node, error) {
	t := p.current()
	switch t.typ {
	case tokLParen:
		p.consume()
		inner, err := p.parseExpr(depth + 1)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokRParen); err != nil {
			return nil, err
		}
		return inner, nil
	case tokIdent:
		left := &identNode{name: t.lit}
		p.consume()
		return p.parseMaybeComparison(left, depth+1)
	case tokString:
		left := &literalNode{val: t.lit}
		p.consume()
		return p.parseMaybeComparison(left, depth+1)
	case tokNumber:
		left := &literalNode{val: t.lit}
		p.consume()
		return p.parseMaybeComparison(left, depth+1)
	case tokBool:
		left := &literalNode{val: t.lit == boolTrueString}
		p.consume()
		return p.parseMaybeComparison(left, depth+1)
	default:
		return nil, fmt.Errorf("unexpected token in primary: %v", t.lit)
	}
}

func (p *parser) parseMaybeComparison(left node, depth int) (node, error) {
	op := p.current().typ
	switch op {
	case tokEq, tokNeq, tokGT, tokGTE, tokLT, tokLTE:
		p.consume()
		rightTok := p.current()
		var right node
		switch rightTok.typ {
		case tokIdent:
			right = &identNode{name: rightTok.lit}
		case tokString:
			right = &literalNode{val: rightTok.lit}
		case tokNumber:
			right = &literalNode{val: rightTok.lit}
		case tokBool:
			right = &literalNode{val: rightTok.lit == boolTrueString}
		default:
			return nil, fmt.Errorf("invalid right operand")
		}
		p.consume()
		return &binaryNode{op: op, left: left, right: right}, nil
	case tokIn:
		p.consume()
		if _, err := p.expect(tokLBracket); err != nil {
			return nil, err
		}
		list := []node{}
		for p.current().typ != tokRBracket && p.current().typ != tokEOF {
			ct := p.current()
			var elem node
			switch ct.typ {
			case tokIdent:
				elem = &identNode{name: ct.lit}
			case tokString:
				elem = &literalNode{val: ct.lit}
			case tokNumber:
				elem = &literalNode{val: ct.lit}
			case tokBool:
				elem = &literalNode{val: ct.lit == boolTrueString}
			default:
				return nil, fmt.Errorf("invalid in-list element")
			}
			p.consume()
			list = append(list, elem)
			if p.current().typ == tokComma {
				p.consume()
				continue
			}
		}
		if _, err := p.expect(tokRBracket); err != nil {
			return nil, err
		}
		return &inListNode{left: left, list: list}, nil
	default:
		// no comparison; treat left value as boolean if possible
		return left, nil
	}
}

// EvaluateExpression parses and evaluates expression with optional limits (nil => defaults).
func EvaluateExpression(expr string, req Request, limits *ExprLimits) (bool, error) {
	lm := DefaultExprLimits
	if limits != nil {
		lm = *limits
	}
	// build context map
	ctx := make(map[string]interface{}, len(req.Context)+3)
	ctx["subject"] = req.Subject
	ctx["resource"] = req.Resource
	ctx["action"] = req.Action
	for k, v := range req.Context {
		ctx[k] = v
	}
	// tokenize
	toks, err := lex(expr, lm)
	if err != nil {
		return false, err
	}
	// parse
	ast, err := parse(toks, lm)
	if err != nil {
		return false, err
	}
	// evaluate root
	ops := 0
	val, err := ast.eval(ctx, lm, &ops, 0)
	if err != nil {
		return false, err
	}
	b, ok := val.(bool)
	if !ok { // attempt truthy interpretation
		if s, ok2 := val.(string); ok2 {
			return s != "" && s != "0", nil
		}
		return false, errors.New("expression did not evaluate to boolean")
	}
	return b, nil
}

// DebugLex returns tokens for diagnostic tests (non-production use).
// DebugToken exported representation of internal token for tests.
type DebugToken struct {
	Type    int
	Literal string
}

func DebugLex(expr string) []DebugToken {
	toks, _ := lex(expr, DefaultExprLimits)
	out := make([]DebugToken, 0, len(toks))
	for _, tk := range toks {
		out = append(out, DebugToken{Type: tk.typ, Literal: tk.lit})
	}
	return out
}
