* [x] 已完成安全审计文档（docs/security/audit-archive-storage.md），列出高/中/低风险与整改状态

* [x] LocalFS 路径越界与软链逃逸测试全部通过（拒绝访问且返回安全错误）

* [x] 写入采用原子替换策略验证通过，异常中断不产生半写文件

* [x] 统一错误映射生效：NotFound/Conflict/Transient/PolicyViolation 等语义一致

* [x] Storage 常规与边界用例测试通过（0 字节、大文件、特殊/Unicode 名称、超长路径）

* [x] Storage 并发用例通过且 -race 无数据竞争

* [x] Manager 与 Archive 测试通过：CRUD、幂等、Replicate 成功/失败/重试、一致性检查报告、Retention

* [x] URI/路径属性与 fuzz 测试无崩溃、不变量满足

* [x] go test -race 针对上述三个包通过

* [x] README/SECURITY 已更新，包含 URI 规范、沙箱说明与测试矩阵
