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

func (n number) Bool() bool {
	if n.isFloat {
		return n.f != 0
	}
	return n.i != 0
}

func (a number) String() string {
	if a.isFloat {
		return strconv.FormatFloat(a.f, 'g', -1, 64)
	}
	return strconv.FormatInt(int64(a.i), 10)
}

type fun func() (number, error)

type (
	binIntFun   func(int, int) (int, error)
	binFloatFun func(float64, float64) (float64, error)
	op          struct {
		i binIntFun
		f binFloatFun
	}
)

func (o op) NewFun(left fun, right fun) fun {
	return func() (number, error) {
		a, err := left()
		if err != nil {
			return number{}, err
		}
		b, err := right()
		if err != nil {
			return number{}, err
		}
		switch {
		case !a.isFloat && !b.isFloat:
			a.i, err = o.i(a.i, b.i)
			return a, err
		case !a.isFloat:
			a = number{f: float64(a.i), isFloat: true}
		case !b.isFloat:
			b = number{f: float64(b.i), isFloat: true}
		}
		a.f, err = o.f(a.f, b.f)
		return a, err
	}
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

type varMap map[string]number

func (vl varMap) Get(s string) (number, error) {
	if n, ok := vl[s]; ok {
		return n, nil
	}
	return number{}, fmt.Errorf("unknown variable %s", s)
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

type opMap map[byte]op

var ops = opMap{
	'+': {
		func(a, b int) (int, error) { return a + b, nil },
		func(a, b float64) (float64, error) { return a + b, nil },
	},
	'-': {
		func(a, b int) (int, error) { return a - b, nil },
		func(a, b float64) (float64, error) { return a - b, nil },
	},
	'*': {
		func(a, b int) (int, error) { return a * b, nil },
		func(a, b float64) (float64, error) { return a * b, nil },
	},
	'/': {
		func(a, b int) (int, error) {
			if b == 0 {
				return 0, ErrZeroDivision
			}
			return a / b, nil
		},
		func(a, b float64) (float64, error) {
			if b == 0 {
				return 0, ErrZeroDivision
			}
			return a / b, nil
		},
	},
	'%': {
		func(a, b int) (int, error) {
			if b == 0 {
				return 0, ErrZeroDivision
			}
			return a % b, nil
		},
		func(a, b float64) (float64, error) {
			if b == 0 {
				return 0, ErrZeroDivision
			}
			return math.Mod(a, b), nil
		},
	},
}

type token struct {
	typ int
	s   string
	n   number
	op  op
	fun fun
}

type yyLex struct {
	r    io.Reader     // input
	tty  bool          // interactive session with a human at a teletype
	in   chan string   // channel for input lines
	c    chan token    // channel for tokens sent to the parser
	done chan struct{} // channel for parser done signal
	s    string        // input string
	next token         // next token to send
	last token         // last token sent
}

func newLexer(r io.Reader) *yyLex {
	yy := yyLex{
		r:    r,
		in:   make(chan string),
		c:    make(chan token),
		done: make(chan struct{}),
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

func (yy *yyLex) sendToken() bool {
	select {
	case <-yy.done:
		return false
	case yy.c <- yy.next:
		yy.last = yy.next
		return true
	}
}

func (yy *yyLex) send(tok token) bool {
	yy.next = tok
	return yy.sendToken()
}

// sendEnd sends an $end token and waits for parser done signal.
func (yy *yyLex) sendEnd() {
	if yy.send(token{}) {
		<-yy.done
	}
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

func (yy *yyLex) getLine() bool {
	select {
	case <-yy.done:
		return false
	case yy.s = <-yy.in:
		return true
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
		tok.typ = int(s[0])
		tok.op, _ = ops[s[0]]
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
		if !yy.getLine() {
			goto reset
		} else if yy.s == "" {
			break
		}
		first = true
		for yy.nextToken() {
			for !yy.sendToken() {
				/*
				 * when sending the first token in an input
				 * line fails, it means the error is on the
				 * previous line.  if in an interactive
				 * session, reset depth and try sending again.
				 */
				if first && yy.tty {
					depth = 0
					continue
				}
				// otherwise reset (skip line or bail out)
				goto reset
			}
			switch yy.last.typ {
			case 0, 1:
				// sent $end or $unk: wait for done and reset
				<-yy.done
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
			if !yy.send(token{typ: ';'}) {
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
	// after $end or $unk, but this is simpler is more robust.
	yy.sendEnd()                          // send $end
	yy.send(token{typ: CMD, fun: cmdEOF}) // send EOF command
	yy.sendEnd()                          // send $end
}

func (yy *yyLex) parse() {
	go yy.input()
	go yy.run()
	for !runtime.eof {
		if yyParse(yy) == 0 {
			if err := runtime.top.Run(); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
		yy.done <- struct{}{}
	}
}

func main() {
	yyErrorVerbose = true
	if false {
		s := `
		for i = 0; i - 5; i = i + 1 {
			for j = -2; j; j = j + 1 {
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
