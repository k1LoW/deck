PKG = github.com/k1LoW/deck
COMMIT = $(shell git rev-parse --short HEAD)

BUILD_LDFLAGS = "-s -w -X $(PKG)/version.Revision=$(COMMIT)"

default: test

ci: depsdev test

test:
	go test ./... -coverprofile=coverage.out -covermode=count -count=1

fulltest:
	env TEST_INTEGRATION=1 go test -v ./... -coverprofile=coverage.out -covermode=count -count=1

build:
	go build -ldflags=$(BUILD_LDFLAGS) -trimpath -o deck cmd/deck/main.go

install:
	go install -ldflags=$(BUILD_LDFLAGS) -trimpath ./cmd/deck

lint:
	golangci-lint run ./...

fuzz:
	go test -fuzz=FuzzParse -fuzztime=1m ./md/.
	go test -fuzz=FuzzGenerateActions -fuzztime=1m .

integration:
	env TEST_INTEGRATION=1 go test -v . -test.failfast -run \
	  'TestApply$$|TestRoundTripSlidesToGoogleSlidesPresentationAndBack$$|TestApplyMarkdown$$|TestAction$$' \
	  -timeout 30m

depsdev:

credits:
	go install github.com/Songmu/gocredits/cmd/gocredits@v1.0.0
	gocredits . > CREDITS

prerelease_for_tagpr:
	$(MAKE) credits
	git add CHANGELOG.md CREDITS go.mod go.sum

.PHONY: default ci test fulltest build install lint fuzz integration depsdev prerelease_for_tagpr credits
