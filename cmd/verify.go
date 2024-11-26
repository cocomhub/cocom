/*
Copyright © 2023 suixibing <suixibing@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/comic"
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
	pattern    string
	autoFix    bool
	workers    int
	reportPath string
	interval   time.Duration
}{}

func init() {
	rootCmd.AddCommand(comicVerifyCmd)

	// 添加子命令
	comicVerifyCmd.AddCommand(verifyStatusCmd)
	comicVerifyCmd.AddCommand(verifyCancelCmd)
	comicVerifyCmd.AddCommand(verifyReportCmd)
	comicVerifyCmd.AddCommand(verifyScheduleCmd)

	// 添加标志
	comicVerifyCmd.PersistentFlags().StringVarP(&verifyFlags.pattern, "pattern", "p", ".*", "匹配规则")
	comicVerifyCmd.PersistentFlags().BoolVarP(&verifyFlags.autoFix, "auto-fix", "f", false, "自动修复损坏的图片")
	comicVerifyCmd.PersistentFlags().IntVarP(&verifyFlags.workers, "workers", "w", 4, "并发工作协程数")
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
		err := service.StartVerify(ctx, verifyFlags.pattern, verifyFlags.autoFix)
		if err != nil {
			return fmt.Errorf("启动验证任务失败: %v", err)
		}

		// 等待任务完成
		progress := service.GetVerifyProgress()
		for progress.Progress < 100 {
			fmt.Printf("\r验证进度: %.2f%% (%d/%d), 损坏: %d, 已修复: %d",
				progress.Progress,
				progress.Checked,
				progress.Total,
				progress.Invalid,
				progress.Fixed,
			)
			time.Sleep(time.Second)
			progress = service.GetVerifyProgress()
		}
		fmt.Println()

		// 生成报告
		if verifyFlags.reportPath != "" {
			verifier := service.GetVerifier()
			if err := verifier.SaveReport(verifyFlags.reportPath); err != nil {
				return fmt.Errorf("保存报告失败: %v", err)
			}
			fmt.Printf("报告已保存到: %s\n", verifyFlags.reportPath)
		}

		return nil
	}
}

var verifyStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看验证进度",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := clog.NewTraceCtx("verify_status")
		service := getComicService(ctx)
		if service == nil {
			return fmt.Errorf("连接数据库失败")
		}

		progress := service.GetVerifyProgress()
		if progress == nil {
			return fmt.Errorf("没有正在进行的验证任务")
		}

		fmt.Printf("验证进度: %.2f%% (%d/%d)\n", progress.Progress, progress.Checked, progress.Total)
		fmt.Printf("损坏文件: %d\n", progress.Invalid)
		fmt.Printf("已修复: %d\n", progress.Fixed)
		fmt.Printf("开始时间: %v\n", progress.StartTime.Format("2006-01-02 15:04:05"))

		return nil
	},
}

var verifyCancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "取消验证任务",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := clog.NewTraceCtx("verify_cancel")
		service := getComicService(ctx)
		if service == nil {
			return fmt.Errorf("连接数据库失败")
		}

		service.CancelVerify()
		fmt.Println("已取消验证任务")
		return nil
	},
}

var verifyReportCmd = &cobra.Command{
	Use:   "report",
	Short: "查看验证报告",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := clog.NewTraceCtx("verify_report")
		service := getComicService(ctx)
		if service == nil {
			return fmt.Errorf("连接数据库失败")
		}

		verifier := service.GetVerifier()
		if verifier == nil {
			return fmt.Errorf("没有验证任务的记录")
		}

		// 打印报告
		verifier.PrintReport()

		// 保存报告
		if verifyFlags.reportPath != "" {
			if err := verifier.SaveReport(verifyFlags.reportPath); err != nil {
				return fmt.Errorf("保存报告失败: %v", err)
			}
			fmt.Printf("报告已保存到: %s\n", verifyFlags.reportPath)
		}

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

		// 启动定时检查
		cfg := comic.ScheduleConfig{
			Pattern:       verifyFlags.pattern,
			Interval:      verifyFlags.interval,
			AutoFix:       verifyFlags.autoFix,
			Concurrent:    verifyFlags.workers,
			RetryInterval: time.Second * 30,
			MaxRetries:    3,
			TimeWindow: []comic.TimeRange{
				{Start: "00:00", End: "06:00"}, // 默认在凌晨执行
			},
			Priority: 1,
			Timeout:  time.Hour,
		}

		err := service.StartScheduleVerify(ctx, cfg)
		if err != nil {
			return fmt.Errorf("启动定时检查失败: %v", err)
		}

		return nil
	},
}

func getComicService(ctx context.Context) *comic.Service {
	// 连接数据库
	client, err := mongo.Connect(ctx, nil)
	if err != nil {
		clog.Errorf(ctx, "连接数据库失败: %v", err)
		return nil
	}

	db := client.Database("comics")
	return comic.NewService(db)
}
