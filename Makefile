VERSION = "v1.1"
SHELL = /bin/bash

build:
	rm -rf dist

	mkdir -p dist/zoom-slack-status.app/Contents/{MacOS,Resources}

	(cd icons && ./generate-icons.sh)

	go build .

	cp Info.plist dist/zoom-slack-status.app/Contents/
	cp zoom-slack-status dist/zoom-slack-status.app/Contents/MacOS/
	cp icons/icon.icns dist/zoom-slack-status.app/Contents/Resources/

	# copy app into Applications folder
	cp -R dist/zoom-slack-status.app /Applications/

release: build
	tar -czvf zoom-slack-status-$(VERSION).tar.gz -C dist/ .
	gh release create v1 "zoom-slack-status-$(VERSION).tar.gz" -n "" -t "$(VERSION)" -R caitlinelfring/zoom-slack-status
.PHONY: build release
