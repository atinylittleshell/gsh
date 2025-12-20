#!/bin/bash
# Script to create a macOS DMG installer for gsh
# Usage: ./create-dmg.sh <version> <arch> <source_dir> <output_dir>
#   version: The version number (e.g., 0.27.0)
#   arch: The architecture (x86_64 or arm64)
#   source_dir: Directory containing the extracted binary and files
#   output_dir: Directory to output the DMG file

set -e

VERSION="$1"
ARCH="$2"
SOURCE_DIR="$3"
OUTPUT_DIR="$4"

if [ -z "$VERSION" ] || [ -z "$ARCH" ] || [ -z "$SOURCE_DIR" ] || [ -z "$OUTPUT_DIR" ]; then
    echo "Usage: $0 <version> <arch> <source_dir> <output_dir>"
    exit 1
fi

APP_NAME="gsh"
DMG_NAME="${APP_NAME}_${VERSION}_macos_${ARCH}.dmg"
VOLUME_NAME="${APP_NAME} ${VERSION}"
DMG_TEMP="${OUTPUT_DIR}/${APP_NAME}_temp.dmg"
DMG_FINAL="${OUTPUT_DIR}/${DMG_NAME}"

echo "Creating DMG for ${APP_NAME} v${VERSION} (${ARCH})..."

# Create a temporary directory for the DMG contents
DMG_CONTENTS=$(mktemp -d)
trap 'rm -rf "${DMG_CONTENTS}"' EXIT

# Create the directory structure
mkdir -p "${DMG_CONTENTS}/${APP_NAME}"

# Copy the binary
cp "${SOURCE_DIR}/gsh" "${DMG_CONTENTS}/${APP_NAME}/"
chmod +x "${DMG_CONTENTS}/${APP_NAME}/gsh"

# Copy documentation files if they exist
[ -f "${SOURCE_DIR}/LICENSE" ] && cp "${SOURCE_DIR}/LICENSE" "${DMG_CONTENTS}/${APP_NAME}/"
[ -f "${SOURCE_DIR}/README.md" ] && cp "${SOURCE_DIR}/README.md" "${DMG_CONTENTS}/${APP_NAME}/"

# Create an installation instructions file
cat > "${DMG_CONTENTS}/${APP_NAME}/INSTALL.txt" << 'EOF'
gsh Installation Instructions
==============================

To install gsh, copy the 'gsh' binary to a directory in your PATH.

Recommended installation:

    sudo cp gsh /usr/local/bin/
    sudo chmod +x /usr/local/bin/gsh

Or for a user-local installation:

    mkdir -p ~/.local/bin
    cp gsh ~/.local/bin/
    chmod +x ~/.local/bin/gsh

Then add ~/.local/bin to your PATH if not already present:

    echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc

Alternatively, install via Homebrew:

    brew install robottwo/gsh/gsh

For more information, visit: https://github.com/robottwo/gsh_prime
EOF

# Create a symlink to /usr/local/bin for easy drag-and-drop installation idea
# (This is informational - users still need to use terminal)

# Create the DMG
echo "Creating temporary DMG..."
hdiutil create -srcfolder "${DMG_CONTENTS}" -volname "${VOLUME_NAME}" -fs HFS+ \
    -fsargs "-c c=64,a=16,e=16" -format UDRW -size 50m "${DMG_TEMP}"

echo "Converting to compressed DMG..."
hdiutil convert "${DMG_TEMP}" -format UDZO -imagekey zlib-level=9 -o "${DMG_FINAL}"

# Clean up temporary DMG
rm -f "${DMG_TEMP}"

echo "Created: ${DMG_FINAL}"

# Output the checksum
echo "SHA256: $(shasum -a 256 "${DMG_FINAL}" | awk '{print $1}')"
