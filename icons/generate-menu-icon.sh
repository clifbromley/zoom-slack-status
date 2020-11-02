#!/bin/sh

export GO111MODULE=off
if [ -z "$GOPATH" ]; then
    echo GOPATH environment variable not set
    exit 1
fi

if [ ! -e "$GOPATH/bin/2goarray" ]; then
    echo "Installing 2goarray..."
    go get github.com/cratonica/2goarray
    if ! go get github.com/cratonica/2goarray; then
        echo Failure executing go get github.com/cratonica/2goarray
        exit 1
    fi
fi

#if [ -z "$1" ]; then
#    echo Please specify a PNG file
#    exit
#fi
#
#if [ ! -f "$1" ]; then
#    echo $1 is not a valid file
#    exit
#fi

SIZE=44
sips -z $SIZE $SIZE icon-original.png --out menu_icon.png

OUTPUT=iconunix.go
echo Generating $OUTPUT
echo "//+build linux darwin" > $OUTPUT
echo >> $OUTPUT
"$GOPATH/bin/2goarray" Data icon >> $OUTPUT < "menu_icon.png"
if [ $? -ne 0 ]; then
    rm menu_icon.png
    echo Failure generating $OUTPUT
    exit 1
fi
gofmt -s -w $OUTPUT
rm menu_icon.png
echo Finished
