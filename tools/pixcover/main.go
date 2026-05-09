// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cocomhub/cocom/internal/rootcli"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// 配置参数
type Config struct {
	InputDir      string
	DownloadDir   string
	MongoURI      string
	Database      string
	Collection    string
	PageSize      int
	MaxSizeGB     int
	MaxTotalSize  int64
	LatestPIDFile string
	FailFile      string
	SuccessFile   string
	StartTime     time.Time
	CreateIndex   bool
}

// 下载管理器
type DownloadManager struct {
	config        *Config
	existingFiles map[string]bool
	downloaded    map[string]bool
	totalSize     int64
	successFile   *os.File
	failFile      *os.File
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	client        *mongo.Client
	latestPID     int
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var (
	cfg     Config
	rootCmd = &cobra.Command{
		Use:   "pixcover",
		Short: "A brief description of your application",
		Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application.`,
		Run: func(cmd *cobra.Command, args []string) {
			executeApp()
		},
	}
)

func init() {
	cobra.OnInitialize(
		rootcli.InitConfig,
	)

	rootcli.InitRootCmd(rootCmd)

	rootCmd.Flags().StringVarP(&cfg.InputDir, "input", "i", "/data/comic/input", "输入目录路径")
	rootCmd.Flags().StringVarP(&cfg.DownloadDir, "download", "d", "/data/comic/pixiv", "下载目录路径")
	rootCmd.Flags().StringVarP(&cfg.MongoURI, "mongo", "m", "mongodb://comic:HxYJdyTRxDLhGtSW@localhost:27017/comic", "MongoDB连接URI")
	rootCmd.Flags().StringVar(&cfg.Database, "db", "comic", "数据库名")
	rootCmd.Flags().StringVar(&cfg.Collection, "collection", "pixivInfo", "集合名")
	rootCmd.Flags().IntVar(&cfg.PageSize, "pagesize", 100, "分页大小")
	rootCmd.Flags().IntVarP(&cfg.MaxSizeGB, "maxsize", "s", 10, "最大下载大小(GB)")
	rootCmd.Flags().StringVar(&cfg.LatestPIDFile, "pidfile", "latest-pid", "最新PID记录文件")
	rootCmd.Flags().StringVar(&cfg.FailFile, "failfile", "fail.txt", "失败记录文件")
	rootCmd.Flags().BoolVar(&cfg.CreateIndex, "create-index", false, "是否为pid字段创建索引")
}

func executeApp() {
	// 计算最大总大小 (必须在 executeApp 中，因为 Flags 刚刚解析)
	cfg.MaxTotalSize = int64(cfg.MaxSizeGB) * 1024 * 1024 * 1024
	cfg.SuccessFile = filepath.Join(cfg.InputDir, time.Now().Format("2006-01-02_15-04-05")+".txt")

	// 创建下载管理器
	dm := &DownloadManager{
		config:        &cfg,
		existingFiles: make(map[string]bool),
		downloaded:    make(map[string]bool),
	}
	dm.ctx, dm.cancel = context.WithCancel(context.Background())

	// 确保目录存在
	if err := os.MkdirAll(cfg.InputDir, 0o755); err != nil {
		fmt.Printf("创建输入目录失败: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(cfg.DownloadDir, 0o755); err != nil {
		fmt.Printf("创建下载目录失败: %v\n", err)
		os.Exit(1)
	}

	// 设置信号处理
	setupSignalHandler(dm)

	// 初始化
	if err := dm.initialize(); err != nil {
		fmt.Printf("初始化失败: %v\n", err)
		os.Exit(1)
	}

	// 执行主逻辑
	if err := dm.run(); err != nil {
		fmt.Printf("执行失败: %v\n", err)
	}

	// 清理资源
	dm.cleanup()
}

// 设置信号处理
func setupSignalHandler(dm *DownloadManager) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\n收到信号: %v，正在保存进度...\n", sig)
		dm.saveProgress()
		dm.cancel()
	}()
}

// 初始化
func (dm *DownloadManager) initialize() error {
	fmt.Println("正在初始化...")

	// 1. 扫描已有文件
	if err := dm.scanExistingFiles(); err != nil {
		return fmt.Errorf("扫描已有文件失败: %w", err)
	}
	fmt.Printf("已加载 %d 个已有文件\n", len(dm.existingFiles))

	// 2. 加载之前的下载进度
	if err := dm.loadProgress(); err != nil {
		fmt.Printf("加载进度失败: %v\n", err)
	}

	// 3. 打开成功和失败记录文件
	if err := dm.openLogFiles(); err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	// 4. 连接MongoDB
	if err := dm.connectMongo(); err != nil {
		return fmt.Errorf("连接MongoDB失败: %w", err)
	}

	// 5. 如果需要，创建索引
	if dm.config.CreateIndex {
		if err := dm.createIndexes(); err != nil {
			fmt.Printf("创建索引失败: %v (但程序将继续运行)\n", err)
		}
	}

	// 6. 计算已下载文件总大小
	if err := dm.calculateTotalSize(); err != nil {
		return fmt.Errorf("计算已下载大小失败: %w", err)
	}

	fmt.Printf("初始化完成，当前已下载大小: %.2f GB\n",
		float64(atomic.LoadInt64(&dm.totalSize))/1024/1024/1024)
	fmt.Printf("最大限制: %d GB\n", dm.config.MaxSizeGB)
	fmt.Printf("从 PID=%d 开始处理\n", dm.latestPID)

	return nil
}

// 连接MongoDB
func (dm *DownloadManager) connectMongo() error {
	clientOptions := options.Client().ApplyURI(dm.config.MongoURI)
	client, err := mongo.Connect(dm.ctx, clientOptions)
	if err != nil {
		return err
	}

	// 测试连接
	if err := client.Ping(dm.ctx, nil); err != nil {
		return err
	}

	dm.client = client
	return nil
}

// 创建索引
func (dm *DownloadManager) createIndexes() error {
	collection := dm.client.Database(dm.config.Database).Collection(dm.config.Collection)

	// 为pid字段创建升序索引
	indexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "pid", Value: 1}},
	}

	_, err := collection.Indexes().CreateOne(dm.ctx, indexModel)
	if err != nil {
		return fmt.Errorf("创建pid索引失败: %w", err)
	}

	fmt.Println("已为pid字段创建索引")
	return nil
}

// 扫描已有文件
func (dm *DownloadManager) scanExistingFiles() error {
	// 扫描input目录下的txt文件
	files, err := filepath.Glob(filepath.Join(dm.config.InputDir, "*.txt"))
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := dm.readFileLines(file); err != nil {
			fmt.Printf("读取文件 %s 失败: %v\n", file, err)
		}
	}
	return nil
}

// 读取文件行
func (dm *DownloadManager) readFileLines(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			dm.mu.Lock()
			dm.existingFiles[line] = true
			dm.mu.Unlock()
		}
	}
	return scanner.Err()
}

// 加载进度
func (dm *DownloadManager) loadProgress() error {
	// 读取最新的PID
	if data, err := os.ReadFile(dm.config.LatestPIDFile); err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			dm.latestPID = pid
		}
	}

	// 读取失败记录，避免重复下载失败的文件
	if file, err := os.Open(dm.config.FailFile); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			parts := strings.Fields(scanner.Text())
			if len(parts) >= 2 {
				dm.mu.Lock()
				filename := path.Base(parts[1])
				dm.existingFiles[filename] = true
				dm.mu.Unlock()
			}
		}
	}

	return nil
}

// 打开日志文件
func (dm *DownloadManager) openLogFiles() error {
	// 打开成功记录文件
	successFile, err := os.OpenFile(dm.config.SuccessFile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	dm.successFile = successFile

	// 打开失败记录文件
	failFile, err := os.OpenFile(dm.config.FailFile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		dm.successFile.Close()
		return err
	}
	dm.failFile = failFile

	return nil
}

// 计算已下载文件总大小
func (dm *DownloadManager) calculateTotalSize() error {
	var total int64
	err := filepath.Walk(dm.config.DownloadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return err
	}

	atomic.StoreInt64(&dm.totalSize, total)
	return nil
}

// 主运行逻辑
func (dm *DownloadManager) run() error {
	defer dm.client.Disconnect(dm.ctx)

	collection := dm.client.Database(dm.config.Database).Collection(dm.config.Collection)

	// 使用游标分页，避免skip操作
	for {
		select {
		case <-dm.ctx.Done():
			return nil
		default:
			if err := dm.processPage(collection); err != nil {
				return err
			}
		}
	}
}

// 处理分页 - 使用游标分页替代skip
func (dm *DownloadManager) processPage(collection *mongo.Collection) error {
	// 使用游标分页：查询pid > lastPID的文档，按pid升序排列
	filter := bson.D{}
	if dm.latestPID > 0 {
		filter = bson.D{{Key: "pid", Value: bson.D{{Key: "$gt", Value: dm.latestPID}}}}
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "pid", Value: 1}}).
		SetLimit(int64(dm.config.PageSize))

	cursor, err := collection.Find(dm.ctx, filter, findOptions)
	if err != nil {
		return fmt.Errorf("查询失败: %w", err)
	}
	defer cursor.Close(dm.ctx)

	var processedDocs int
	for cursor.Next(dm.ctx) {
		var data DataInfo
		if err := cursor.Decode(&data); err != nil {
			fmt.Printf("解码文档失败: %v\n", err)
			continue
		}

		if err := dm.processDocument(data); err != nil {
			if errors.Is(err, ErrMaxDownload) {
				return ErrMaxDownload
			}
			fmt.Printf("处理文档失败 (PID: %d): %v\n", data.PID, err)
		}

		// 更新最新PID
		dm.latestPID = data.PID
		processedDocs++
	}

	// 保存进度
	if processedDocs > 0 {
		if err := dm.saveProgress(); err != nil {
			fmt.Printf("保存进度失败: %v\n", err)
		}
		fmt.Printf("已处理 %d 个文档，当前PID: %d\n", processedDocs, dm.latestPID)
	}

	// 如果没有数据，等待一会儿再检查
	if processedDocs == 0 {
		fmt.Println("没有更多数据，等待10秒后重新检查...")
		select {
		case <-dm.ctx.Done():
			return nil
		case <-time.After(10 * time.Second):
		}
	}

	return nil
}

// 处理单个文档
func (dm *DownloadManager) processDocument(data DataInfo) error {
	for _, picInfo := range data.PicInfos {
		select {
		case <-dm.ctx.Done():
			return nil
		default:
			if err := dm.downloadFile(data.PID, picInfo.Href); err != nil {
				if errors.Is(err, ErrMaxDownload) {
					return ErrMaxDownload
				}
				dm.recordFailure(data.PID, picInfo.Href, err)
			}
		}
	}
	return nil
}

var ErrMaxDownload = errors.New("已达到最大下载限制")

// 下载文件
func (dm *DownloadManager) downloadFile(pid int, url string) error {
	// 检查是否已达大小限制
	if atomic.LoadInt64(&dm.totalSize) >= dm.config.MaxTotalSize {
		fmt.Printf("已达到最大下载限制 %d GB\n", dm.config.MaxSizeGB)
		dm.saveProgress()
		dm.cancel()
		return ErrMaxDownload
	}

	// 提取文件名
	filename := path.Base(url)
	if filename == "" || filename == "." || filename == "/" {
		return fmt.Errorf("无法从URL提取文件名: %s", url)
	}

	// 检查是否已存在
	dm.mu.RLock()
	if dm.existingFiles[filename] || dm.downloaded[filename] {
		dm.mu.RUnlock()
		fmt.Printf("文件已存在，跳过: %s\n", filename)
		return nil
	}
	dm.mu.RUnlock()

	// 构建目标路径
	filepath := filepath.Join(dm.config.DownloadDir, filename)

	// 检查wget是否存在
	if _, err := exec.LookPath("wget"); err != nil {
		// 如果wget不存在，使用Go的HTTP客户端下载
		return dm.downloadWithHTTP(pid, url, filepath, filename)
	}

	// 使用wget下载文件（支持断点续传）
	cmd := exec.Command("wget", "-c", "-O", filepath, url, "-q", "--show-progress")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("正在下载(PID:%d): %s -> %s\n", pid, url, filename)

	if err := cmd.Run(); err != nil {
		// 如果下载失败，删除可能的部分文件
		os.Remove(filepath)
		return fmt.Errorf("wget下载失败: %w", err)
	}

	// 更新下载状态
	return dm.updateDownloadStatus(filepath, filename)
}

// 使用HTTP客户端下载
func (dm *DownloadManager) downloadWithHTTP(pid int, url, filepath, filename string) error {
	fmt.Printf("wget不可用，使用HTTP下载(PID:%d): %s\n", pid, filename)

	// 发送HTTP请求
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}

	// 创建文件
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 写入文件
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(filepath)
		return fmt.Errorf("写入文件失败: %w", err)
	}

	// 更新下载状态
	return dm.updateDownloadStatus(filepath, filename)
}

// 更新下载状态
func (dm *DownloadManager) updateDownloadStatus(filepath, filename string) error {
	// 检查文件大小
	info, err := os.Stat(filepath)
	if err != nil {
		os.Remove(filepath)
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 更新总大小
	newTotal := atomic.AddInt64(&dm.totalSize, info.Size())

	// 检查是否超过限制
	if newTotal > dm.config.MaxTotalSize {
		fmt.Printf("警告: 下载后总大小 %.2f GB 超过限制 %d GB\n",
			float64(newTotal)/1024/1024/1024, dm.config.MaxSizeGB)
		os.Remove(filepath)
		atomic.AddInt64(&dm.totalSize, -info.Size())
		return fmt.Errorf("超过大小限制")
	}

	// 记录成功
	dm.mu.Lock()
	dm.existingFiles[filename] = true
	dm.downloaded[filename] = true
	dm.mu.Unlock()

	// 写入成功记录
	dm.recordSuccess(filename)

	fmt.Printf("下载成功: %s (大小: %.2f MB, 总计: %.2f GB)\n",
		filename, float64(info.Size())/1024/1024,
		float64(newTotal)/1024/1024/1024)

	return nil
}

// 记录成功
func (dm *DownloadManager) recordSuccess(filename string) {
	if dm.successFile != nil {
		dm.successFile.WriteString(filename + "\n")
		dm.successFile.Sync()
	}
}

// 记录失败
func (dm *DownloadManager) recordFailure(pid int, url string, err error) {
	if dm.failFile != nil {
		record := fmt.Sprintf("%d %s %v\n", pid, url, err)
		dm.failFile.WriteString(record)
		dm.failFile.Sync()
		fmt.Printf("下载失败: PID=%d, URL=%s, Error=%v\n", pid, url, err)
	}
}

// 保存进度
func (dm *DownloadManager) saveProgress() error {
	// 保存最新PID
	pidData := fmt.Appendf(nil, "%d", dm.latestPID)
	if err := os.WriteFile(dm.config.LatestPIDFile, pidData, 0o644); err != nil {
		return fmt.Errorf("保存PID失败: %w", err)
	}

	// 同步记录文件
	if dm.successFile != nil {
		dm.successFile.Sync()
	}
	if dm.failFile != nil {
		dm.failFile.Sync()
	}

	fmt.Printf("进度已保存: PID=%d, 已下载: %d 文件, 总大小: %.2f GB\n",
		dm.latestPID, len(dm.downloaded),
		float64(atomic.LoadInt64(&dm.totalSize))/1024/1024/1024)

	return nil
}

// 清理资源
func (dm *DownloadManager) cleanup() {
	dm.cancel()
	dm.wg.Wait()

	if dm.successFile != nil {
		dm.successFile.Close()
	}
	if dm.failFile != nil {
		dm.failFile.Close()
	}
	if dm.client != nil {
		dm.client.Disconnect(context.Background())
	}

	dm.saveProgress()
	fmt.Println("程序已退出，进度已保存")
}

// 数据结构
type DataInfo struct {
	PID      int       `bson:"pid"`
	PicInfos []PicInfo `bson:"pic_infos"`
}

type PicInfo struct {
	Href string `bson:"href"`
}
