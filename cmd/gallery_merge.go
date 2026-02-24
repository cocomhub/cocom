package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type mergeGalleryConfig struct {
	TargetRelative string
	MergeDir       string
	VolumeBase     string
	DryRun         bool
	Verbose        bool
	ExcludeVolumes []string
}

type directoryInfo struct {
	Path    string
	ModTime time.Time
	Volume  string
}

type mergeStats struct {
	Count           int
	LatestDirectory *directoryInfo
	AllDirectories  []*directoryInfo
}

var mergeGalleryFlags = struct {
	output  string
	target  string
	volumes string
	dryRun  bool
	verbose bool
	exclude string
}{}

var mergeGalleryCmd = &cobra.Command{
	Use:   "merge",
	Short: "合并各卷的 gallery 目录为统一软链接视图",
	Long: `扫描所有挂载卷下的目标 gallery 目录，按子目录名合并，选择最新的目录作为源，
在指定的合并目录下创建指向源目录的软链接，支持排除卷、模拟执行等。

示例：
  # 默认扫描 /Volumes/*/cocom/data/gallery 并在 /opt/cocom/data/gallery 生成软链接
  cocom gallery merge

  # 自定义目标相对路径与合并输出目录
  cocom gallery merge --target cocom/data/gallery --output /opt/cocom/data/gallery

  # 仅模拟执行不真正创建
  cocom gallery merge --dry-run

  # 排除某些卷
  cocom gallery merge --exclude "Macintosh HD,Time Machine"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := &mergeGalleryConfig{
			TargetRelative: mergeGalleryFlags.target,
			MergeDir:       mergeGalleryFlags.output,
			VolumeBase:     mergeGalleryFlags.volumes,
			DryRun:         mergeGalleryFlags.dryRun,
			Verbose:        mergeGalleryFlags.verbose,
			ExcludeVolumes: splitAndTrim(mergeGalleryFlags.exclude),
		}
		if cfg.DryRun {
			fmt.Println("=== 模拟运行模式 ===")
		}
		return runMergeGallery(cfg)
	},
}

func init() {
	galleryCmd.AddCommand(mergeGalleryCmd)

	mergeGalleryCmd.Flags().StringVarP(&mergeGalleryFlags.output, "output", "o", "/opt/cocom/data/gallery", "合并后的软链接目录")
	mergeGalleryCmd.Flags().StringVarP(&mergeGalleryFlags.target, "target", "t", "cocom/data/gallery", "各卷上的目标相对路径")
	mergeGalleryCmd.Flags().StringVarP(&mergeGalleryFlags.volumes, "volumes", "v", "/Volumes", "卷挂载基础路径")
	mergeGalleryCmd.Flags().BoolVar(&mergeGalleryFlags.dryRun, "dry-run", false, "模拟执行，不实际创建或删除")
	mergeGalleryCmd.Flags().BoolVarP(&mergeGalleryFlags.verbose, "verbose", "V", false, "显示详细输出")
	mergeGalleryCmd.Flags().StringVarP(&mergeGalleryFlags.exclude, "exclude", "e", "Macintosh HD", "排除的卷名，逗号分隔")
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func runMergeGallery(config *mergeGalleryConfig) error {
	fmt.Printf("开始扫描挂载卷中的目标目录...\n")
	fmt.Printf("卷基础路径: %s\n", config.VolumeBase)
	fmt.Printf("目标目录模式: %s/*/%s\n", config.VolumeBase, config.TargetRelative)
	fmt.Printf("合并目录: %s\n", config.MergeDir)
	if config.DryRun {
		fmt.Printf("模式: 模拟运行\n")
	}
	fmt.Println()

	statsMap := make(map[string]*mergeStats)
	excluded := make(map[string]bool)

	for _, vol := range config.ExcludeVolumes {
		excluded[vol] = true
	}

	volumes, err := os.ReadDir(config.VolumeBase)
	if err != nil {
		return fmt.Errorf("无法读取 %s 目录: %v", config.VolumeBase, err)
	}

	for _, volumeEntry := range volumes {
		volumeName := volumeEntry.Name()
		if volumeName == "." || volumeName == ".." || excluded[volumeName] {
			if config.Verbose {
				fmt.Printf("跳过: %s\n", volumeName)
			}
			continue
		}

		volumePath := filepath.Join(config.VolumeBase, volumeName)
		volumeInfo, err := os.Stat(volumePath)
		if err != nil || !volumeInfo.IsDir() {
			continue
		}

		targetPath := filepath.Join(volumePath, config.TargetRelative)
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			continue
		}

		fmt.Printf("发现: %s\n", targetPath)
		processVolumeForMerge(targetPath, volumeName, statsMap, config)
	}

	if len(statsMap) == 0 {
		fmt.Println("未找到任何目标目录")
		return nil
	}

	printMergeStatistics(statsMap)
	createMergeLinks(statsMap, config)
	return nil
}

func processVolumeForMerge(targetPath, volumeName string, statsMap map[string]*mergeStats, config *mergeGalleryConfig) {
	dir, err := os.Open(targetPath)
	if err != nil {
		if config.Verbose {
			fmt.Printf("警告: 无法打开目录 %s: %v\n", targetPath, err)
		}
		return
	}
	defer dir.Close()

	entries, err := dir.ReadDir(-1)
	if err != nil {
		if config.Verbose {
			fmt.Printf("警告: 无法读取目录 %s: %v\n", targetPath, err)
		}
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subdirName := entry.Name()
		subdirPath := filepath.Join(targetPath, subdirName)

		info, err := entry.Info()
		if err != nil {
			if config.Verbose {
				fmt.Printf("警告: 无法获取目录信息 %s: %v\n", subdirPath, err)
			}
			continue
		}

		dirInfo := &directoryInfo{
			Path:    subdirPath,
			ModTime: info.ModTime(),
			Volume:  volumeName,
		}

		if stats, exists := statsMap[subdirName]; exists {
			stats.Count++
			if dirInfo.ModTime.After(stats.LatestDirectory.ModTime) {
				stats.LatestDirectory = dirInfo
			}
			stats.AllDirectories = append(stats.AllDirectories, dirInfo)
		} else {
			statsMap[subdirName] = &mergeStats{
				Count:           1,
				LatestDirectory: dirInfo,
				AllDirectories:  []*directoryInfo{dirInfo},
			}
		}
	}
}

func printMergeStatistics(statsMap map[string]*mergeStats) {
	fmt.Println("\n========== 统计结果 ==========")
	dirNames := make([]string, 0, len(statsMap))
	for dirName := range statsMap {
		dirNames = append(dirNames, dirName)
	}
	sort.Strings(dirNames)

	for _, dirName := range dirNames {
		stats := statsMap[dirName]
		fmt.Printf("%-20s: 出现 %2d 次 (最新: %s)\n",
			dirName, stats.Count, stats.LatestDirectory.Volume)
	}
	fmt.Printf("\n总计: %d 个子目录\n", len(statsMap))
}

func createMergeLinks(statsMap map[string]*mergeStats, config *mergeGalleryConfig) {
	fmt.Println("\n========== 准备创建软链接 ==========")

	if config.DryRun {
		fmt.Println("模拟运行 - 不会实际创建文件")
	} else {
		if err := os.MkdirAll(config.MergeDir, 0o755); err != nil {
			log.Fatalf("错误: 无法创建合并目录 %s: %v", config.MergeDir, err)
		}
	}

	created := 0
	skipped := 0
	dirNames := make([]string, 0, len(statsMap))
	for dirName := range statsMap {
		dirNames = append(dirNames, dirName)
	}
	sort.Strings(dirNames)

	for _, dirName := range dirNames {
		stats := statsMap[dirName]
		latestDir := stats.LatestDirectory
		linkPath := filepath.Join(config.MergeDir, dirName)
		targetPath := latestDir.Path

		if _, err := os.Lstat(linkPath); err == nil {
			if config.DryRun {
				fmt.Printf("将移除: %s\n", linkPath)
			} else {
				if err := os.RemoveAll(linkPath); err != nil {
					fmt.Printf("警告: 无法移除 %s: %v\n", linkPath, err)
					skipped++
					continue
				}
				fmt.Printf("已移除: %s\n", linkPath)
			}
		}

		if config.DryRun {
			fmt.Printf("将创建: %s -> %s\n", linkPath, targetPath)
			created++
		} else {
			if err := os.Symlink(targetPath, linkPath); err != nil {
				fmt.Printf("失败: %s -> %s: %v\n", linkPath, targetPath, err)
				skipped++
				continue
			}
			fmt.Printf("已创建: %s -> %s\n", linkPath, targetPath)
			created++
		}
	}

	fmt.Println("\n========== 完成 ==========")
	fmt.Printf("成功创建: %d 个软链接\n", created)
	fmt.Printf("失败: %d 个\n", skipped)

	if !config.DryRun {
		fmt.Printf("合并目录: %s\n", config.MergeDir)
		fmt.Printf("使用 'ls -la %s' 查看创建的软链接\n", config.MergeDir)
	}
}
