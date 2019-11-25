build:
	go build
testunit:
	go test -v -coverprofile=coverage.txt
bench:
	go test -bench .
testrace:
	go test -v -race
test: testunit testrace
all: build test
