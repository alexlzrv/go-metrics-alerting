.PHONY: golangci-lint-run
golangci-lint-run: _golangci-lint-rm-unformatted-report

.PHONY: _golangci-lint-reports-mkdir
_golangci-lint-reports-mkdir:
	mkdir -p ./golangci-lint

.PHONY: _golangci-lint-run
_golangci-lint-run: _golangci-lint-reports-mkdir
	-docker run --rm \
    -v $(shell pwd):/app \
    -v $(shell pwd)/golangci-lint/cache:/root/.cache \
    -w /app \
    golangci/golangci-lint:v1.53.3 \
        golangci-lint run \
            -c .golangci.yml \
	> ./golangci-lint/report-unformatted.json

.PHONY: _golangci-lint-format-report
_golangci-lint-format-report: _golangci-lint-run
	cat ./golangci-lint/report-unformatted.json | jq > ./golangci-lint/report.json

.PHONY: _golangci-lint-rm-unformatted-report
_golangci-lint-rm-unformatted-report: _golangci-lint-format-report
	rm ./golangci-lint/report-unformatted.json

.PHONY: golangci-lint-clean
golangci-lint-clean:
	sudo rm -rf ./golangci-lint

.PHONY: proto
proto:
	protoc -I ./api  --go_out ./api/server --go_opt paths=source_relative --go-grpc_out ./api/server --go-grpc_opt paths=source_relative ./api/server.proto
