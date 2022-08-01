%{
package main

import (
        "fmt"
        "os"
)
%}

%union {
        num  int
        word string
        fun  func() int
        list list
}

%token <num> NUM
%token <word> IDENT FOR

%type <num> num
%type <fun> stmt stmt2 block forloop assign expr expr2 expr3 expr4 var
%type <list> top stmts

%%

top:
        stmts
        {
                runtime.top = $1
        }

stmts:
        {
        }
|       stmts ';'
|       stmts stmt ';'
        {
                $$ = append($1, $2)
        }

stmt:
        stmt2
|       block
|       forloop

stmt2:
        assign
|       expr
        {
                a := $1
                $$ = func() int {
                        fmt.Println(a())
                        return 0
                }
        }

forloop:
        FOR expr block
        {
                a, b := $2, $3
                $$ = func() int {
                        for a() != 0 {
                                b()
                        }
                        return 0
                }
        }
|       FOR stmt2 ';' expr ';' stmt2 block
        {
                a, b, c, d := $2, $4, $6, $7
                $$ = func() int {
                        for a(); b() != 0; c() {
                                d()
                        }
                        return 0
                }
        }

block:
        '{' stmts '}'
        {
                a := $2
                $$ = func() int {
                        a.Run()
                        return 0
                }
        }

assign:
        IDENT '=' expr
        {
                s, a := $1, $3
                $$ = func() int {
                        v := a()
                        runtime.vars[s] = v
                        return v
                }
        }

expr:
        expr2
|       expr '+' expr2
        {
                a, b := $1, $3
                $$ = func() int {
                        return a() + b()
                }
        }
|       expr '-' expr2
        {
                a, b := $1, $3
                $$ = func() int {
                        return a() - b()
                }
        }
expr2:
        expr3
|       expr2 '*' expr3
        {
                a, b := $1, $3
                $$ = func() int {
                        return a() * b()
                }
        }
|       expr2 '/' expr3
        {
                a, b := $1, $3
                $$ = func() int {
                        if c := b(); c != 0 {
                                return a() / c
                        }
                        fmt.Fprintln(os.Stderr, "division by zero")
                        return 0
                }
        }
|       expr2 '%' expr3
        {
                a, b := $1, $3
                $$ = func() int {
                        if c := b(); c != 0 {
                                return a() % c
                        }
                        fmt.Fprintln(os.Stderr, "division by zero")
                        return 0
                }
        }

expr3:
        num
        {
                n := $1
                $$ = func() int {
                        return n
                }
        }
|       expr4

num:
        NUM
|       '-' NUM
        {
                $$ = -$2
        }

expr4:
        var
|       '-' expr4
        {
                a := $2
                $$ = func() int {
                        return -a()
                }
        }
|       '(' expr ')'
        {
                $$ = $2
        }

var:
        IDENT
        {
                s := $1
                $$ = func() int {
                        return runtime.vars.Get(s)
                }
        }

%%
