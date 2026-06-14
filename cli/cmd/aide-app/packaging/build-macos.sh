#!/usr/bin/env bash
set -euo pipefail

# Builds Aide.app for macOS.
#
# Optional environment variables:
#   VERSION           Version string baked into the bundle (default: dev)
#   DEVELOPER_ID      "Developer ID Application: Name (TEAMID)" for real signing.
#                     If unset, the app is ad-hoc signed (runs locally only).
#   NOTARIZE_PROFILE  notarytool keychain profile name. If set (and DEVELOPER_ID
#                     is set), the app is notarized and stapled.
#   MAKE_DMG          If "1", also produce Aide-<version>.dmg.

VERSION="${VERSION:-dev}"
export MACOSX_DEPLOYMENT_TARGET="${MACOSX_DEPLOYMENT_TARGET:-11.0}"
BUILD_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_DIR="$(cd "$BUILD_DIR/../../.." && pwd)"
OUT_DIR="$BUILD_DIR/bin"
APP="$OUT_DIR/Aide.app"

echo "==> Building frontend"
( cd "$CLI_DIR/internal/agent/frontend" && npm run build )

echo "==> Building universal Go binary ($VERSION)"
rm -rf "$OUT_DIR"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

LDFLAGS="-X main.version=$VERSION -X aide/cli/internal/agent.Version=$VERSION"
build_arch() {
  local pkg="$1" prefix="$2" arch="$3" tags="${4:-}"
  echo "    - $prefix $arch"
  ( cd "$CLI_DIR" && CGO_ENABLED=1 GOOS=darwin GOARCH="$arch" \
      go build -tags "$tags" -ldflags "$LDFLAGS" -o "$OUT_DIR/$prefix-$arch" "$pkg" )
}
build_arch ./cmd/aide-app app arm64 production
build_arch ./cmd/aide-app app amd64 production

echo "==> Building universal CLI binary ($VERSION)"
build_arch ./cmd/aide cli arm64
build_arch ./cmd/aide cli amd64

echo "==> Creating universal binaries"
lipo -create -output "$APP/Contents/MacOS/Aide" "$OUT_DIR/app-arm64" "$OUT_DIR/app-amd64"
lipo -create -output "$APP/Contents/MacOS/aide-cli" "$OUT_DIR/cli-arm64" "$OUT_DIR/cli-amd64"
rm -f "$OUT_DIR/app-arm64" "$OUT_DIR/app-amd64" "$OUT_DIR/cli-arm64" "$OUT_DIR/cli-amd64"

echo "==> Writing Info.plist"
sed "s/__VERSION__/$VERSION/g" "$BUILD_DIR/Info.plist.tmpl" > "$APP/Contents/Info.plist"

if [[ -f "$BUILD_DIR/icon.icns" ]]; then
  cp "$BUILD_DIR/icon.icns" "$APP/Contents/Resources/icon.icns"
fi

echo "==> Code signing"
if [[ -n "${DEVELOPER_ID:-}" ]]; then
  codesign --force --deep --options runtime --timestamp \
    --entitlements "$BUILD_DIR/entitlements.plist" \
    --sign "$DEVELOPER_ID" "$APP"

  if [[ -n "${NOTARIZE_PROFILE:-}" ]]; then
    echo "==> Notarizing"
    ZIP="$OUT_DIR/Aide.zip"
    ditto -c -k --keepParent "$APP" "$ZIP"
    xcrun notarytool submit "$ZIP" --keychain-profile "$NOTARIZE_PROFILE" --wait
    xcrun stapler staple "$APP"
    rm -f "$ZIP"
  fi
else
  echo "    (ad-hoc signing — no DEVELOPER_ID set; app runs on this machine only)"
  codesign --force --deep --sign - "$APP"
fi

echo "==> Verifying"
codesign --verify --verbose "$APP" || true

if [[ "${MAKE_DMG:-0}" == "1" ]]; then
  echo "==> Creating DMG"
  DMG="$OUT_DIR/Aide-$VERSION.dmg"
  STAGE="$OUT_DIR/dmg"
  rm -rf "$STAGE"; mkdir -p "$STAGE"
  cp -R "$APP" "$STAGE/"
  ln -s /Applications "$STAGE/Applications"
  hdiutil create -volname "Aide" -srcfolder "$STAGE" -ov -format UDZO "$DMG"
  rm -rf "$STAGE"
  echo "    $DMG"
fi

echo "==> Done: $APP"
