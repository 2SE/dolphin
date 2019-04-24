APPNAME = dolphin
EXECPATH = ./cmd/dolphin
DATEVERSION=$(shell date -u +%Y%m%d)
COMMIT_SHA=$(shell git rev-parse --short HEAD)
LDFLAGS="-X main.VERSION=${DATEVERSION} -X main.GITCOMMIT=${COMMIT_SHA}"

.PHONY:all
all:build

.PHONY:build
build:generate
	go build -o ${EXECPATH}/${APPNAME} -ldflags $(LDFLAGS) ${EXECPATH}/*.go

.PHONY:cluster
cluster:
	go build -o ./cmd/cluster/gateway ./cmd/cluster/main.go

.PHONY:generate
generate:
	go generate ${EXECPATH}

.PHONY:clean
clean:
	-rm -f ${EXECPATH}/${APPNAME}
	-rm -f ./${APPNAME}