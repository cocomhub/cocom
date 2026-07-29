# Phase 1: 扫描 v0.0.57 配置链路

## 目标

全面扫描 v0.0.57 版本中所有配置键的完整链路：SetDefault 定义位置 → 读取方 → 传导路径 → 最终用途。

## 方法

1. 从 v0.0.57 tag 检出，扫描所有 `viper.SetDefault` 调用，列出所有配置键及其默认值
2. 对于每个配置键，搜索 `viper.Get*` 消费方，记录每个消费方的文件路径和行号
3. 对每个消费方，追踪配置值的使用路径（参数传递、结构体字段、全局变量等）
4. 记录每个配置键的最终用途（业务目的）

## 输出文件

将结果写入 `D:\workdir\leon\cocomhub\cocom\.superpowers\sdd\phase1-v0.0.57-config-inventory.md`

## 输出格式

```markdown
# v0.0.57 配置键清单

## 配置键: `<key-path>`
- 默认值: `<value>`
- 定义位置: `<file:line>`
- 消费方:
  - `<file:line>`: `<usage description>`
  - `<file:line>`: `<usage description>`
- 传导路径: `<how the value flows from viper to final use>`
- 最终用途: `<business purpose>`
```

## 约束

- 不要修改任何文件
- 仅基于 v0.0.57 tag 的代码输出做判断
- 避免幻觉