#!/bin/sh

TMP_NAME="./$(head -n 1 -c 32 /dev/urandom | tr -dc 'a-zA-Z0-9'| fold -w 32)"

if which curl > /dev/null; then
    if curl --help 2>&1 | grep "--progress-bar" > /dev/null 2>&1; then
        PROGRESS="--progress-bar"
    fi

    set -- curl -L $PROGRESS -o "$TMP_NAME"
    LATEST=$(curl -sL https://api.github.com/repos/tez-capital/tezpay/releases/latest | grep tag_name | sed 's/  "tag_name": "//g' | sed 's/",//g')
else
    if wget --help 2>&1 | grep "--show-progress" > /dev/null 2>&1; then
        PROGRESS="--show-progress"
    fi
    set -- wget -q $PROGRESS -O "$TMP_NAME"
    LATEST=$(wget -qO- https://api.github.com/repos/tez-capital/tezpay/releases/latest | grep tag_name | sed 's/  "tag_name": "//g' | sed 's/",//g')
fi

if ./tezpay version | grep "$LATEST"; then
    echo "Latest tezpay already available."
    exit 0
fi

PLATFORM=$(uname -m)
# remap platform
if [ "$PLATFORM" = "x86_64" ]; then
	PLATFORM="amd64"
elif [ "$PLATFORM" = "aarch64" ]; then
	PLATFORM="arm64"
fi
echo "Downloading tezpay-linux-$PLATFORM $LATEST..."


printf "https://github.com/tez-capital/tezpay/releases/download/$LATEST/tezpay-linux-$PLATFORM"
if "$@" "https://github.com/tez-capital/tezpay/releases/download/$LATEST/tezpay-linux-$PLATFORM" &&
    mv "$TMP_NAME" ./tezpay &&
    chmod +x ./tezpay; then
    echo "tezpay $LATEST for $PLATFORM successfuly installed."
else 
    echo "tezpay installation failed!" 1>&2
    exit 1
fi
