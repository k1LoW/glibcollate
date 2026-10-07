# Image whose locale-archive the glibc 2.41 tables come from. It is used only to
# pin glibc, since its digest fixes the glibc build and the compiled locale-archive.
IMAGE_GLIBC2_41 = postgres:18@sha256:74935e72241653ca55e0414067e6d8763aceb8a810eb51b452253ec3dcfc4336

default: test

ci:
	go test ./... -coverpkg=./... -coverprofile=coverage.out -covermode=count

test:
	go test ./... -coverpkg=./... -coverprofile=coverage.out -covermode=count

race:
	go test ./... -race

lint:
	golangci-lint run ./...
	cd devtools && golangci-lint run --config ../.golangci.yml ./...

generate:
	cd devtools && go run ./gen -image $(IMAGE_GLIBC2_41) -locale en_US.UTF-8

# Compares every collation with the real strcoll_l. Needs Docker.
difftest:
	cd devtools && go test -count=1 ./...

credits:
	go install github.com/Songmu/gocredits/cmd/gocredits@v1.0.0
	gocredits . > CREDITS
	cat _EXTRA_CREDITS >> CREDITS

prerelease_for_tagpr:
	$(MAKE) credits
	git add CHANGELOG.md CREDITS

.PHONY: default ci test race lint generate difftest credits prerelease_for_tagpr
