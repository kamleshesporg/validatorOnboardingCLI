#!/bin/bash


BINARY_NAME="mrmintchain.exe"
 

echo "Mrmintchain Binary files downloading..."

echo "1. mrmintchain binary fetching..."
# curl --progress-bar -LO https://raw.githubusercontent.com/kamleshesporg/validatorOnboardingCLI/main/mrmintchain.exe

chmod +x mrmintchain.exe

echo "2. mrmintd binary fetching..."
# curl --progress-bar -LO https://raw.githubusercontent.com/kamleshesporg/validatorOnboardingCLI/main/chain/ethermintd.exe
 
chmod +x ethermintd.exe

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



# Set target path
TARGET_DIR="/c/tools/bin"
EXE_NAME="mrmintchain.exe"
SOURCE_EXE="./$EXE_NAME"

# Create the directory if it doesn't exist
mkdir -p "$TARGET_DIR"

# Copy the binary
cp "$SOURCE_EXE" "$TARGET_DIR/"

echo "✅ Copied $EXE_NAME to $TARGET_DIR"

# Convert POSIX path to Windows path for PowerShell
WIN_TARGET_DIR=$(echo $TARGET_DIR | sed 's|/c/|C:/|')

# Add to PATH permanently (user scope) using PowerShell
powershell.exe -Command "
\$currentPath = [Environment]::GetEnvironmentVariable('Path', 'User');
if (-not \$currentPath.Split(';') -contains '$WIN_TARGET_DIR') {
    [Environment]::SetEnvironmentVariable('Path', \$currentPath + ';$WIN_TARGET_DIR', 'User');
    Write-Host '✅ Added $WIN_TARGET_DIR to User PATH. Restart your terminal to apply.';
} else {
    Write-Host 'ℹ️ $WIN_TARGET_DIR already exists in PATH.';
}
"
echo "currentPath = [Environment]::GetEnvironmentVariable('Path', 'User');
if (-not \$currentPath.Split(';') -contains '$WIN_TARGET_DIR') {
    [Environment]::SetEnvironmentVariable('Path', \$currentPath + ';$WIN_TARGET_DIR', 'User');
    Write-Host '✅ Added $WIN_TARGET_DIR to User PATH. Restart your terminal to apply.';
} else {
    Write-Host 'ℹ️ $WIN_TARGET_DIR already exists in PATH.';
}"

echo "✅ Installation complete. Try running: mrmintchain.exe"