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
%type <op> op3 op4 op5 unop assignop incdec
%type <fun> stmt stmt2 assign
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
        IDENT assignop expr     { $$ = NewAssign($1, $2, $3) }
|       IDENT incdec            { $$ = NewAssign($1, $2, nil) }

assignop:
        '=' | ADDEQ | SUBEQ | MULEQ | DIVEQ | MODEQ
|       ANDEQ | XOREQ | BICEQ | OREQ | LSHIFTEQ | RSHIFTEQ

incdec: INC | DEC

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
|       num                     { $$ = $1.NewFun() }

num:
        NUM
|       unop num                { $$ = $2.RunUnary($1) }

expr7:
        '(' expr ')'            { $$ = $2 }
|       IDENT                   { $$ = runtime.vars.NewGet($1) }
|       unop expr7              { $$ = $1.NewFun($2, nil) }

unop:    '-' | '^' | '!'

%%
