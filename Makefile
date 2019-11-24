build:
	go build
testunit:
	go test -coverprofile=coverage.txt
bench:
	go test -bench .
testrace:
	go test -race
test: testunit testrace
all: build test
