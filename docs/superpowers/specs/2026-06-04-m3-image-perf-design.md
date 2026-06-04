# M3：图片浏览与性能优化 — 设计规格

> 基于 2026-06-04 路线图设计的 M3 阶段，范围：cocom 子项目。

---

## 1. 响应式图片服务（运行时缩放）

### 行为

- `GET /galleries/{cid}/{name}?w=200` 支持 `w` 参数，将图片缩放到指定宽度（保持宽高比）
- 无 `w` 参数时返回原始图片（保持现有行为）
- 缩放结果直接写入响应（不落盘）
- 输出格式与输入格式相同（JPEG → JPEG, PNG → PNG）

### 不可缩放格式降级

- WebP 格式：`pkg/imaging` 依赖 `cwebp` CLI，解码可能失败。检测到无法处理的格式时，打 warn 日志并返回原始图片
- 其他不可解码格式类似处理

### 涉及文件

- 修改：`cmd/server/view/picture.go`（`Picture` handler）

---

## 2. 浏览器缓存策略

### 行为

- 为 `/galleries/` 图片响应设置缓存头：
  - `Cache-Control: public, max-age=31536000`（1 年）
  - `ETag: "{name}-{mtime}"`（含 `w` 参数时：`"{name}-w{w}-{mtime}"`）
  - `Last-Modified` 基于文件修改时间

### 涉及文件

- 修改：`cmd/server/view/picture.go`

---

## 验收标准

- [ ] `GET /galleries/{cid}/1.jpg?w=200` 返回宽度为 200px 的缩放图片
- [ ] `GET /galleries/{cid}/1.jpg` 返回原始图片（未缩放）
- [ ] 图片响应包含 `Cache-Control: public, max-age=31536000` 头
- [ ] WebP 等不可缩放格式返回原始图片并打印 warn 日志
- [ ] `make build` 编译通过

---

## 不包含的范围

- 大图模式键盘翻页（已跳过）
- 虚拟滚动（已跳过）
- 图片预压缩（已跳过）
- 懒加载改进（保持现状）
- 缩略图渐进加载（LQIP，推迟）