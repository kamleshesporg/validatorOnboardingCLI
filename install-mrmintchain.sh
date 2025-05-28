#!/bin/bash

GOBIN="$(go env GOBIN -1)"

if [ -z "$GOBIN" ]; then
    GOBIN="$(go env GOPATH -1)"
fi 

BINARY_NAME="mrmintchain"
INSTALL_PATH="${GOBIN}/${BINARY_NAME}"
 

echo "Mrmintchain Binary files downloading..."

echo "1. mrmintchain binary fetching..."
curl -sSLO https://raw.githubusercontent.com/kamleshesporg/validatorOnboardingCLI/main/mrmintchain

echo "2. mrmintd binary fetching..."
curl -sSLO https://raw.githubusercontent.com/kamleshesporg/validatorOnboardingCLI/main/chain/ethermintd

echo -e "\xE2\x9C\x94 Binary files downloaded!"

echo

echo "Installing [${BINARY_NAME}] to [${INSTALL_PATH}]"
mkdir -p ${GOBIN}
cp ${BINARY_NAME} ${INSTALL_PATH}
echo -e "\xE2\x9C\x94 Binary installed!"
echo


CONTAINER_CLI=docker
image_tag=latest
 

echo "===> Pulling mrmintchain Image"
image_name="docker.io/kamleshesp/mrmintchain:${image_tag}"

echo "===>  ${image_name}"
echo  -e "===>  ${CONTAINER_CLI} pull ${image_name}"

echo
echo  "RUN : mrmintchain --help"
echo -e "\xE2\x9C\x94 Congratulations mrmintchain installed!"
