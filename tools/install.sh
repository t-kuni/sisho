#!/bin/bash

VERSION=$1

if [ -z "$VERSION" ]; then
  echo "Usage: $0 <version>"
  exit 1
fi

while true; do
  go install github.com/t-kuni/sisho@v$VERSION
  if [ $? -eq 0 ]; then
    echo "Installation successful"
    break
  else
    echo "Installation failed, retrying in 5 seconds..."
    sleep 5
  fi
done