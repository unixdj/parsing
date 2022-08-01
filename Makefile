STAGES=	stage0 stage1 stage2 stage3 stage4 stage5 stage6a stage6b stage6

all:

present: install

.PHONY: all install present download clean

present download:
	make -C slides $@

all install clean:
	for i in slides $(STAGES) ; do make -C $$i $@ || exit 1 ; done
