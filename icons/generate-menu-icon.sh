#!/bin/bash -e

if [ ! -e "$(go env GOPATH)/bin/2goarray" ]; then
    export GO111MODULE=off
    echo "Installing 2goarray..."
    go get github.com/cratonica/2goarray
    if ! go get github.com/cratonica/2goarray; then
        echo Failure executing go get github.com/cratonica/2goarray
        exit 1
    fi
fi


generate() {
    SIZE=44

    local INPUT=$1
    local OUTPUT=$2
    local VAR=$3
    sips -z $SIZE $SIZE $INPUT --out menu_icon.png
    echo Generating $OUTPUT
    echo "//+build linux darwin" > "$OUTPUT"
    "$(go env GOPATH)/bin/2goarray" "$VAR" icons >> "$OUTPUT" < "$INPUT"
    # if [ $? -ne 0 ]; then
    #     rm menu_icon.png
    #     echo Failure generating $OUTPUT
    #     exit 1
    # fi
    gofmt -s -w "$OUTPUT"
    rm menu_icon.png
    echo Finished
}

generate icon-original.png icon_free_unix.go Free
generate icon-blue.png icon_busy_unix.go Busy
