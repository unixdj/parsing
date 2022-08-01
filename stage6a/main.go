//go:generate goyacc parse.y

package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/mattn/go-isatty"
)

var ErrZeroDivision = errors.New("division by zero")

type fun func() (int, error)
type op func(int, int) (int, error)

func newFun(left fun, f op, right fun) fun {
	return func() (int, error) {
		a, err := left()
		if err != nil {
			return 0, err
		}
		b, err := right()
		if err != nil {
			return 0, err
		}
		return f(a, b)
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

type varMap map[string]int

func (vl varMap) Get(s string) (int, error) {
	if n, ok := vl[s]; ok {
		return n, nil
	}
	return 0, fmt.Errorf("unknown variable %s", s)
}

var runtime = struct {
	top  list
	vars varMap
	eof  bool
}{
	vars: make(varMap),
}

func cmdEOF() (int, error) {
	runtime.eof = true
	return 0, nil
}

type opMap map[byte]op

var ops = opMap{
	'+': func(a, b int) (int, error) { return a + b, nil },
	'-': func(a, b int) (int, error) { return a - b, nil },
	'*': func(a, b int) (int, error) { return a * b, nil },
	'/': func(a, b int) (int, error) {
		if b == 0 {
			return 0, ErrZeroDivision
		}
		return a / b, nil
	},
	'%': func(a, b int) (int, error) {
		if b == 0 {
			return 0, ErrZeroDivision
		}
		return a % b, nil
	},
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
	n   int
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
	if s == "" {
		return false
	}
	var (
		tok  = token{typ: 1}
		tlen = 1
	)
	const bareTokens = "!%&()*+-/;<=>^{|}"
	switch {
	case strings.Index(bareTokens, s[:1]) != -1:
		tok.typ = int(s[0])
		if op, ok := ops[s[0]]; ok {
			tok.op = op
		}
	case s[0] >= '0' && s[0] <= '9':
		for tlen < len(s) && s[tlen] >= '0' && s[tlen] <= '9' {
			tlen++
		}
		u, err := strconv.ParseUint(s[:tlen], 10, 63)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			break
		}
		tok.typ = NUM
		tok.n = int(u)
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
			if yy.sendEnd() != lexParseSuccess {
				goto reset
			}
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
	// after $end or $unk, but this simpler is more robust.
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
		for i = 0; i - 5; i = i + 1 {
			for j = -2; j; j = j + 1 {
				i; j
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
