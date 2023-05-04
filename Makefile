.EXPORT_ALL_VARIABLES:
GOOS=linux
GOARCH=amd64

target = bin/${GOOS}

all: echoserver primetime means

echoserver: ${target}/echosrvr
primetime: ${target}/primetime
means: ${target}/means

${target}/echosrvr: ./echoserver/$(wildcard *.go)
	@mkdir -p ${target}
	go build -o ${target}/echosrvr ./echoserver/...

${target}/primetime: ./primetime/$(wildcard *.go)
	@mkdir -p bin
	go build -o ${target}/primetime ./primetime/...

${target}/means: ./means-to-an-end/*.go
	@mkdir -p bin
	go build -o ${target}/means ./means-to-an-end/...

clean:
	rm -rf ./bin