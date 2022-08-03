# Parsing and interpretation with goyacc and closures (presentation)

A presentation about parsing, interpreters and a way to write
those in Go.  The sample code parses and runs arithmetic
expressions.


## Authors and copyrights

Copyright (c) 2022, Vadim Vygonets  
Portions Copyright (c) 2013-2015, Scott Vokes  
Portions Copyright (c) 2010, J. Finkelstein  
Portions Copyright (c) ca. 1930, Willi Illger

See the [LICENSE](LICENSE) file for details.


## Running the presentation

- Install graphviz.
  https://graphviz.org/
- Install Go prerequisites
```shell
go install golang.org/x/tools/cmd/goyacc@latest
go install golang.org/x/tools/cmd/present@latest
```
- Build and run the presentation
```shell
make present
```

The slides will be available at:

http://localhost:3999

Badly rendered PDFs are available in branch pdf, but the above
will let you run the code samples.


## Building the code

- Install goyacc:
```shell
go install golang.org/x/tools/cmd/goyacc@latest
```
- In a stage directory:
```shell
go generate
go build
```


## Code

The example code is broken into 7 stages of development.


### Stage 0

- Lexer, parser
- Simple syntax:
  - Integer numbers
  - Five arithmetic operators (`+`, `-`, `*`, `/`, `%`)
  - Parentheses
  - Variables: assignment, use


### Stage 1

- Printing the parse tree


### Stage 2

- Interpreter


### Stage 3

- `for` loops


### Stage 4

- Error handling


### Stage 5

- Interactivity


### Stage 6

- Floating point support
- A better parser, making adding more operators easier


#### Development stage 6a

- Simpler parser code for binary integer operators


#### Development stage 6b

- Floating point support for Stage 6a
  - Automatic conversion between integer and floating point numbers


#### Stage 6 final

- Better parser
- Floating point support
- More operators
  - Binary integer / floating point operators:
    `+`, `-`, `*`, `/`, `%`
  - Binary integer-only operators:
    `&`, `^`, `&^`, `|`, `<<`, `>>`
  - Unary operators:
    `-`, `^`, `!`
  - Short-circuit logic: `&&`, `||`
  - Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=`
  - Assignment:
    - Simple: `=`
    - With arithmetic operators:
      `+=`, `-=`, `*=`, `/=`, `%=`, `&=`, `^=`, `&^=`, `|=`, `<<=`, `>>=`
    - Increment/decrement: `++`, `--`


## References

Yacc: Yet Another Compiler-Compiler  
*Stephen C. Johnson*, 31 July 1978  
Unix Programmer's Manual, Seventh Edition, Volume 2B  
January 1979  
https://plan9.io/7thEdMan/v7vol2b.pdf  
pages 3-35

`goyacc`  
https://pkg.go.dev/golang.org/x/tools/cmd/goyacc

Guns and Butter: Towards Formal Axioms of Input Validation  
_Robert J. Hansen, Meredith L. Patterson_  
Presented at the Black Hat conference USA, 2005  
https://www.blackhat.com/presentations/bh-usa-05/BH_US_05-Hansen-Patterson/HP2005.pdf

RFC 791 - Internet Protocol  
_Jon Postel_ (Editor), September 1981  
https://www.rfc-editor.org/rfc/rfc791.txt

Defective C++  
_Yossi Kreinin_, 2007-2009  
https://yosefk.com/c++fqa/defective.html  
Part of C++ FQA Lite  
https://yosefk.com/c++fqa/

Slavoj Žižek Responds to Noam Chomsky:
‘I Don’t Know a Guy Who Was So Often Empirically Wrong’  
_Mike Springer_, Open Culture, 17 July 2013  
https://www.openculture.com/2013/07/slavoj-zizek-responds-to-noam-chomsky.html

Less is exponentially more  
_Rob Pike_, 25 June 2012  
https://commandcenter.blogspot.com/2012/06/less-is-exponentially-more.html


## Other resources

Compilers: Principles, Techniques, and Tools (2nd Edition)  
_Alfred V. Aho, Monica S. Lam, Ravi Sethi, and Jeffrey D. Ullman_, 2006  
ISBN 0-321-48681-1  
https://suif.stanford.edu/dragonbook/

Reflections on Trusting Trust  
_Ken Thompson_'s Turing Award lecture, 1984  
Communications of the ACM, Volume 27, Number 8, pages 761-763  
https://www.cs.cmu.edu/~rdriley/487/papers/Thompson_1984_ReflectionsonTrustingTrust.pdf

LANGSEC: Language-theoretic Security  
https://langsec.org/
