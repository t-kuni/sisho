#!/bin/bash

set -e

# デフォルトバージョンを設定
DEFAULT_VERSION="latest"

# バージョンを引数から取得、指定がない場合はデフォルトを使用
VERSION=${1:-$DEFAULT_VERSION}

# 最新バージョンを取得する関数
get_latest_version() {
    curl -s https://api.github.com/repos/t-kuni/sisho/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

# バージョンが "latest" の場合、最新バージョンを取得
if [ "$VERSION" = "latest" ]; then
    VERSION=$(get_latest_version)
fi

# バイナリのダウンロードURL
DOWNLOAD_URL="https://github.com/t-kuni/sisho/releases/download/${VERSION}/sisho-linux"

# インストール先ディレクトリ
INSTALL_DIR="/usr/local/bin"

# バイナリのダウンロードとインストール
echo "Downloading sisho ${VERSION}..."
if ! sudo curl -L -o "${INSTALL_DIR}/sisho" "${DOWNLOAD_URL}"; then
    echo "Error: Failed to download sisho. Exiting."
    exit 1
fi

sudo chmod +x "${INSTALL_DIR}/sisho"

# PATHの確認と追加
if ! echo $PATH | grep -q "${INSTALL_DIR}"; then
    echo "Adding ${INSTALL_DIR} to PATH..."
    echo "export PATH=\$PATH:${INSTALL_DIR}" >> ~/.bashrc
    source ~/.bashrc
fi

echo "sisho ${VERSION} has been installed successfully!"
echo "You can now use the 'sisho' command."