#!/bin/bash

GOBIN="$(go env GOBIN -1)"

if [ -z "$GOBIN" ]; then
    GOBIN="$(go env GOPATH -1)/bin"
fi 

BINARY_NAME="mrmintchain"
INSTALL_PATH="${GOBIN}/${BINARY_NAME}"
 

echo "Mrmintchain Binary files downloading..."

echo "1. mrmintchain binary fetching..."
curl --progress-bar -LO https://raw.githubusercontent.com/kamleshesporg/validatorOnboardingCLI/main/mrmintchain

chmod +x mrmintchain

echo "2. mrmintd binary fetching..."
curl --progress-bar -LO https://raw.githubusercontent.com/kamleshesporg/validatorOnboardingCLI/main/chain/ethermintd

chmod +x ethermintd

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
image_name="kamleshesp/mrmintchain:${image_tag}"

echo "===>  ${image_name}"
echo  -e "===>  ${CONTAINER_CLI} pull docker.io/${image_name}"

cat <<EOF > .env
IMAGE_NAME=$image_name
EOF

echo "✅ .env file created!"

echo
echo  "RUN : mrmintchain --help"
echo -e "\xE2\x9C\x94 Congratulations mrmintchain installed!"
