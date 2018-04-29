# dwm-status Makefile

FILE = dwmStatus.go
DIR = /usr/local/bin

dwmStatus: $(FILE)
	go build $(FILE)

clean:
	echo "cleaning binary file"
	rm -f dwmStatus

install:
	echo "installing dwmStatus file to ${DIR}"
	cp -f dwmStatus ${DIR}/
	chmod 755 ${DIR}/dwmStatus

uninstall:
	echo removing execurable file from ${DIR}
	rm -rf ${DIR}/dwmStatus
