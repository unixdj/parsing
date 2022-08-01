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
%token <word> IDENT

%type <num> num
%type <fun> stmt assign expr expr2 expr3 expr4 var
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
        assign
|       expr
        {
                a := $1
                $$ = func() int {
                        fmt.Println(a())
                        return 0
                }
        }

assign:
        IDENT '=' expr
        {
                s, a := $1, $3
                $$ = func() int {
                        runtime.vars[s] = a()
                        return 0
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
                        x, y := a(), b()
                        if y == 0 {
                                fmt.Fprintln(os.Stderr, "division by zero")
                                return 0
                        }
                        return x / y
                }
        }
|       expr2 '%' expr3
        {
                a, b := $1, $3
                $$ = func() int {
                        x, y := a(), b()
                        if y == 0 {
                                fmt.Fprintln(os.Stderr, "division by zero")
                                return 0
                        }
                        return x % y
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
