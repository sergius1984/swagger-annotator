BINARY_NAME=swagger-annotator
CMD_DIR=./cmd/swagger-annotator
BUILD_DIR=./build

.PHONY: build test lint clean install

build:
	@echo "==> Сборка ${BINARY_NAME}"
	go build -o ${BUILD_DIR}/${BINARY_NAME} ${CMD_DIR}

test:
	@echo "==> Запуск тестов"
	go test ./... -v

lint:
	@echo "==> Линтинг"
	golangci-lint run ./...

install:
	@echo "==> Установка в GOPATH/bin"
	go install ${CMD_DIR}

clean:
	@echo "==> Очистка"
	rm -rf ${BUILD_DIR}