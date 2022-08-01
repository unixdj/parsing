//go:generate goyacc parse.y

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

type token struct {
	typ int
	s   string
}

func (tok token) String() string {
	var tt string
	switch tok.typ {
	case 0:
		tt = "$end"
	case 1:
		tt = "$unk"
	case NUM:
		tt = "NUM"
	case IDENT:
		tt = "IDENT"
	default:
		tt = string(rune(tok.typ))
	}
	return fmt.Sprintf("%s %q", tt, tok.s)
}

type yyLex struct {
	c chan token
}

func (yy *yyLex) Error(s string) {
	fmt.Fprintln(os.Stderr, s)
}

func (yy *yyLex) Lex(yylval *yySymType) int {
	tok := <-yy.c
	fmt.Println("token:", tok)
	return tok.typ
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
		tok.typ = NUM
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
			a = 4+5/6`)
		go yy.run(in)
		fmt.Println("parser returned", yyParse(&yy))
		return
	}
	go yy.run(os.Stdin)
	yyParse(&yy)
}
