%{
package main
%}

%union {
        num  number
        word string
        op   op
        fun  fun
        list list
}

%token <num> NUM
%token <word> IDENT
%token <fun> CMD
%token <op> '+' '-' '*' '/' '%' '&' '^' BIC '|' LSHIFT RSHIFT
%token <op> '!' LAND LOR '<' '>' LE GE EQ NE
%token <op> '=' ADDEQ SUBEQ MULEQ DIVEQ MODEQ ANDEQ XOREQ BICEQ OREQ
%token <op> LSHIFTEQ RSHIFTEQ INC DEC FOR

%type <num> num
%type <op> op3 op4 op5 unop assop postop
%type <fun> stmt stmt2 assign var
%type <fun> expr expr2 expr3 expr4 expr5 expr6 expr7
%type <list> stmts list block

%%

top:
        stmts                   { runtime.top = $1 }
|       CMD                     { runtime.top = append(runtime.top[:0], $1) }

stmts:
                                { }
|       stmts ';'
|       stmts stmt ';'          { $$ = append($1, $2) }
|       stmts list              { $$ = append($1, $2...) }

list:
        block
|       FOR stmt2 ';' expr ';' stmt2 block
        {
                $$ = list{$2, $1.NewFun($4, append($7, $6).NewFun())}
        }

block:  '{' stmts '}'           { $$ = $2 }

stmt:
        stmt2
|       FOR expr block          { $$ = $1.NewFun($2, $3.NewFun()) }

stmt2:
        assign
|       expr                    { $$ = printOp.NewFun($1, nil) }

assign:
        IDENT assop expr        { $$ = NewAssign($1, $2, $3) }
|       IDENT postop            { $$ = NewAssign($1, $2, nil) }

assop:    
        '=' | ADDEQ | SUBEQ | MULEQ | DIVEQ | MODEQ
|       ANDEQ | XOREQ | BICEQ | OREQ | LSHIFTEQ | RSHIFTEQ

postop: INC | DEC

expr:
        expr2
|       expr LOR expr2          { $$ = $2.NewFun($1, $3) }

expr2:
        expr3
|       expr2 LAND expr3        { $$ = $2.NewFun($1, $3) }

expr3:
        expr4
|       expr3 op3 expr4         { $$ = $2.NewFun($1, $3) }

op3:    EQ | NE | '<' | LE | '>' | GE

expr4:
        expr5
|       expr4 op4 expr5         { $$ = $2.NewFun($1, $3) }

op4:    '+' | '-' | '|' | '^'

expr5:
        expr6
|       expr5 op5 expr6         { $$ = $2.NewFun($1, $3) }

op5:    '*' | '/' | '%' | '&' | BIC | LSHIFT | RSHIFT

expr6:
        expr7
|       num
        {
                n := $1
                $$ = func() (number, error) {
                        return n, nil
                }
        }

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

expr7:
        var
|       unop expr7              { $$ = $1.NewFun($2, nil) }
|       '(' expr ')'            { $$ = $2 }

unop:    '-' | '^' | '!'

var:    IDENT                   { $$ = runtime.vars.NewGet($1) }

%%
