// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/cocomhub/cocom/pkg/comic"
	comicStorage "github.com/cocomhub/cocom/pkg/comic/storage"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/mongo"
)

var comicVerifyCmd = &cobra.Command{
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

  # 查看验证报告
  cocom verify report

  # 定时检查
  cocom verify schedule --pattern ".*" --interval 24h`,
}

var verifyFlags = struct {
	pattern    string        // 匹配规则
	autoFix    bool          // 自动修复
	workers    int32         // 并发数
	reportPath string        // 报告路径
	interval   time.Duration // 检查间隔
}{}

func init() {
	rootCmd.AddCommand(comicVerifyCmd)

	// 添加子命令
	comicVerifyCmd.AddCommand(verifyStatusCmd)
	comicVerifyCmd.AddCommand(verifyCancelCmd)
	comicVerifyCmd.AddCommand(verifyScheduleCmd)

	// 添加标志
	comicVerifyCmd.PersistentFlags().StringVarP(&verifyFlags.pattern, "pattern", "p", ".*", "匹配规则")
	comicVerifyCmd.PersistentFlags().BoolVarP(&verifyFlags.autoFix, "auto-fix", "f", false, "自动修复损坏的图片")
	comicVerifyCmd.PersistentFlags().Int32VarP(&verifyFlags.workers, "workers", "w", 4, "并发工作协程数")
	comicVerifyCmd.PersistentFlags().StringVarP(&verifyFlags.reportPath, "report", "r", "verify_report.json", "报告输出路径")
	comicVerifyCmd.PersistentFlags().DurationVarP(&verifyFlags.interval, "interval", "i", 24*time.Hour, "定时检查间隔")

	// 添加验证命令的执行函数
	comicVerifyCmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := clog.NewTraceCtx("verify")
		service := getComicService(ctx)
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
			return fmt.Errorf("启动验证任务失败: %v", err)
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
			return fmt.Errorf("验证任务执行失败: %v", err)
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

		ctx := clog.NewTraceCtx("verify_status")
		service := getComicService(ctx)
		if service == nil {
			return fmt.Errorf("连接数据库失败")
		}

		taskID := args[0]
		progress, err := service.GetVerifyProgress(ctx, taskID)
		if err != nil {
			return fmt.Errorf("获取验证进度失败: %v", err)
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

		ctx := clog.NewTraceCtx("verify_cancel")
		service := getComicService(ctx)
		if service == nil {
			return fmt.Errorf("连接数据库失败")
		}

		taskID := args[0]
		progress, err := service.GetVerifyProgress(ctx, taskID)
		if err != nil {
			return fmt.Errorf("获取验证进度失败: %v", err)
		}

		if progress == nil {
			return fmt.Errorf("没有正在进行的验证任务")
		}

		service.CancelVerifyTask(ctx, taskID)
		fmt.Println("已取消验证任务")
		return nil
	},
}

var verifyScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "启动定时检查",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := clog.NewTraceCtx("verify_schedule")
		service := getComicService(ctx)
		if service == nil {
			return fmt.Errorf("连接数据库失败")
		}

		err := service.StartScheduleVerify(ctx, &comic.ScheduleConfig{
			Pattern:       verifyFlags.pattern,
			Interval:      verifyFlags.interval,
			AutoFix:       verifyFlags.autoFix,
			RetryInterval: time.Second * 30,
			Active:        true,
			MaxRetry:      3,
			RetryWait:     time.Second * 30,
			Options: &comic.VerifyOptions{
				ComicFilter: comic.ComicFilter{
					TitlePattern: &verifyFlags.pattern,
				},
				AutoFix:    verifyFlags.autoFix,
				MaxWorkers: verifyFlags.workers,
			},
		})
		if err != nil {
			return fmt.Errorf("启动定时检查失败: %v", err)
		}

		fmt.Printf("定时检查已启动，间隔: %v\n", verifyFlags.interval)
		return nil
	},
}

func getComicService(ctx context.Context) comic.Service {
	// 连接数据库
	client, err := mongo.Connect(ctx, nil)
	if err != nil {
		clog.Errorf(ctx, "连接数据库失败: %v", err)
		return nil
	}

	// 创建服务实例
	service, err := comic.NewService(ctx, comicStorage.NewMongoStorage(client.Database("")))
	if err != nil {
		clog.Errorf(ctx, "创建服务实例失败: %v", err)
		return nil
	}

	return service
}
