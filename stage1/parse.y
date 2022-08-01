%{
package main
%}

%union {
        num  int
        word string
        tree *tree
        list list
}

%token <num> NUM
%token <word> IDENT

%type <num> num
%type <tree> stmt assign expr expr2 expr3 expr4 var
%type <list> top stmts

%%

top:
        stmts
        {
                top = $1
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

assign:
        var '=' expr
        {
                $$ = &tree{
                        left:  $1,
                        typ:   '=',
                        right: $3,
                }
        }

expr:
        expr2
|       expr '+' expr2
        {
                $$ = &tree{
                        left:  $1,
                        typ:   '+',
                        right: $3,
                }
        }
|       expr '-' expr2
        {
                $$ = &tree{
                        left:  $1,
                        typ:   '-',
                        right: $3,
                }
        }
expr2:
        expr3
|       expr2 '*' expr3
        {
                $$ = &tree{
                        left:  $1,
                        typ:   '*',
                        right: $3,
                }
        }
|       expr2 '/' expr3
        {
                $$ = &tree{
                        left:  $1,
                        typ:   '/',
                        right: $3,
                }
        }
|       expr2 '%' expr3
        {
                $$ = &tree{
                        left:  $1,
                        typ:   '%',
                        right: $3,
                }
        }

expr3:
        num
        {
                $$ = &tree{
                        typ: NUM,
                        n:   $1,
                }
        }
|       expr4

num:
        NUM
|       '-' num {
                $$ = -$2
        }

expr4:
        var
|       '-' expr4 {
                $$ = &tree{
                        typ:   '-',
                        right: $2,
                }
        }
|       '(' expr ')'
        {
                $$ = $2
        }

var:
        IDENT
        {
                $$ = &tree{
                        typ: IDENT,
                        s:   $1,
                }
        }

%%
