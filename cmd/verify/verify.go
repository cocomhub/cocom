// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/pkg/comic"
	comicStorage "github.com/cocomhub/cocom/pkg/comic/storage"
	"github.com/cocomhub/cocom/pkg/logging"
	"github.com/cocomhub/cocom/pkg/mongowrap"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "verify",
	Short: "验证漫画图片完整性",
	Long: `验证漫画图片完整性，支持以下功能：

1. 验证图片是否完整可用
2. 自动修复损坏的图片
3. 支持断点续传
4. 支持优先级队列
5. 支持定时检查
6. 支持性能监控
7. 支持结果汇总和报告

使用示例：
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

  # 定时检查
  cocom verify schedule --pattern ".*" --interval 24h`,
}

var verifyFlags = struct {
	pattern  string        // 匹配规则
	autoFix  bool          // 自动修复
	workers  int32         // 并发数
	interval time.Duration // 检查间隔
}{}

// GetComicService 创建 comic.Service 实例（可被测试覆盖）
var GetComicService func(ctx context.Context) comic.Service

func init() {
	GetComicService = getComicServiceDefault

	// root registration handled in cmd/root.go

	// 添加子命令
	Cmd.AddCommand(verifyStatusCmd)
	Cmd.AddCommand(verifyCancelCmd)
	Cmd.AddCommand(verifyScheduleCmd)

	// 添加标志
	Cmd.PersistentFlags().StringVarP(&verifyFlags.pattern, "pattern", "p", ".*", "匹配规则")
	Cmd.PersistentFlags().BoolVarP(&verifyFlags.autoFix, "auto-fix", "f", false, "自动修复损坏的图片")
	Cmd.PersistentFlags().Int32VarP(&verifyFlags.workers, "workers", "w", 4, "并发工作协程数")
	Cmd.PersistentFlags().DurationVarP(&verifyFlags.interval, "interval", "i", 24*time.Hour, "定时检查间隔")

	// 添加验证命令的执行函数
	Cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := logging.NewTraceCtx("verify")
		service := GetComicService(ctx)
		if service == nil {
			return fmt.Errorf("连接数据库失败")
		}

		// 启动验证任务
		taskID, err := service.StartVerifyTask(ctx, &comic.VerifyOptions{
			ComicFilter: comic.ComicFilter{
				TitlePattern: &verifyFlags.pattern,
			},
			AutoFix:    verifyFlags.autoFix,
			MaxWorkers: verifyFlags.workers,
		})
		if err != nil {
			return fmt.Errorf("启动验证任务失败: %w", err)
		}

		// 等待任务完成并显示进度
		progress, err := service.GetVerifyProgress(ctx, taskID)
		for err == nil && !progress.IsCompleted() {
			fmt.Printf("\r验证进度: %.2f%% (%d/%d), 损坏: %d, 已修复: %d",
				progress.GetProgress(),
				progress.Current.Load(),
				progress.Total.Load(),
				progress.Invalid.Load(),
				progress.Fixed.Load())
			time.Sleep(time.Second)
			progress, err = service.GetVerifyProgress(ctx, taskID)
		}
		fmt.Println()

		if err != nil {
			return fmt.Errorf("验证任务执行失败: %w", err)
		}

		// 输出最终结果
		fmt.Printf("验证完成，共处理 %d 个文件，发现 %d 个损坏文件，修复 %d 个文件\n",
			progress.Total.Load(),
			progress.Invalid.Load(),
			progress.Fixed.Load())

		return nil
	}
}

var verifyStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看验证进度",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("无效的参数")
		}

		ctx := logging.NewTraceCtx("verify_status")
		service := GetComicService(ctx)
		if service == nil {
			return fmt.Errorf("连接数据库失败")
		}

		taskID := args[0]
		progress, err := service.GetVerifyProgress(ctx, taskID)
		if err != nil {
			return fmt.Errorf("获取验证进度失败: %w", err)
		}

		fmt.Printf("验证进度: %.2f%% (%d/%d)\n", progress.GetProgress(),
			progress.Current.Load(), progress.Total.Load())
		fmt.Printf("无效图片: %d\n", progress.Invalid.Load())
		fmt.Printf("已修复图片: %d\n", progress.Fixed.Load())
		fmt.Printf("开始时间: %v\n", progress.StartTime.Format("2006-01-02 15:04:05"))

		return nil
	},
}

var verifyCancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "取消验证任务",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("无效的参数")
		}

		ctx := logging.NewTraceCtx("verify_cancel")
		service := GetComicService(ctx)
		if service == nil {
			return fmt.Errorf("连接数据库失败")
		}

		taskID := args[0]
		progress, err := service.GetVerifyProgress(ctx, taskID)
		if err != nil {
			return fmt.Errorf("获取验证进度失败: %w", err)
		}

		if progress == nil {
			return fmt.Errorf("没有正在进行的验证任务")
		}

		_ = service.CancelVerifyTask(ctx, taskID)
		fmt.Println("已取消验证任务")
		return nil
	},
}

var verifyScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "启动定时检查",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logging.NewTraceCtx("verify_schedule")
		service := GetComicService(ctx)
		if service == nil {
			return fmt.Errorf("连接数据库失败")
		}

		err := service.StartScheduleVerify(ctx, &comic.ScheduleConfig{
			Pattern:  verifyFlags.pattern,
			Interval: verifyFlags.interval,
			AutoFix:  verifyFlags.autoFix,
			Active:   true,
			Options: &comic.VerifyOptions{
				ComicFilter: comic.ComicFilter{
					TitlePattern: &verifyFlags.pattern,
				},
				AutoFix:    verifyFlags.autoFix,
				MaxWorkers: verifyFlags.workers,
			},
		})
		if err != nil {
			return fmt.Errorf("启动定时检查失败: %w", err)
		}

		fmt.Printf("定时检查已启动，间隔: %v（前台驻留，Ctrl+C 退出）\n", verifyFlags.interval)

		// 保持进程驻留：此前 RunE 直接返回，进程立即退出，定时任务根本不会触发。
		sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		<-sigCtx.Done()

		if closer, ok := service.(interface{ Close() error }); ok {
			if closeErr := closer.Close(); closeErr != nil {
				slog.WarnContext(ctx, "关闭验证服务失败", slog.String("errmsg", closeErr.Error()))
			}
		}
		fmt.Println("定时检查已停止")
		return nil
	},
}

func getComicServiceDefault(ctx context.Context) comic.Service {
	// 使用统一配置连接的 mongowrap：此前 mongo.Connect(ctx, nil) 用的是驱动默认地址
	// （忽略 mongo.host/user/password/authSource），再配 client.Database("") 空库名 ——
	// 会连错库/集合并静默返回空数据。库名同样取自配置（comic.mongo.database）。
	if _, err := mongowrap.Client(); err != nil {
		slog.ErrorContext(ctx, "连接数据库失败", slog.String("errmsg", err.Error()))
		return nil
	}
	db, err := mongowrap.DB(config.Get().Comic.Mongo.Database)
	if err != nil {
		slog.ErrorContext(ctx, "获取数据库失败", slog.String("errmsg", err.Error()))
		return nil
	}

	service, err := comic.NewService(ctx, comicStorage.NewMongoStorage(db), config.Get().Download.DownloadDir)
	if err != nil {
		slog.ErrorContext(ctx, "创建服务实例失败", slog.String("errmsg", err.Error()))
		return nil
	}

	return service
}
