TARGET!=	basename `pwd`
INSTALLDIR=	../go
GENTARGET=	${INSTALLDIR}/${TARGET}.go
CLEANFILES=	y.go y.output

all: ${TARGET}

.PHONY: all install clean

${TARGET}: main.go parse.y
	go generate
	go build

install: ${GENTARGET}

${GENTARGET}: ${TARGET}
	mkdir -p ../go
	( sed 's/if false/if true/' main.go ; \
	  sed -e '/^import (/,/^)/d' \
	      -e '/^\(package\|import\|\/\/line\)/d' \
	      -e 's/__yyfmt__/fmt/g' y.go ) \
	  > ${GENTARGET}

clean:
	-rm -rf ${GENTARGET} ${TARGET} ${CLEANFILES}
