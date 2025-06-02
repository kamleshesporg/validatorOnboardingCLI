#!/bin/bash

 if ! docker; then
    echo "Install and launch Docker Desktop (it must be open and running in the background)."
     exit 1
 fi

BINARY_NAME="mrmintchain.exe"
 

echo "Mrmintchain Binary files downloading..."

echo "1. mrmintchain binary fetching..."
curl --progress-bar -LO https://raw.githubusercontent.com/kamleshesporg/validatorOnboardingCLI/main/mrmintchain-mac-amd64

chmod=+x mrmintchain-mac-amd64

echo "2. mrmintd binary fetching..."
# curl --progress-bar -LO https://raw.githubusercontent.com/kamleshesporg/validatorOnboardingCLI/main/chain/ethermintd-amd64

chmod=+x ethermintd

echo -e "\xE2\x9C\x94 Binary files downloaded!"

echo

echo "Installing [${BINARY_NAME}] to [${INSTALL_PATH}]"
mkdir -p ${GOBIN}
cp ${BINARY_NAME} ${INSTALL_PATH}
echo -e "\xE2\x9C\x94 Binary installed!"
echo

# For universal use : 
lipo -create -output mrmintchain mrmintchain-mac-amd64


CONTAINER_CLI=docker
image_tag=latest
 


echo "===> Pulling mrmintchain Image"
image_name="kamleshesp/mrmintchain:${image_tag}"

echo "===>  ${image_name}"
if ! ${CONTAINER_CLI} pull docker.io/${image_name}; then
    echo "❌ Failed to pull image: ${image_name}"
    exit 1
fi

cat <<EOF > .env
IMAGE_NAME=$image_name
EOF

echo "✅ .env file created!"

echo
echo  "RUN : mrmintchain --help"
echo -e "\xE2\x9C\x94 Congratulations mrmintchain installed!"




# echo "✅ Installation complete. Try running: mrmintchain.exe"