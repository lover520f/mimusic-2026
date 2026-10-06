# 客户端选择

Songloft 提供 Flutter 与 Lynx 两套播放器客户端；它们连接同一后端，发布包和能力范围分别维护。

| 项目            | Flutter                                  | Lynx（预览版）                       |
| --------------- | ---------------------------------------- | ------------------------------------ |
| 技术            | Flutter / Dart                           | ReactLynx / TypeScript               |
| 平台            | Android、iOS、macOS、Windows、Linux、Web | Android、iOS、HarmonyOS、Web         |
| Bundle 本地后端 | 支持移动端与桌面端                       | 未实现，所有平台需独立服务器         |
| 桌面客户端      | 支持                                     | 未实现                               |
| 客户端更新      | 以 Flutter 客户端指南为准                | 重新下载并安装，尚无客户端内检查更新 |
| 核心功能        | 播放、曲库、歌单、歌词、插件、主题等     | 已有对应实现，仍需原生设备回归       |
| 许可证          | Apache-2.0                               | Apache-2.0                           |

## Flutter

- [标准版下载](https://github.com/songloft-org/songloft-player/releases/latest)：连接独立服务器。
- [Bundle 下载](https://github.com/songloft-org/songloft/releases/latest)：选择 `songloft-bundled-*` 文件。
- [架构](player/architecture.md) · [构建](player/build_guide.md) · [源码](https://github.com/songloft-org/songloft-player)。

后端完整镜像与二进制默认嵌入 Flutter Web；加入 Lynx 下载入口不会自动更换内嵌界面。

## Lynx 预览版

底层技术：[Lynx 官网](https://lynxjs.org/) · [ReactLynx 官方文档](https://lynxjs.org/react/)。

- [开发版下载](https://github.com/songloft-org/songloft-player-lynx/releases/tag/dev)：main 代码推送后构建，五个包全部成功才更新。
- [正式版本](https://github.com/songloft-org/songloft-player-lynx/releases/latest)：版本 tag 触发；首次成功发布前可能没有正式包。
- [客户端概览](player-lynx/index.md) · [安装](player-lynx/installation.md) · [构建](player-lynx/build-and-run.md) · [测试](player-lynx/testing.md) · [发版](player-lynx/releasing.md) · [贡献](player-lynx/contributing.md)。

| 平台                     | 产物                                             | 使用边界                                                |
| ------------------------ | ------------------------------------------------ | ------------------------------------------------------- |
| Android 5.0+             | `songloft-lynx-android.apk`                      | release 签名，覆盖升级需相同签名与较新构建号            |
| iOS 15+                  | `songloft-lynx-ios-nosign.ipa`                   | 未签名，安装前需自行重签；Live Activity 需 iOS 16.2+    |
| HarmonyOS NEXT / API 13+ | `songloft-lynx-harmony.hap`                      | 实验性；签名 profile 决定可安装设备范围，全屏视频未实现 |
| Web                      | `songloft-lynx-web-{standalone,embedded}.tar.gz` | HTTPS 与 COOP/COEP 必需，无 Bundle、DLNA 或单曲离线缓存 |

每个 Release 同时提供 `version.json` 与 `checksums.txt`。dev 下载包同样是 Release 构建，不开放 TCP/E2E 测试桥。

Web standalone 显示服务器地址输入；embedded 使用同源服务器并隐藏该输入。两者都需要正确的 MIME 与隔离响应头；替换 Go 的 Flutter 内嵌资源并不能自动满足 Lynx 宿主契约。部署请参阅 [Web 指南](player-lynx/web-deployment.md)，子路径部署目前未验证。

## 发布与文档维护

Lynx 在自己的仓库使用 `pnpm run release patch --dry-run` 审核发版计划，实际发版同步版本并原子推送 main 与版本 tag。无需为客户端发版同步修改服务端版本。

本站的 Lynx 公共指南从 `clients/player-lynx` 子模块自动同步，中文在 `/player-lynx/`、英文在 `/en/player-lynx/`。编辑源仓库、提交推送后，在主工程更新子模块指针；不要手改生成的文档站页面。平台编译和签名结果以该仓库 Actions/Release 为准，后台播放、通知、投屏等还需真机验证。
