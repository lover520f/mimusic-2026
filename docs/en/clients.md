# Choosing a client

Songloft offers Flutter and Lynx players connecting to the same backend. Their packages and capability scopes are maintained separately.

| Item                  | Flutter                                               | Lynx (preview)                                            |
| --------------------- | ----------------------------------------------------- | --------------------------------------------------------- |
| Technology            | Flutter / Dart                                        | ReactLynx / TypeScript                                    |
| Platforms             | Android, iOS, macOS, Windows, Linux, Web              | Android, iOS, HarmonyOS, Web                              |
| Bundled local backend | Supported on mobile and desktop                       | Not implemented; all platforms require a separate server  |
| Desktop client        | Supported                                             | Not implemented                                           |
| Client updates        | See Flutter's client guides                           | Download and install again; no in-app update checks yet   |
| Core features         | Playback, library, playlists, lyrics, plugins, themes | Implemented, with native device regression still required |
| License               | Apache-2.0                                            | Apache-2.0                                                |

## Flutter

- [Standalone downloads](https://github.com/songloft-org/songloft-player/releases/latest): connect to a separate server.
- [Bundle downloads](https://github.com/songloft-org/songloft/releases/latest): choose `songloft-bundled-*` assets.
- [Architecture (Chinese)](../player/architecture.md) · [Build guide (Chinese)](../player/build_guide.md) · [Source](https://github.com/songloft-org/songloft-player).

Full backend images/binaries embed Flutter Web by default. Adding a Lynx download option does not switch the bundled interface.

## Lynx preview

Underlying technology: [Lynx official website](https://lynxjs.org/) · [ReactLynx documentation](https://lynxjs.org/react/).

- [Development downloads](https://github.com/songloft-org/songloft-player-lynx/releases/tag/dev): main code pushes build automatically; all five packages must pass before downloads update.
- [Stable releases](https://github.com/songloft-org/songloft-player-lynx/releases/latest): version tags trigger publication; packages may not exist before the first successful release.
- [Overview](player-lynx/index.md) · [Installation](player-lynx/installation.md) · [Build](player-lynx/build-and-run.md) · [Testing](player-lynx/testing.md) · [Releasing](player-lynx/releasing.md) · [Contributing](player-lynx/contributing.md).

| Platform                 | Package                                          | Limits                                                                      |
| ------------------------ | ------------------------------------------------ | --------------------------------------------------------------------------- |
| Android 5.0+             | `songloft-lynx-android.apk`                      | Release-signed; upgrades require the same key and a newer build number      |
| iOS 15+                  | `songloft-lynx-ios-nosign.ipa`                   | Unsigned; re-sign before installation. Live Activity needs iOS 16.2+        |
| HarmonyOS NEXT / API 13+ | `songloft-lynx-harmony.hap`                      | Experimental; profile controls eligible devices. No fullscreen video        |
| Web                      | `songloft-lynx-web-{standalone,embedded}.tar.gz` | Requires HTTPS and COOP/COEP; no Bundle, DLNA, or single-song offline cache |

Releases include `version.json` and `checksums.txt`. Downloadable dev packages are also Release builds with no TCP/E2E bridge.

Standalone Web shows a server-address field; embedded uses the same-origin server and hides that field. Both require suitable MIME types and isolation headers. Replacing Go's embedded Flutter assets does not automatically satisfy the Lynx host contract. See [Web deployment](player-lynx/web-deployment.md); subpath deployment is unverified.

## Releases and documentation upkeep

Preview a Lynx release in its own repository with `pnpm run release patch --dry-run`. Actual release synchronizes versions and atomically pushes main and its version tag. A client release does not require changing the server version.

Public guides are synced from the `clients/player-lynx` submodule to `/player-lynx/` and `/en/player-lynx/`. Edit, commit, and push the source repository, then update its submodule pointer in the parent. Do not edit generated site pages. Actions/Release are authoritative for builds and signing; background playback, notifications, and casting also need device validation.
