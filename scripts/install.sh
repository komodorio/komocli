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
echo $(get_arch)
get_download_url() {
  curl -s https://api.github.com/repos/komodorio/komocli/releases/latest \
  | jq --arg platform "${os}_${arch}" -r '.assets[] | select(.browser_download_url | contains($platform)) | .browser_download_url'
}

os=$(get_os)
arch=$(get_arch)
download_url=$(get_download_url)

echo $os
echo $arch
echo $download_url
echo "Downloading komocli package..."
curl -LO "$download_url"



echo "Extracting komocli package..."
tar -xf "komocli_0.0.3_${os}_${arch}.tar.gz"

echo "Installing komocli..."
sudo mv komocli /usr/local/bin/
rm "komocli_0.0.3_${os}_${arch}.tar.gz"
echo "komocli installation completed!"
