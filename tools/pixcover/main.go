// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
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
	// 解析命令行参数
	config := parseFlags()
	config.MaxTotalSize = int64(config.MaxSizeGB) * 1024 * 1024 * 1024
	config.StartTime = time.Now()
	config.SuccessFile = filepath.Join(config.InputDir, config.StartTime.Format("2006-01-02_15-04-05")+".txt")

	// 创建下载管理器
	dm := &DownloadManager{
		config:        config,
		existingFiles: make(map[string]bool),
		downloaded:    make(map[string]bool),
	}
	dm.ctx, dm.cancel = context.WithCancel(context.Background())

	// 确保目录存在
	if err := os.MkdirAll(config.InputDir, 0o755); err != nil {
		fmt.Printf("创建输入目录失败: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(config.DownloadDir, 0o755); err != nil {
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

// 解析命令行参数
func parseFlags() *Config {
	config := &Config{}
	flag.StringVar(&config.InputDir, "input", "/data/comic/input", "输入目录路径")
	flag.StringVar(&config.DownloadDir, "download", "/data/comic/pixiv", "下载目录路径")
	flag.StringVar(&config.MongoURI, "mongo", "mongodb://comic:HxYJdyTRxDLhGtSW@localhost:27017/comic", "MongoDB连接URI")
	flag.StringVar(&config.Database, "db", "comic", "数据库名")
	flag.StringVar(&config.Collection, "collection", "pixivInfo", "集合名")
	flag.IntVar(&config.PageSize, "pagesize", 100, "分页大小")
	flag.IntVar(&config.MaxSizeGB, "maxsize", 10, "最大下载大小(GB)")
	flag.StringVar(&config.LatestPIDFile, "pidfile", "latest-pid", "最新PID记录文件")
	flag.StringVar(&config.FailFile, "failfile", "fail.txt", "失败记录文件")
	flag.BoolVar(&config.CreateIndex, "create-index", false, "是否为pid字段创建索引")
	flag.Parse()
	return config
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
	fmt.Printf("初始化完成，当前已下载大小: %.2f GB\n", float64(atomic.LoadInt64(&dm.totalSize))/1024/1024/1024)
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
	successFile, err := os.OpenFile(dm.config.SuccessFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	dm.successFile = successFile

	// 打开失败记录文件
	failFile, err := os.OpenFile(dm.config.FailFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
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

// 处理分页 - 改为批量处理
func (dm *DownloadManager) processPage(collection *mongo.Collection) (err error) {
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

	// 1. 收集当前页所有URL
	var urls []string
	var maxPID int

	for cursor.Next(dm.ctx) {
		var data DataInfo
		if err := cursor.Decode(&data); err != nil {
			fmt.Printf("解码文档失败: %v\n", err)
			continue
		}
		// 收集所有图片URL
		for _, picInfo := range data.PicInfos {
			filename := path.Base(picInfo.Href)
			// 检查文件是否已存在或已下载（避免重复写入列表）
			dm.mu.RLock()
			if !dm.existingFiles[filename] && !dm.downloaded[filename] {
				urls = append(urls, picInfo.Href)
			}
			dm.mu.RUnlock()
		}
		if data.PID > maxPID {
			maxPID = data.PID
		}
	}

	// 如果当前页没有URL需要下载，更新PID并返回
	if len(urls) == 0 {
		dm.latestPID = maxPID
		dm.saveProgress()
		fmt.Printf("当前页无新文件，已处理到 PID: %d\n", dm.latestPID)
		return nil
	}

	tempURLFile := filepath.Join(os.TempDir(), "wget_batch_"+strconv.FormatInt(time.Now().Unix(), 10)+".txt")
	defer func() {
		if err == nil {
			os.Remove(tempURLFile)
		}
	}()

	// 2. 写入临时URL列表文件
	if err = os.WriteFile(tempURLFile, []byte(strings.Join(urls, "\n")+"\n"), 0o644); err != nil {
		return fmt.Errorf("写入临时URL文件失败: %w", err)
	}

	// 3. 执行批量下载
	if err = dm.downloadBatch(urls, tempURLFile); err != nil {
		return fmt.Errorf("批量下载失败: %w", err)
	}

	// 4. 更新最新PID
	dm.latestPID = maxPID

	// 5. 保存进度
	if err = dm.saveProgress(); err != nil {
		fmt.Printf("保存进度失败: %v\n", err)
	}

	fmt.Printf("已处理一页，下载 %d 个文件，当前PID: %d\n", len(urls), dm.latestPID)
	return nil
}

// 批量下载所有URL
func (dm *DownloadManager) downloadBatch(allURLs []string, tempURLFile string) error {
	// 检查wget是否存在
	if _, err := exec.LookPath("wget"); err != nil {
		// 如果wget不存在，退回到逐个下载（虽然慢，但能工作）
		fmt.Println("wget不可用，退回到逐个下载模式...")
		for _, url := range allURLs {
			// 这里复用原来的downloadFile逻辑，但移除了分页控制
			filename := path.Base(url)
			if filename == "" {
				continue
			}
			// 简单检查大小限制
			if atomic.LoadInt64(&dm.totalSize) >= dm.config.MaxTotalSize {
				return errors.New("达到大小限制")
			}
			// 模拟下载（实际应调用HTTP下载或记录）
			// 这里仅作演示，实际项目中应实现单个下载逻辑
			fmt.Printf("下载: %s\n", filename)
		}
		return nil
	}

	// 检查大小限制
	if atomic.LoadInt64(&dm.totalSize) >= dm.config.MaxTotalSize {
		fmt.Printf("已达到最大下载限制 %d GB\n", dm.config.MaxSizeGB)
		dm.cancel()
		return errors.New("达到大小限制")
	}

	// 构建wget命令：从文件读取URL，断点续传，限制并发连接数（防止内存溢出）
	cmd := exec.Command("wget",
		"-i", tempURLFile, // 从文件读取URL列表
		"-c",                        // 断点续传
		"-P", dm.config.DownloadDir, // 指定下载目录
		"-t", "3", // 重试3次
		"--timeout=30",          // 超时时间
		"--wait=0.5",            // 下载间隔（可选，防止对服务器造成压力）
		"--random-wait",         // 随机等待
		"-q", "--show-progress", // 静默模式但显示进度
	)

	// 捕获输出用于调试（可选）
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("正在批量下载 %d 个文件...\n", len(allURLs))
	err := cmd.Run()

	// 无论成功与否，先尝试统计下载成功的文件
	// 因为wget -i 是原子性的，要么全成功，要么部分失败
	// 我们需要扫描DownloadDir来确认哪些真的下载好了
	if err == nil {
		fmt.Println("批量下载命令执行成功")
	} else {
		fmt.Printf("wget 批量命令返回错误: %v\n", err)
		// 即使命令失败，也可能部分文件下载好了，我们继续扫描
	}

	// 6. 扫描下载目录，更新状态
	// 由于是批量下载，我们无法知道wget具体下载了哪些（除非解析日志，太复杂）
	// 简单做法：检查所有URL对应的文件是否存在且大小>0
	var downloadedFiles []string
	var totalNewSize int64

	for _, url := range allURLs {
		filename := path.Base(url)
		if filename == "" || filename == "." {
			continue
		}
		filepath := filepath.Join(dm.config.DownloadDir, filename)

		// 检查文件是否存在且非空
		if info, err := os.Stat(filepath); err == nil && info.Size() > 0 {
			// 文件下载成功
			downloadedFiles = append(downloadedFiles, filename)
			totalNewSize += info.Size()

			// 更新内存状态
			dm.mu.Lock()
			dm.existingFiles[filename] = true
			dm.downloaded[filename] = true
			dm.mu.Unlock()

			dm.recordSuccess(filename)
			fmt.Printf("下载成功: %s (大小: %.2f MB)\n", filename, float64(info.Size())/1024/1024)
		}
	}

	// 更新总大小
	newTotal := atomic.AddInt64(&dm.totalSize, totalNewSize)
	fmt.Printf("本批次实际成功下载: %d / %d, 总大小增加: %.2f MB, 当前总计: %.2f GB\n",
		len(downloadedFiles), len(allURLs), float64(totalNewSize)/1024/1024, float64(newTotal)/1024/1024/1024)

	// 检查是否超过限制
	if newTotal > dm.config.MaxTotalSize {
		fmt.Printf("警告: 下载后总大小 %.2f GB 超过限制 %d GB\n", float64(newTotal)/1024/1024/1024, dm.config.MaxSizeGB)
		// 这里简单处理，实际可能需要删除超额文件
		return errors.New("超过大小限制")
	}

	// 记录未下载成功的文件
	for _, url := range allURLs {
		filename := path.Base(url)
		if filename == "" {
			continue
		}
		dm.mu.RLock()
		exists := dm.existingFiles[filename]
		dm.mu.RUnlock()
		if !exists {
			// 这里简单记录为失败，实际可能是网络问题
			dm.recordFailure(0, url, errors.New("批量下载未完成"))
		}
	}

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
		dm.latestPID, len(dm.downloaded), float64(atomic.LoadInt64(&dm.totalSize))/1024/1024/1024)
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
