//go:generate goyacc parse.y

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type list []func() int

func (l list) Run() {
	for _, v := range l {
		v()
	}
}

type varMap map[string]int

func (vl varMap) Get(s string) int {
	if n, ok := vl[s]; ok {
		return n
	}
	fmt.Fprintln(os.Stderr, "unknown variable", s)
	return 0
}

var runtime = struct {
	top  list
	vars varMap
}{
	vars: make(varMap),
}

type token struct {
	typ int
	s   string
	n   int
}

type yyLex struct {
	c    chan token
	last token
}

func (yy *yyLex) Lex(yylval *yySymType) int {
	tok := <-yy.c
	yy.last = tok
	switch tok.typ {
	case NUM:
		yylval.num = tok.n
	case IDENT:
		yylval.word = tok.s
	}
	return tok.typ
}

func (yy *yyLex) Error(s string) {
	fmt.Fprintln(os.Stderr, s)
	fmt.Fprintln(os.Stderr, "last token:", yy.last)
}

func (yy *yyLex) sendToken(tok token) {
	yy.c <- tok
}

func (yy *yyLex) nextToken(s string) (token, string) {
	var (
		tok  = token{typ: 1}
		tlen = 1
	)
	const bareTokens = "=+-*/%();"
	switch {
	case strings.Index(bareTokens, s[:1]) != -1:
		tok.typ = int(s[0])
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
		tok.typ = IDENT
	}
	tok.s, s = s[:tlen], s[tlen:]
	return tok, strings.TrimSpace(s)
}

func (yy *yyLex) run(in io.Reader) {
	var tok token
	sc := bufio.NewScanner(in)
	for sc.Scan() {
		s := strings.TrimSpace(sc.Text())
		for s != "" {
			tok, s = yy.nextToken(s)
			yy.sendToken(tok)
			if tok.typ == 1 {
				return
			}
		}
		yy.sendToken(token{typ: ';'})
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	yy.sendToken(token{})
}

func main() {
	yyErrorVerbose = true
	yy := yyLex{
		c: make(chan token),
	}
	if false {
		in := bytes.NewBufferString(`1 + 2 + 3
			a = 4+5/6
			a`)
		go yy.run(in)
		fmt.Println("parser returned", yyParse(&yy))
		runtime.top.Run()
		return
	}
	go yy.run(os.Stdin)
	yyParse(&yy)
	runtime.top.Run()
}
