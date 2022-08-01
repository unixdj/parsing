DOTFILES= \
	enc_sm.dot \
	packet-len.dot \
	re.dot \
	banana.dot \
       	123-num.dot \
       	123-token.dot \
       	123-p1.dot \
       	123-p2.dot \
       	123-p3.dot \
       	123-p4.dot \
       	lex-parse.dot

TARGETS=	${DOTFILES:%.dot=${OBJDIR}/%.png}

OBJDIR=		obj
GODIR!=		realpath ../go

all: ${TARGETS}

.PHONY: all install present download

install: all download

present: install
	GOPATH=${GODIR}:"$$GOPATH" present -orighost localhost

download: ${GODIR}/src/github.com/mattn/go-isatty ${GODIR}/src/golang.org/x/sys

${GODIR}/src/github.com/mattn/go-isatty:
	git clone https://github.com/mattn/go-isatty \
		${GODIR}/src/github.com/mattn/go-isatty

${GODIR}/src/golang.org/x/sys:
	git clone https://go.googlesource.com/sys ${GODIR}/src/golang.org/x/sys

.SUFFIXES: .png .dot

${OBJDIR}/%.png: %.dot
	mkdir -p ${OBJDIR}
	dot -Tpng -o$@ $<

clean:
	-chmod -R +w ${GODIR}
	-rm -rf ${GODIR} ${OBJDIR}
