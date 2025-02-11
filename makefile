.PHONY: generate-env

# 現在の環境変数をすべて.envファイルに出力
generate-env:
	@echo "# Generated from env on $$(date)" > .env
	@env | sort >> .env
	@echo ".env file has been generated with all environment variables"

# .envファイルの内容を確認
check-env:
	@if [ -f .env ]; then \
		cat .env; \
	else \
		echo "Error: .env file does not exist"; \
		exit 1; \
	fi