%{
package main

import "fmt"
%}

%union {
        num  number
        word string
        op   op
        fun  fun
        list list
}

%token <num> NUM
%token <word> IDENT FOR
%token <fun> CMD
%token <op> '+' '-' '*' '/' '%'

%type <num> num
%type <op> op1 op2
%type <fun> stmt stmt2 forloop assign expr expr2 expr3 expr4 var
%type <list> top stmts block

%%

top:
        stmts
        {
                runtime.top = $1
        }
|       CMD
        {
                runtime.top = list{$1}
        }

stmts:
        {
        }
|       stmts ';'
|       stmts stmt ';'
        {
                $$ = append($1, $2)
        }
|       stmts block ';'
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
                $$ = func() (number, error) {
                        x, err := a()
                        if err != nil {
                                return number{}, err
                        }
                        fmt.Println(x)
                        return number{}, nil
                }
        }

forloop:
        FOR expr block
        {
                a, b := $2, $3
                $$ = func() (number, error) {
                        for {
                                if v, err := a(); err != nil || !v.Bool() {
                                        return number{}, err
                                }
                                if err := b.Run(); err != nil {
                                        return number{}, err
                                }
                        }
                }
        }
|       FOR stmt2 ';' expr ';' stmt2 block
        {
                a, b, c := $2, $4, append($7, $6)
                $$ = func() (number, error) {
                        if _, err := a(); err != nil {
                                return number{}, err
                        }
                        for {
                                if v, err := b(); err != nil || !v.Bool() {
                                        return number{}, err
                                }
                                if err := c.Run(); err != nil {
                                        return number{}, err
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
                $$ = func() (number, error) {
                        v, err := a()
                        if err != nil {
                                return number{}, err
                        }
                        runtime.vars[s] = v
                        return v, nil
                }
        }

expr:
        expr2
|       expr op1 expr2          { $$ = $2.NewFun($1, $3) }

op1:    '+' | '-'

expr2:
        expr3
|       expr2 op2 expr3         { $$ = $2.NewFun($1, $3) }

op2:    '*' | '/' | '%'

expr3:
        num
        {
                n := $1
                $$ = func() (number, error) {
                        return n, nil
                }
        }
|       expr4

num:
        NUM
|       '-' NUM
        {
                $$ = number{
                        i:       -$2.i,
                        f:       -$2.f,
                        isFloat: $2.isFloat,
                }
        }

expr4:
        var
|       '-' expr4
        {
                a := $2
                $$ = func() (number, error) {
                        n, err := a()
                        return number{
                                i:       -n.i,
                                f:       -n.f,
                                isFloat: n.isFloat,
                        }, err
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
                $$ = func() (number, error) {
                        return runtime.vars.Get(s)
                }
        }

%%
