export GO111MODULE=on
export GOFLAGS=-mod=vendor
BIN = jenkins-exporter
SRC = cmd/jenkins-exporter/main.go

default: all

all:
	$(info * compiling $(BIN))
	@CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o $(BIN) $(SRC)
