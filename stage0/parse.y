%{
package main
%}

%union {
}

%token NUM
%token IDENT

%%

top:    stmts
stmts:  | stmts ';' | stmts stmt ';'
stmt:   assign | expr
assign: IDENT '=' expr
expr:   expr2 | expr '+' expr2 | expr '-' expr2
expr2:  expr3 | expr2 '*' expr3 | expr2 '/' expr3 | expr2 '%' expr3
expr3:  NUM | IDENT | '-' expr3 | '(' expr ')'

%%
