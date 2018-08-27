S3BUCKET := ec2-instance-lifecycle
VERSION := $(shell git describe --tags --always)

PROGS := $(subst cmd/,,$(wildcard cmd/*))

zip: $(patsubst %,dist/%.zip,$(PROGS))

bin/%: export GOOS := linux
bin/%: export GOARCH := amd64
bin/%: ./cmd/%/main.go internal/*.go
	go build -o $@ $<

dist/%.zip: bin/% | dist
	zip -u -j $@ $<

dist:
	mkdir dist

upload: zip
	aws s3 sync dist/ s3://$(S3BUCKET)/$(VERSION)/
.PHONY: upload

test_ecs_instance_drainer:
	LAMBDA_VERSION=$(VERSION) go test -v -timeout 30m ./test/ecs_instance_drainer

test_ecs_instance_ready:
	LAMBDA_VERSION=$(VERSION) go test -v -timeout 30m ./test/ecs_instance_ready
