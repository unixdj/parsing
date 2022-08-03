//go:generate goyacc parse.y

package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/mattn/go-isatty"
)

var ErrZeroDivision = errors.New("division by zero")

type number struct {
	i       int
	f       float64
	isFloat bool
}

func boolToNumber(b bool) number {
	if b {
		return number{i: 1}
	}
	return number{}
}

func (a number) Bool() bool {
	if a.isFloat {
		return a.f != 0
	}
	return a.i != 0
}

func (a number) Int() int {
	if a.isFloat {
		return int(a.f)
	}
	return a.i
}

func (a number) String() string {
	if a.isFloat {
		return strconv.FormatFloat(a.f, 'g', -1, 64)
	}
	return strconv.FormatInt(int64(a.i), 10)
}

func (a number) NewFun() fun {
	return func() (number, error) {
		return a, nil
	}
}

func (a number) RunUnary(f op) number {
	if m, ok := f.(multiOp); ok {
		f = m.un
	}
	return f.(unOp)(a)
}

type fun func() (number, error)

func (f fun) Denominator() fun {
	return func() (number, error) {
		n, err := f()
		if err == nil && !n.Bool() {
			err = ErrZeroDivision
		}
		return n, err
	}
}

type op interface {
	NewFun(fun, fun) fun
}

type (
	unOp        func(number) number
	binOp       func(number, number) number
	unIntFun    func(int) int
	unFloatFun  func(float64) float64
	binIntFun   func(int, int) int
	binFloatFun func(float64, float64) float64
)

func (f unOp) NewFun(left, right fun) fun {
	return func() (number, error) {
		a, err := left()
		if err != nil {
			return number{}, err
		}
		return f(a), nil
	}
}

func newUnIntOp(f unIntFun) unOp {
	return func(a number) number {
		return number{i: f(a.Int())}
	}
}

func newUnOp(uif unIntFun, uff unFloatFun) unOp {
	return func(a number) number {
		if a.isFloat {
			a.f = uff(a.f)
		} else {
			a.i = uif(a.i)
		}
		return a
	}
}

func (f binOp) NewFun(left, right fun) fun {
	return func() (number, error) {
		a, err := left()
		if err != nil {
			return number{}, err
		}
		b, err := right()
		if err != nil {
			return number{}, err
		}
		return f(a, b), nil
	}
}

func newBinIntOp(f binIntFun) binOp {
	return func(a, b number) number {
		return number{i: f(a.Int(), b.Int())}
	}
}

func castToSame(f binOp) binOp {
	return func(a, b number) number {
		if a.isFloat != b.isFloat {
			if !a.isFloat {
				a = number{f: float64(a.i), isFloat: true}
			} else {
				b = number{f: float64(b.i), isFloat: true}
			}
		}
		return f(a, b)
	}
}

func newBinOp(bif binIntFun, bff binFloatFun) binOp {
	return castToSame(func(a, b number) number {
		if a.isFloat {
			a.f = bff(a.f, b.f)
		} else {
			a.i = bif(a.i, b.i)
		}
		return a
	})
}

type divModOp binOp

func newDivModOp(bif binIntFun, bff binFloatFun) divModOp {
	return divModOp(newBinOp(bif, bff))
}

func (f divModOp) NewFun(left, right fun) fun {
	return binOp(f).NewFun(left, right.Denominator())
}

type multiOp struct {
	un, bin op
}

func (f multiOp) NewFun(left, right fun) fun {
	if right == nil {
		return f.un.NewFun(left, nil)
	}
	return f.bin.NewFun(left, right)
}

var (
	equalOp = castToSame(func(a, b number) number {
		if a.isFloat {
			return boolToNumber(a.f == b.f)
		}
		return boolToNumber(a.i == b.i)
	})
	lessOp = castToSame(func(a, b number) number {
		if a.isFloat {
			return boolToNumber(a.f < b.f)
		}
		return boolToNumber(a.i < b.i)
	})
	greaterOp binOp = func(a, b number) number {
		return lessOp(b, a)
	}
)

type compareOp uint8

const (
	Equal = compareOp(1 << iota)
	Less
	Greater
)

func (f compareOp) BinOp() binOp {
	var (
		bf  binOp
		not bool
	)
	if (f & (f - 1)) != 0 {
		f ^= Equal | Less | Greater
		not = true
	}
	switch f {
	case Equal:
		bf = equalOp
	case Less:
		bf = lessOp
	case Greater:
		bf = greaterOp
	}
	if not {
		return func(a, b number) number {
			ans := bf(a, b)
			ans.i ^= 1
			return ans
		}
	}
	return bf
}

func (f compareOp) NewFun(left, right fun) fun {
	return f.BinOp().NewFun(left, right)
}

type logicOp bool

const (
	logicalOr  = logicOp(false)
	logicalAnd = logicOp(true)
)

func (cont logicOp) NewFun(left, right fun) fun {
	return func() (number, error) {
		a, err := left()
		if err != nil {
			return number{}, err
		}
		ans := a.Bool()
		if ans == bool(cont) {
			a, err = right()
			if err != nil {
				return number{}, err
			}
			ans = a.Bool()
		}
		return boolToNumber(ans), nil
	}
}

type forLoop struct{}

func (forLoop) NewFun(expr, block fun) fun {
	return func() (number, error) {
		for {
			if v, err := expr(); err != nil || !v.Bool() {
				return number{}, err
			}
			if _, err := block(); err != nil {
				return number{}, err
			}
		}
	}
}

func NewAssign(lval string, op op, rval fun) fun {
	/*
	 * Possibilities:
	 * op == nil:    '=' operator, rval returns the value
	 * rval == nil:  "++" or "--", unOp.NewFun() ignores the right operand
	 * both non-nil: operator like "+=", uses Get(lval) and rval
	 */
	if op != nil {
		rval = op.NewFun(runtime.vars.NewGet(lval), rval)
	}
	return runtime.vars.NewSet(lval, rval)
}

var (
	addOp = newBinOp(
		func(a, b int) int { return a + b },
		func(a, b float64) float64 { return a + b },
	)
	subOp = multiOp{
		newUnOp(
			func(a int) int { return -a },
			func(a float64) float64 { return -a },
		),
		newBinOp(
			func(a, b int) int { return a - b },
			func(a, b float64) float64 { return a - b },
		),
	}
	mulOp = newBinOp(
		func(a, b int) int { return a * b },
		func(a, b float64) float64 { return a * b },
	)
	divOp = newDivModOp(
		func(a, b int) int { return a / b },
		func(a, b float64) float64 { return a / b },
	)
	modOp = newDivModOp(
		func(a, b int) int { return a % b },
		func(a, b float64) float64 { return math.Mod(a, b) },
	)
	lShiftOp = newBinIntOp(func(a, b int) int { return a << b })
	rShiftOp = newBinIntOp(func(a, b int) int { return a >> b })
	andOp    = newBinIntOp(func(a, b int) int { return a & b })
	bicOp    = newBinIntOp(func(a, b int) int { return a &^ b })
	orOp     = newBinIntOp(func(a, b int) int { return a | b })
	xorOp    = multiOp{
		newUnIntOp(func(a int) int { return ^a }),
		newBinIntOp(func(a, b int) int { return a ^ b }),
	}
	incOp = newUnOp(
		func(a int) int { return a + 1 },
		func(a float64) float64 { return a + 1 },
	)
	decOp = newUnOp(
		func(a int) int { return a - 1 },
		func(a float64) float64 { return a - 1 },
	)
	notOp   unOp = func(a number) number { return boolToNumber(!a.Bool()) }
	printOp unOp = func(a number) number { fmt.Println(a); return a }
)

type opMap map[string]struct {
	typ int
	op  op
}

func (m opMap) find(s string) (token, int) {
	tlen := len(s)
	if tlen > 3 {
		tlen = 3
	}
	for tlen > 0 {
		if o, ok := m[s[:tlen]]; ok {
			return token{typ: o.typ, op: o.op}, tlen
		}
		tlen--
	}
	return token{typ: int(s[0])}, 1
}

var ops = opMap{
	"+":   {'+', addOp},
	"-":   {'-', subOp},
	"*":   {'*', mulOp},
	"/":   {'/', divOp},
	"%":   {'%', modOp},
	"&":   {'&', andOp},
	"^":   {'^', xorOp},
	"&^":  {BIC, bicOp},
	"|":   {'|', orOp},
	"<<":  {LSHIFT, lShiftOp},
	">>":  {RSHIFT, rShiftOp},
	"!":   {'!', notOp},
	"<":   {'<', Less},
	">":   {'>', Greater},
	"<=":  {LE, Less | Equal},
	">=":  {GE, Greater | Equal},
	"==":  {EQ, Equal},
	"!=":  {NE, Less | Greater},
	"&&":  {LAND, logicalAnd},
	"||":  {LOR, logicalOr},
	"+=":  {ADDEQ, addOp},
	"-=":  {SUBEQ, subOp},
	"*=":  {MULEQ, mulOp},
	"/=":  {DIVEQ, divOp},
	"%=":  {MODEQ, modOp},
	"&=":  {ANDEQ, andOp},
	"^=":  {XOREQ, xorOp},
	"&^=": {BICEQ, bicOp},
	"|=":  {OREQ, orOp},
	"<<=": {LSHIFTEQ, lShiftOp},
	">>=": {RSHIFTEQ, rShiftOp},
	"++":  {INC, incOp},
	"--":  {DEC, decOp},
}

type list []fun

func (l list) Run() error {
	for _, v := range l {
		if _, err := v(); err != nil {
			return err
		}
	}
	return nil
}

func (l list) NewFun() fun {
	return func() (number, error) {
		return number{}, l.Run()
	}
}

type varMap map[string]number

func (vl varMap) NewGet(s string) fun {
	return func() (number, error) {
		if n, ok := vl[s]; ok {
			return n, nil
		}
		return number{}, fmt.Errorf("unknown variable %s", s)
	}
}

func (vl varMap) NewSet(s string, f fun) fun {
	return func() (number, error) {
		n, err := f()
		if err != nil {
			return number{}, err
		}
		vl[s] = n
		return n, nil
	}
}

var runtime = struct {
	top  list
	vars varMap
	eof  bool
}{
	vars: make(varMap),
}

func cmdEOF() (number, error) {
	runtime.eof = true
	return number{}, nil
}

const (
	lexOK           = iota                   // operation completed
	lexParseSuccess = iota | lexParserStatus // parser signalled success
	lexParseError                            // parser signalled failure

	lexParserStatus = 0x02 // flag for parser status
)

type token struct {
	typ int
	s   string
	n   number
	op  op
	fun fun
}

type yyLex struct {
	r    io.Reader   // input
	tty  bool        // interactive session with a human at a teletype
	in   chan string // channel for input lines
	c    chan token  // channel for tokens sent to the parser
	ps   chan int    // channel for parser status
	s    string      // input string
	next token       // next token to send
	last token       // last token sent
}

func newLexer(r io.Reader) *yyLex {
	yy := yyLex{
		r:  r,
		c:  make(chan token),
		in: make(chan string),
		ps: make(chan int),
	}
	if f, ok := r.(*os.File); ok {
		yy.tty = isatty.IsTerminal(f.Fd())
	}
	return &yy
}

func (yy *yyLex) Lex(yylval *yySymType) int {
	tok := <-yy.c
	switch tok.typ {
	case NUM:
		yylval.num = tok.n
	case IDENT:
		yylval.word = tok.s
	case CMD:
		yylval.fun = tok.fun
	default:
		yylval.op = tok.op
	}
	return tok.typ
}

func (yy *yyLex) Error(s string) {
	fmt.Fprintln(os.Stderr, s)
}

func (yy *yyLex) sendToken() int {
	select {
	case status := <-yy.ps:
		return status
	case yy.c <- yy.next:
		yy.last = yy.next
		return lexOK
	}
}

func (yy *yyLex) send(tok token) int {
	yy.next = tok
	return yy.sendToken()
}

// sendEnd sends an $end token and waits for parser status.
func (yy *yyLex) sendEnd() int {
	if status := yy.send(token{}); status != lexOK {
		return status
	}
	return <-yy.ps
}

func (yy *yyLex) input() {
	sc := bufio.NewScanner(yy.r)
	for sc.Scan() {
		s := sc.Text()
		if s == "" {
			s = " "
		}
		yy.in <- s
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	yy.in <- ""
}

func (yy *yyLex) getLine() int {
	select {
	case status := <-yy.ps:
		return status
	case yy.s = <-yy.in:
		return lexOK
	}
}

func (yy *yyLex) nextToken() bool {
	s := strings.TrimSpace(yy.s)
	if s == "" || s[0] == '#' {
		return false
	}
	var (
		tok  = token{typ: 1}
		tlen = 1
	)
	const bareTokens = "!%&()*+-/;<=>^{|}"
	switch {
	case strings.IndexByte(bareTokens, s[0]) != -1:
		tok, tlen = ops.find(s)
	case s[0] >= '0' && s[0] <= '9':
		for tlen < len(s) &&
			(s[tlen] >= '0' && s[tlen] <= '9' || s[tlen] == '.') {
			tlen++
		}
		if u, err := strconv.ParseUint(s[:tlen], 10, 63); err == nil {
			tok.typ = NUM
			tok.n.i = int(u)
			break
		}
		if f, err := strconv.ParseFloat(s[:tlen], 64); err == nil {
			tok.typ = NUM
			tok.n.f = f
			tok.n.isFloat = true
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
	case s[0] >= 'a' && s[0] <= 'z':
		for tlen < len(s) && s[tlen] >= 'a' && s[tlen] <= 'z' {
			tlen++
		}
		switch s[:tlen] {
		case "for":
			tok.typ = FOR
			tok.op = forLoop{}
		default:
			tok.typ = IDENT
		}
	}
	tok.s, yy.s = s[:tlen], s[tlen:]
	yy.next = tok
	return true
}

func (yy *yyLex) run() {
	var (
		depth int
		first bool
	)
	for {
		if yy.getLine() != lexOK {
			goto reset
		} else if yy.s == "" {
			break
		}
		first = true
		for yy.nextToken() {
			for yy.sendToken() != lexOK {
				/*
				 * when sending the first token in an input
				 * line fails, it means the error is on the
				 * previous line.  if in an interactive
				 * session, reset depth and try sending again.
				 */
				if yy.tty && first {
					depth = 0
					continue
				}
				// otherwise reset (skip line or bail out)
				goto reset
			}
			switch yy.last.typ {
			case 0, 1:
				// sent $end or $unk: wait for status and reset
				<-yy.ps
				goto reset
			case '{':
				depth++
			case '}':
				depth--
			}
			first = false
		}
		// end of line
		switch yy.last.typ {
		case 0, 1:
			// if we haven't sent any tokens, read next line
			continue
		case ';':
			// no semicolon needed
		default:
			// inject semicolon at EOL
			if yy.send(token{typ: ';'}) != lexOK {
				goto reset
			}
		}
		if yy.tty && depth <= 0 {
			// interactive and not within a block:
			// send $end and reset depth
			yy.sendEnd()
			depth = 0
		}
		continue
	reset:
		if !yy.tty {
			break
		}
		depth = 0
	}
	// EOF
	// we could check yy.last here to avoid sending $end
	// after $end or $unk, but this is simpler and more robust.
	yy.sendEnd()                          // send $end
	yy.send(token{typ: CMD, fun: cmdEOF}) // send EOF command
	yy.sendEnd()                          // send $end
}

func (yy *yyLex) parse() {
	go yy.input()
	go yy.run()
	for !runtime.eof {
		status := yyParse(yy)
		if status == 0 {
			if err := runtime.top.Run(); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
		yy.ps <- status | lexParserStatus
	}
}

func main() {
	yyErrorVerbose = true
	if false {
		s := `
		for i = 0; i < 5; i++ {
			for j = -2; j != 0; j++ {
				i; j / 2.0
			}
		}
		`
		yy := newLexer(bytes.NewBufferString(s))
		yy.parse()
		return
	}
	yy := newLexer(os.Stdin)
	yy.parse()
}
