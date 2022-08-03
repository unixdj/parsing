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

type tree struct {
	typ   int
	n     int
	s     string
	left  *tree
	right *tree
}

type list []*tree

var top list

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
	case strings.IndexByte(bareTokens, s[0]) != -1:
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

func (t *tree) print(pref []byte, last bool) {
	var angle, wall byte = '+', '|'
	if last {
		angle, wall = '`', ' '
	}
	fmt.Printf("%s|\n%s%c---", pref, pref, angle)
	switch t.typ {
	case NUM:
		fmt.Println("--> num", t.n)
	case IDENT:
		fmt.Println("--> ident", t.s)
	default:
		fmt.Println("+-> op", string(t.typ))
		pref = append(pref, wall, ' ', ' ', ' ')
		if t.left != nil {
			t.left.print(pref, false)
		}
		if t.right != nil {
			t.right.print(pref, true)
		}
	}
}

func (l list) print() {
	fmt.Println("×≡≡ top")
	if len(l) == 0 {
		return
	}
	pref := make([]byte, 0, 64)
	pref = append(pref, []byte("|   ")...)
	for k, v := range l {
		if k == len(l)-1 {
			fmt.Printf("|\n`---+-> stmt[%d]\n", k)
			pref[0] = ' '
		} else {
			fmt.Printf("|\n+---+-> stmt[%d]\n", k)
		}
		v.print(pref, true)
	}
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
		top.print()
		return
	}
	go yy.run(os.Stdin)
	yyParse(&yy)
	top.print()
}
