#!/usr/bin/env bash
set -e
set -x

rm -rf dist/zoom-slack-status.app

pushd icons || exit 1
bash generate-icons.sh
bash generate-menu-icon.sh
popd || exit 1

go build .

mkdir -p dist/zoom-slack-status.app/Contents/{MacOS,Resources}

cp Info.plist dist/zoom-slack-status.app/Contents/
cp zoom-slack-status dist/zoom-slack-status.app/Contents/MacOS/
cp icons/icon.icns dist/zoom-slack-status.app/Contents/Resources/
