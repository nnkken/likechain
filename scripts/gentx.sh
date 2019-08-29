#!/bin/bash

set -e

LIKE_HOME="$(dirname "$0")/.."
pushd "$LIKE_HOME" > /dev/null
LIKE_HOME=$(pwd)
popd > /dev/null

COMMISSION_RATE="0.2"
COMMISSION_RATE_MAX="0.5"
COMMISSION_RATE_CHANGE="0.01"

read -p "Enter some description of your node: " DETAILS
read -p "Enter the amount you want to stake (including the coin name, example: '1000000000000000nanolike'):" AMOUNT

echo ""
echo "Now the script will generate the genesis transaction, please confirm and enter your passphrase."

docker run -it --rm \
    --volume "$LIKE_HOME/.liked:/root/.liked" \
    --volume "$LIKE_HOME/.likecli:/root/.likecli" \
    likechain/likechain liked add-genesis-account \
        validator "$AMOUNT"

docker run -it --rm \
    --volume "$LIKE_HOME/.liked:/root/.liked" \
    --volume "$LIKE_HOME/.likecli:/root/.likecli" \
    likechain/likechain liked gentx \
        --name validator \
        --details "$DETAILS" \
        --amount "$AMOUNT" \
        --commission-rate "$COMMISSION_RATE" \
        --commission-max-rate "$COMMISSION_RATE_MAX" \
        --commission-max-change-rate "$COMMISSION_RATE_CHANGE"

echo "Genesis transaction generated. Please send the generated file in ./.liked/config/gentx/ to us."