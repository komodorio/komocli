#!/bin/bash

get_os() {
    case "$OSTYPE" in
        linux*)   echo "linux";;
        darwin*)  echo "darwin";;
        win*)     echo "windows";;
        *)        echo "unknown";;
    esac
}
get_arch() {
    arch1=$(uname -m)
    if [[ "$arch1" == "x86_64" ]]; then
        arch1="amd64"
    fi
    echo $arch1
}

get_download_url() {
    curl -s https://api.github.com/repos/komodorio/komocli/releases/latest \
    | curl -s https://api.github.com/repos/komodorio/komocli/releases/latest | grep -o "https://.*${os}_${arch}"
}

get_version() {
    curl -s https://api.github.com/repos/komodorio/komocli/releases/latest | jq -r '.name'
}

os=$(get_os)
arch=$(get_arch)
download_url=$(get_download_url)
version=$(get_version)

echo $os
echo $arch
echo $download_url
echo "Downloading komocli package..."
curl -LO "$download_url"

echo "Installing komocli..."
sudo mv "komocli_${version}_${os}_${arch}" /usr/local/bin/komocli
chmod +x /usr/local/bin/komocli
echo "komocli installation completed!"
