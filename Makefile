.EXPORT_ALL_VARIABLES:
GOOS=linux
GOARCH=amd64

target = bin/${GOOS}

all: echoserver primetime means budgetchat

echoserver: ${target}/echosrvr
primetime: ${target}/primetime
means: ${target}/means
budgetchat: ${target}/budgetchat

${target}/echosrvr: ./echoserver/$(wildcard *.go)
	@mkdir -p ${target}
	go build -o ${target}/echosrvr ./echoserver/...

${target}/primetime: ./primetime/$(wildcard *.go)
	@mkdir -p bin
	go build -o ${target}/primetime ./primetime/...

${target}/means: ./means-to-an-end/*.go
	@mkdir -p bin
	go build -o ${target}/means ./means-to-an-end/...

${target}/budgetchat: ./budgetchat/*.go
	@mkdir -p bin
	go build -o ${target}/budgetchat ./budgetchat/...
	
clean:
	rm -rf ./bin