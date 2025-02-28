FROM ubuntu:22.04

# 対話的なプロンプトを防止するために環境変数を設定
ENV DEBIAN_FRONTEND=noninteractive

# 必要なパッケージをインストール
RUN apt-get update && apt-get install -y \
    curl \
    wget \
    unzip \
    git \
    python3 \
    python3-pip \
    python3-venv \
    build-essential \
    golang \
    && rm -rf /var/lib/apt/lists/*

# Goの最新バージョンをインストール
RUN curl -OL https://go.dev/dl/go1.22.0.linux-arm64.tar.gz \
    && tar -C /usr/local -xzf go1.22.0.linux-arm64.tar.gz \
    && rm go1.22.0.linux-arm64.tar.gz

# PATH環境変数を設定
ENV PATH="/usr/local/go/bin:${PATH}"

# AWS CLIのインストール - ARM64バージョン
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-aarch64.zip" -o "awscliv2.zip" \
    && unzip awscliv2.zip \
    && ./aws/install \
    && rm -rf aws awscliv2.zip

# Python仮想環境を作成してSAM CLIをインストール
RUN python3 -m venv /opt/sam-cli \
    && . /opt/sam-cli/bin/activate \
    && pip install --no-cache-dir --upgrade aws-sam-cli \
    && deactivate

# シンボリックリンクを作成してSAMコマンドを利用可能にする
RUN ln -sf /opt/sam-cli/bin/sam /usr/local/bin/sam

# AWS SAM用ディレクトリを作成し、権限を設定
RUN mkdir -p /root/.aws-sam && chmod -R 755 /root/.aws-sam

# Goパッケージをプリインストール（必要に応じてカスタマイズ）
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# 作業ディレクトリの設定
WORKDIR /workspaces

# AWS SAMテレメトリを無効化（オプション）
ENV SAM_CLI_TELEMETRY=0

# 仮想環境のPATHを通す
ENV PATH="/opt/sam-cli/bin:${PATH}"

# ユーザーが必要なコマンドを実行できるようにするためのエントリポイント
CMD ["/bin/bash"]