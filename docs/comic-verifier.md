# 漫画验证功能说明文档

## 功能特性

1. 图片完整性验证
   - 支持多种图片格式
   - 验证图片是否损坏
   - 检查图片尺寸和大小
   - 记录验证结果

2. 自动修复功能
   - 自动重新下载损坏图片
   - 支持断点续传
   - 记录修复结果
   - 保持原有文件名

3. 断点续传
   - 记录最后检查位置
   - 支持状态保存和恢复
   - 自动从断点继续
   - 避免重复验证

4. 优先级队列
   - 按最后验证时间排序
   - 支持并发安全操作
   - 动态调整优先级
   - 优化验证顺序

5. 定时检查
   - 支持定期自动验证
   - 可配置检查间隔
   - 支持自动修复
   - 记录检查历史

6. 性能监控
   - 实时监控处理速度
   - 记录内存使用情况
   - 统计处理文件数量
   - 输出性能报告

7. 结果汇总和报告
   - JSON 格式报告
   - 控制台打印报告
   - 详细的验证结果
   - 支持结果导出

## 命令行使用

### 基本命令

```bash
# 验证所有漫画
cocom verify --pattern ".*"

# 自动修复损坏的图片
cocom verify --pattern ".*" --auto-fix

# 指定并发数
cocom verify --pattern ".*" --workers 4

# 查看验证进度
cocom verify status

# 取消验证任务
cocom verify cancel

# 查看验证报告
cocom verify report

# 定时检查
cocom verify schedule --pattern ".*" --interval 24h
```

### 命令参数

全局参数：
- `-p, --pattern`: 匹配规则，支持正则表达式
- `-f, --auto-fix`: 启用自动修复功能
- `-w, --workers`: 并发工作协程数 (默认: 4)
- `-r, --report`: 报告输出路径
- `-i, --interval`: 定时检查间隔

### 验证报告格式

```json
{
  "start_time": "2024-11-27T10:00:00Z",
  "end_time": "2024-11-27T11:00:00Z",
  "duration": "1h0m0s",
  "pattern": ".*",
  "total_comics": 100,
  "valid_comics": 90,
  "invalid_comics": 10,
  "fixed_comics": 8,
  "total_files": 1000,
  "valid_files": 950,
  "invalid_files": 50,
  "fixed_files": 40,
  "processed_mb": 1024.5,
  "average_speed": 17.1,
  "results": [
    {
      "comic_id": "123",
      "title": "示例漫画",
      "images": [
        {
          "path": "/path/to/1.jpg",
          "url": "http://example.com/1.jpg",
          "invalid": false,
          "info": {
            "format": "jpeg",
            "width": 1024,
            "height": 1456,
            "size": 229394
          }
        }
      ],
      "invalid_count": 0,
      "fixed_count": 0,
      "timestamp": "2024-11-27T10:30:00Z"
    }
  ]
}
```

### 性能监控指标

```json
{
  "start_time": "2024-11-27T10:00:00Z",
  "duration": "1h0m0s",
  "num_goroutine": 5,
  "num_cpu": 8,
  "mem_stats": {
    "alloc": 10485760,
    "total_alloc": 104857600,
    "sys": 73400320,
    "num_gc": 10
  },
  "processed_mb": 1024.5,
  "average_speed": 17.1,
  "current_speed": 16.8,
  "total_files": 1000,
  "processed_files": 950,
  "failed_files": 50
}
```

## 错误处理

1. 验证错误
   - 文件不存在
   - 格式不支持
   - 图片损坏
   - 下载失败

2. 数据库错误
   - 连接失败
   - 查询错误
   - 更新错误
   - 事务错误

3. 系统错误
   - 内存不足
   - 磁盘空间不足
   - 权限不足
   - 网络错误

## 最佳实践

1. 定期验证
   - 每天凌晨执行
   - 错峰检查
   - 分批处理
   - 记录历史

2. 性能优化
   - 合理设置并发数
   - 控制内存使用
   - 优化查询性能
   - 使用索引

3. 错误处理
   - 记录详细日志
   - 设置重试机制
   - 保存错误信息
   - 通知管理员

4. 数据备份
   - 定期备份数据
   - 保存验证结果
   - 备份配置信息
   - 维护历史记录