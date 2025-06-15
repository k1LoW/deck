PKG = github.com/k1LoW/deck
COMMIT = $$(git describe --tags --always)
OSNAME=${shell uname -s}
DATE = $$(date '+%Y-%m-%d_%H:%M:%S%z')

export GO111MODULE=on

BUILD_LDFLAGS = -X $(PKG).commit=$(COMMIT) -X $(PKG).date=$(DATE)

default: test

ci: depsdev test

test:
	go test ./... -coverprofile=coverage.out -covermode=count -count=1

build:
	go build -ldflags="$(BUILD_LDFLAGS)" -o deck cmd/deck/main.go

lint:
	golangci-lint run ./...

fuzz:
	go test -fuzz=FuzzParse -fuzztime=1m ./md/.
	go test -fuzz=FuzzGenerateActions -fuzztime=1m .

export TEST_PRESENTATION_ID=1_QRwonGFKTcsakL0QFCUNvNKWMedDS-C5KRMqMTwz6E
integration:
	env TEST_INTEGRATION=1 go test -v -test.failfast . -run 'TestConvert$$|TestApply$$|TestApplyAction$$' -timeout 30m

depsdev:
	go install github.com/Songmu/ghch/cmd/ghch@latest
	go install github.com/Songmu/gocredits/cmd/gocredits@latest

prerelease:
	git pull origin main --tag
	go mod tidy
	ghch -w -N ${VER}
	gocredits -w .
	git add CHANGELOG.md CREDITS go.mod go.sum
	git commit -m'Bump up version number'
	git tag ${VER}

prerelease_for_tagpr: depsdev
	go mod download
	gocredits -w .
	git add CHANGELOG.md CREDITS go.mod go.sum

release:
	git push origin main --tag

.PHONY: default test
