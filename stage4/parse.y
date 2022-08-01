%{
package main

import "fmt"
%}

%union {
        num  int
        word string
        fun  fun
        list list
}

%token <num> NUM
%token <word> IDENT FOR

%type <num> num
%type <fun> stmt stmt2 forloop assign expr expr2 expr3 expr4 var
%type <list> top stmts block

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
|       stmts block
        {
                $$ = append($1, $2...)
        }

stmt:
        stmt2
|       forloop

stmt2:
        assign
|       expr
        {
                a := $1
                $$ = func() (int, error) {
                        x, err := a()
                        if err != nil {
                                return 0, err
                        }
                        fmt.Println(x)
                        return 0, nil
                }
        }

forloop:
        FOR expr block
        {
                a, b := $2, $3
                $$ = func() (int, error) {
                        for {
                                if v, err := a(); err != nil || v == 0 {
                                        return 0, err
                                }
                                if err := b.Run(); err != nil {
                                        return 0, err
                                }
                        }
                }
        }
|       FOR stmt2 ';' expr ';' stmt2 block
        {
                a, b, c := $2, $4, append($7, $6)
                $$ = func() (int, error) {
                        if _, err := a(); err != nil {
                                return 0, err
                        }
                        for {
                                if v, err := b(); err != nil || v == 0 {
                                        return 0, err
                                }
                                if err := c.Run(); err != nil {
                                        return 0, err
                                }
                        }
                }
        }

block:
        '{' stmts '}'
        {
                $$ = $2
        }

assign:
        IDENT '=' expr
        {
                s, a := $1, $3
                $$ = func() (int, error) {
                        v, err := a()
                        if err != nil {
                                return 0, err
                        }
                        runtime.vars[s] = v
                        return v, nil
                }
        }

expr:
        expr2
|       expr '+' expr2
        {
                $$ = newFun($1, $3, func(a, b int) (int, error) {
                        return a + b, nil
                })
        }
|       expr '-' expr2
        {
                $$ = newFun($1, $3, func(a, b int) (int, error) {
                        return a - b, nil
                })
        }
expr2:
        expr3
|       expr2 '*' expr3
        {
                $$ = newFun($1, $3, func(a, b int) (int, error) {
                        return a * b, nil
                })
        }
|       expr2 '/' expr3
        {
                $$ = newFun($1, $3, func(a, b int) (int, error) {
                        if b == 0 {
                                return 0, ErrZeroDivision
                        }
                        return a / b, nil
                })
        }
|       expr2 '%' expr3
        {
                $$ = newFun($1, $3, func(a, b int) (int, error) {
                        if b == 0 {
                                return 0, ErrZeroDivision
                        }
                        return a % b, nil
                })
        }

expr3:
        num
        {
                n := $1
                $$ = func() (int, error) {
                        return n, nil
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
                $$ = func() (int, error) {
                        x, err := a()
                        return -x, err
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
                $$ = func() (int, error) {
                        return runtime.vars.Get(s)
                }
        }

%%
