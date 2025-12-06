.PHONY: build run clean test fmt vet

# アプリケーション名
APP_NAME = salary

# ビルドターゲット
build:
	go build -o $(APP_NAME) main.go

# 実行
run:
	go run main.go

# クリーンアップ
clean:
	rm -f $(APP_NAME)

# テスト実行
test:
	go test ./...

# フォーマット
fmt:
	go fmt ./...

# 静的解析
vet:
	go vet ./...

# 依存関係更新
deps:
	go mod tidy
