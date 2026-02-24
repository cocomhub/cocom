# install 命令

- 安装依赖组件，目前支持：`webp`（cwebp、dwebp） `cmd/install.go:13-18`
- 用法：`cocom install webp` 不同平台实现：
  - macOS（Homebrew）`cmd/install.go:60-70`
  - Linux（apt/yum/dnf）`cmd/install.go:72-90`
  - Windows（Chocolatey）`cmd/install.go:92-103`
