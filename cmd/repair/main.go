// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// 配置常量
const (
	MongoDBURI     = "mongodb://root:root123@suixi-mini.local:27017/?authSource=admin"
	DatabaseName   = "cocom"
	CollectionName = "comicInfo"
	SuccDir        = "./succ"
	FailDir        = "./fail"
	AuditDir       = "./audit_passed"
	AuditFailDir   = "./audit_failed"
	MaxConcurrency = 5
	DefaultPort    = 8088
)

// 全局变量
var (
	mode       string
	port       int
	mongoURI   string
	serverAddr string
)

// APIResponse 定义API返回的结构
type APIResponse struct {
	Head struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	} `json:"head"`
	Body struct {
		InvalidImages []InvalidImage `json:"invalid_images"`
	} `json:"body"`
}

// InvalidImage 定义无效图片结构
type InvalidImage struct {
	Index int `json:"index"`
}

// Task 定义要处理的任务
type Task struct {
	CID     int
	ResBody APIResponse
}

// ComicInfo 定义漫画信息结构
type ComicInfo struct {
	CID      int       `bson:"cid" json:"cid"`
	Title    string    `bson:"title,omitempty" json:"title,omitempty"`
	Author   string    `bson:"author,omitempty" json:"author,omitempty"`
	Images   []string  `json:"images,omitempty"`
	Total    int       `json:"total,omitempty"`
	Repaired int       `json:"repaired,omitempty"`
	Modified time.Time `json:"modified"`
	Status   string    `json:"status"`
}

func main() {
	// 解析命令行参数
	flag.StringVar(&mode, "mode", "worker", "运行模式: worker(执行修复) 或 server(启动HTTP服务器) 或 commit(提交修复结果)")
	flag.IntVar(&port, "port", DefaultPort, "HTTP服务器端口")
	flag.StringVar(&mongoURI, "mongo", MongoDBURI, "MongoDB连接URI")
	flag.StringVar(&serverAddr, "server", "http://suixi-mini.local:15456", "API服务器地址")
	flag.Parse()

	// 创建必要目录
	os.MkdirAll(SuccDir, 0o755)
	os.MkdirAll(FailDir, 0o755)
	os.MkdirAll(AuditDir, 0o755)
	os.MkdirAll(AuditFailDir, 0o755)

	switch mode {
	case "worker":
		runWorkerMode()
	case "server":
		runServerMode()
	case "commit":
		runCommitMode()
	default:
		fmt.Printf("未知模式: %s，支持的模式: worker, server, commit\n", mode)
		os.Exit(1)
	}
}

// ==================== Worker模式 ====================

func runWorkerMode() {
	fmt.Println("========== 启动Worker模式 ==========")
	fmt.Printf("API服务器: %s\n", serverAddr)
	fmt.Printf("MongoDB URI: %s\n", mongoURI)
	fmt.Println("正在执行漫画图片修复任务...")

	// 1. 连接MongoDB并查询符合条件的cid列表
	cidList, err := getCIDsFromMongo()
	if err != nil {
		fmt.Printf("从MongoDB获取CID列表失败: %v\n", err)
		return
	}
	if len(cidList) == 0 {
		fmt.Println("没有找到需要处理的CID")
		return
	}
	fmt.Printf("共获取到 %d 个需要处理的CID\n", len(cidList))

	// 2. 设置并发限制
	taskChan := make(chan Task, len(cidList))
	var wg sync.WaitGroup

	// 3. 启动工作goroutine
	for i := range MaxConcurrency {
		wg.Add(1)
		go worker(i, taskChan, &wg)
	}

	// 4. 为每个cid调用API，并将需要处理的任务放入通道
	client := &http.Client{Timeout: 30 * time.Second}
	for _, cid := range cidList {
		resp, err := callArchiveAPI(client, cid)
		if err != nil {
			fmt.Printf("[CID:%d] 调用API失败: %v\n", cid, err)
			moveToFail(cid)
			continue
		}

		if resp.Head.Code == -1001 {
			fmt.Printf("[CID:%d] 发现 %d 个异常图片需要修复\n", cid, len(resp.Body.InvalidImages))
			taskChan <- Task{CID: cid, ResBody: *resp}
		} else {
			fmt.Printf("[CID:%d] API返回code: %d, 无需处理\n", cid, resp.Head.Code)
			moveToSucc(cid)
		}
	}

	// 5. 关闭通道并等待所有worker完成
	close(taskChan)
	wg.Wait()
	fmt.Println("========== 所有任务处理完成 ==========")
}

// ==================== Server模式 ====================

func runServerMode() {
	fmt.Println("========== 启动Server模式 ==========")
	fmt.Printf("HTTP服务器监听端口: %d\n", port)
	fmt.Printf("succ目录: %s\n", SuccDir)
	fmt.Printf("审计通过目录: %s\n", AuditDir)

	// 创建路由器
	r := mux.NewRouter()

	protection := http.NewCrossOriginProtection()
	protection.AddTrustedOrigin("http://suixi-mini.local:8088")

	// 设置路由
	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/api/comics", getComicsHandler).Methods("GET")
	r.HandleFunc("/api/comic/{cid}", getComicImagesHandler).Methods("GET")
	r.HandleFunc("/api/comic/{cid}/images/{filename}", serveImageHandler).Methods("GET")
	r.HandleFunc("/api/comic/{cid}/audit", auditComicHandler).Methods("POST")
	r.HandleFunc("/api/comic/{cid}/reject", rejectComicHandler).Methods("POST")
	r.HandleFunc("/api/rejected/comic/{cid}", getRejectedComicImagesHandler).Methods("GET")
	r.HandleFunc("/api/rejected/comic/{cid}/images/{filename}", serveRejectedImageHandler).Methods("GET")
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// 如果没有static目录，创建并生成默认页面
	ensureStaticDir()

	// 启动HTTP服务器
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("请访问 http://localhost:%d 查看UI界面\n", port)
	fmt.Printf("API接口:\n")
	fmt.Printf("  GET  /api/comics             - 获取所有漫画列表\n")
	fmt.Printf("  GET  /api/comic/{cid}         - 获取指定漫画的图片列表\n")
	fmt.Printf("  POST /api/comic/{cid}/audit   - 审计通过，移动到审计目录\n")

	h := protection.Handler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		slog.Info("Received request", "method", req.Method, "path", req.URL.Path)
		r.ServeHTTP(w, req)
	}))

	if err := http.ListenAndServe(addr, h); err != nil {
		fmt.Printf("HTTP服务器启动失败: %v\n", err)
		os.Exit(1)
	}
}

// ==================== Commit模式 ====================

func runCommitMode() {
	fmt.Println("========== 启动Commit模式 ==========")
	fmt.Printf("API服务器: %s\n", serverAddr)
	fmt.Printf("审计通过目录: %s\n", AuditDir)
	fmt.Println("正在提交修复结果...")

	// 1、从AuditDir目录读取所有文件
	files, err := os.ReadDir(AuditDir)
	if err != nil {
		fmt.Printf("读取审计目录失败: %v\n", err)
		return
	}
	if len(files) == 0 {
		fmt.Println("没有找到需要提交的修复结果")
		return
	}
	fmt.Printf("共获取到 %d 个需要提交的修复结果\n", len(files))

	client := &http.Client{Timeout: 30 * time.Second}
	// 2、遍历每个文件，提交修复结果
	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		fmt.Printf("正在提交修复结果: %s\n", file.Name())

		cid, err := strconv.Atoi(file.Name())
		if err != nil {
			fmt.Printf("转换CID失败: %v\n", err)
			return
		}

		// 检查是否已经归档
		resp, err := callArchiveAPI(client, cid)
		if err != nil {
			fmt.Printf("[CID:%d] 调用API失败: %v\n", cid, err)
			moveToFail(cid)
			continue
		}
		if resp.Head.Code == -1001 {
			fmt.Printf("[CID:%d] 发现 %d 个异常图片需要修复\n", cid, len(resp.Body.InvalidImages))
		} else {
			fmt.Printf("[CID:%d] API返回code: %d, 无需处理\n", cid, resp.Head.Code)
			os.MkdirAll("commit", 0o755)
			os.Rename(filepath.Join(AuditDir, file.Name()), filepath.Join("commit", file.Name()))
			continue
		}

		// 3、查询图片保存目录
		saveDir, err := queryImageSaveDir(client, cid)
		if err != nil {
			fmt.Printf("查询图片保存目录失败: %v\n", err)
			return
		}
		fmt.Printf("图片保存目录: %s\n", saveDir)

		entries, err := os.ReadDir(filepath.Join(AuditDir, file.Name()))
		if err != nil {
			fmt.Printf("读取图片目录失败: %v\n", err)
			return
		}

		for _, entry := range entries {
			if entry.IsDir() {
				fmt.Printf("跳过目录: %s\n", entry.Name())
				continue
			}

			// mv xx-repaired.jpg saveDir && touch -r xx.jpg xx-repaired.jpg
			oldPath := filepath.Join(AuditDir, file.Name(), entry.Name())
			newPath := filepath.Join(saveDir, entry.Name())
			err = exec.Command("cp", oldPath, newPath).Run()
			if err != nil {
				fmt.Printf("复制图片到保存目录失败: %v\n", err)
				continue
			}
			fmt.Printf("图片移动到: %s\n", saveDir)

			errPicPath := filepath.Join(saveDir, strings.Replace(entry.Name(), "-repaired", "", 1))
			oldStat, err := os.Stat(errPicPath)
			if err != nil {
				fmt.Printf("获取原始图片时间戳失败: %v\n", err)
				continue
			}

			// 6、更新修复时间戳
			err = os.Chtimes(newPath, oldStat.ModTime(), oldStat.ModTime())
			if err != nil {
				fmt.Printf("更新修复时间戳失败: %v\n", err)
				continue
			}
			fmt.Printf("修复时间戳更新成功: %s\n", newPath)

			err = os.Rename(newPath, errPicPath)
			if err != nil {
				fmt.Printf("失败: %v\n", err)
				continue
			}
			fmt.Printf("成功: %s\n", errPicPath)
		}

		// 5、调用API提交修复结果
		resp, err = callCommitAPI(client, cid)
		if err != nil {
			fmt.Printf("提交修复结果失败: %v\n", err)
			continue
		}
		fmt.Printf("提交修复结果成功: %s\n", resp.Head.Msg)
	}
}

// ==================== UI界面处理函数 ====================

func homeHandler(w http.ResponseWriter, r *http.Request) {
	// 返回HTML页面
	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>漫画图片审计系统</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: Arial, sans-serif; background: #f5f5f5; padding: 20px; }
        .header { background: #1890ff; color: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
        .header h1 { margin-bottom: 10px; }
        .stats { display: flex; gap: 20px; margin-bottom: 20px; }
        .stat-card { background: white; padding: 15px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); flex: 1; }
        .stat-card h3 { color: #666; margin-bottom: 10px; }
        .stat-card .value { font-size: 24px; font-weight: bold; }
        .comic-list { background: white; border-radius: 8px; padding: 20px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
        .comic-item { display: flex; justify-content: space-between; align-items: center; padding: 15px; border-bottom: 1px solid #eee; }
        .comic-item:last-child { border-bottom: none; }
        .comic-info { flex: 1; }
        .comic-info h3 { color: #1890ff; margin-bottom: 5px; }
        .comic-meta { color: #666; font-size: 14px; }
        .comic-actions { display: flex; gap: 10px; }
        .btn { padding: 8px 16px; border: none; border-radius: 4px; cursor: pointer; font-weight: bold; }
        .btn-view { background: #52c41a; color: white; }
        .btn-audit { background: #1890ff; color: white; }
        .btn-audit:hover { background: #40a9ff; }
        .btn-view:hover { background: #73d13d; }
        .modal { display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.5); z-index: 1000; }
        .modal-content { position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); background: white; padding: 20px; border-radius: 8px; max-width: 90%; max-height: 90%; overflow: auto; }
		/* 新增：图片预览模态框全屏样式 */
        .modal-content.fullscreen {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            transform: none;
            width: 100%;
            height: 100%;
            max-width: 100%;
            max-height: 100%;
            border-radius: 0;
            padding: 0;
        }
        
        /* 新增：全屏预览容器 */
        .image-preview-container {
            display: flex;
            flex-direction: column;
            height: 100%;
            background-color: #000;
        }
        
        /* 修改：工具栏 */
        .preview-toolbar {
            padding: 12px 20px;
            background: rgba(0, 0, 0, 0.9);
        }
        
        /* 新增：图片导航 */
        .image-navigation {
            display: flex;
            align-items: center;
            gap: 20px;
        }

		/* 新增：鼠标拖动状态 */
        .dragging {
            cursor: grabbing !important;
        }
        
		/* 修改：图片导航按钮 */
        .image-navigation button {
            padding: 8px 16px;
            font-size: 16px;
        }
        
        /* 修改：当前图片信息 */
        #currentImageInfo {
            color: white;
            font-size: 16px;
            font-weight: bold;
        }

        /* 修改：图片显示区域 */
        .image-display {
            flex: 1;
            display: flex;
            justify-content: center;
            align-items: center;
            overflow: auto;
            padding: 10px;
        }
        
        /* 修改：大图样式 - 自动适应高度 */
        .preview-image {
            max-height: calc(100vh - 180px); /* 减去工具栏和缩略图高度 */
            max-width: 100%;
            object-fit: contain;
            cursor: pointer;
            transition: transform 0.3s ease;
        }

		/* 新增：图片缩放状态 */
        .preview-image.zoomed {
            max-height: none;
            max-width: none;
            width: auto;
            height: auto;
            cursor: move;
        }
        
        /* 修改：缩略图网格 */
        .thumbnail-grid {
            height: 100px;
            padding: 10px;
            background: rgba(0, 0, 0, 0.9);
        }
        .thumbnail {
            width: 60px;
            height: 60px;
            object-fit: cover;
            cursor: pointer;
            border: 2px solid transparent;
            border-radius: 4px;
        }
        
        .thumbnail.active {
            border-color: #1890ff;
        }
        
        /* 修改：审核按钮样式 */
        .audit-buttons {
            display: flex;
            gap: 15px;
        }
        
        /* 新增：图片缩放控件 */
        .zoom-controls {
            display: flex;
            align-items: center;
            gap: 10px;
            margin-left: 20px;
        }

		.zoom-controls button {
            background: #666;
            color: white;
            border: none;
            border-radius: 4px;
            width: 30px;
            height: 30px;
            font-size: 18px;
            cursor: pointer;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        
        .zoom-controls button:hover {
            background: #888;
        }

		/* 新增：图片操作提示 */
        .image-hints {
            color: white;
            font-size: 12px;
            margin-top: 5px;
            opacity: 0.8;
        }
        
        /* 新增：加载指示器 */
        .image-loading {
            color: white;
            font-size: 18px;
        }
        
        .btn-reject {
            background: #f5222d;
            color: white;
        }
        
        .btn-reject:hover {
            background: #ff4d4f;
        }
        .modal-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
        .modal-title { font-size: 20px; font-weight: bold; }
        .close { font-size: 24px; cursor: pointer; }
        .image-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 10px; margin-top: 20px; }
        .image-item { position: relative; }
        .image-item img { width: 100%; height: 150px; object-fit: cover; border-radius: 4px; }
        .image-index { position: absolute; top: 5px; left: 5px; background: rgba(0,0,0,0.7); color: white; padding: 2px 5px; border-radius: 3px; font-size: 12px; }
        .loading { text-align: center; padding: 20px; color: #666; }
        .audit-btn { margin-top: 20px; text-align: center; }
    </style>
</head>
<body>
    <div class="header">
        <h1>漫画图片审计系统</h1>
        <p>已修复的漫画图片审计界面 - 共 <span id="totalCount">0</span> 个项目</p>
    </div>
    
    <div class="stats">
		<div class="stat-card">
			<h3>等待审计</h3>
			<div class="value" id="pendingCount">0</div>
		</div>
		<div class="stat-card">
			<h3>已通过审计</h3>
			<div class="value" id="passedCount">0</div>
		</div>
		<div class="stat-card">
			<h3>审核不通过</h3>
			<div class="value" id="rejectedCount">0</div>
		</div>
		<div class="stat-card">
			<h3>修复失败</h3>
			<div class="value" id="failedCount">0</div>
		</div>
	</div>
    
    <div class="comic-list">
        <h2>等待审计的漫画</h2>
        <div id="comicList" class="loading">正在加载...</div>
    </div>

    <div id="imageModal" class="modal">
        <div class="modal-content" id="modalContent">
            <div class="preview-toolbar">
                <div>
                    <button class="btn btn-view" onclick="toggleFullscreen()" id="fullscreenBtn">全屏</button>
                    <span id="currentImageInfo" style="margin-left: 20px;"></span>
                    <div class="image-hints" id="imageHints">点击图片缩放，按住拖动查看细节</div>
                </div>
                <div class="image-navigation">
                    <button class="btn btn-view" onclick="prevImage()" title="上一张 (←)">← 上一张</button>
                    <span id="imageCounter" style="color: white; margin: 0 10px; min-width: 80px; text-align: center;"></span>
                    <button class="btn btn-view" onclick="nextImage()" title="下一张 (→)">下一张 →</button>
                </div>
                <div class="zoom-controls">
                    <button onclick="zoomOut()" title="缩小">-</button>
                    <button onclick="resetZoom()" title="重置缩放">↺</button>
                    <button onclick="zoomIn()" title="放大">+</button>
                </div>
                <div class="audit-buttons">
                    <button class="btn btn-reject" onclick="rejectCurrentComic()" title="审核不通过 (R)">审核不通过</button>
                    <button class="btn btn-audit" onclick="auditCurrentComic()" title="审计通过 (A)">审计通过</button>
                    <button class="btn" onclick="closeModal()" title="关闭 (ESC)">关闭</button>
                </div>
            </div>
            
            <div class="image-display" id="imageDisplay">
                <div class="image-loading" id="imageLoading">正在加载图片...</div>
                <img id="previewImage" class="preview-image" src="" alt="" 
                     onclick="toggleZoom()"
                     onload="onImageLoaded()"
                     onerror="onImageError()">
            </div>
            
            <div class="thumbnail-grid" id="thumbnailGrid">
                <!-- 缩略图动态生成 -->
            </div>
        </div>
    </div>
    
    <script>
        let currentComic = null;
        let currentImageIndex = 0;
        let isFullscreen = false;
        let isZoomed = false;
        let isDragging = false;
        let startX = 0;
        let startY = 0;
        let scrollLeft = 0;
        let scrollTop = 0;
        let currentScale = 1;
        const scaleStep = 0.2;
        const minScale = 0.5;
        const maxScale = 3;
        
        // 加载漫画列表
        async function loadComics() {
            try {
                const response = await fetch('/api/comics');
                const comics = await response.json();
                
                const listElement = document.getElementById('comicList');
                const totalCount = document.getElementById('totalCount');
                const pendingCount = document.getElementById('pendingCount');
                const passedCount = document.getElementById('passedCount');
                const rejectedCount = document.getElementById('rejectedCount');
                const failedCount = document.getElementById('failedCount');
                
                totalCount.textContent = comics.total;
                pendingCount.textContent = comics.pending;
                passedCount.textContent = comics.passed;
				rejectedCount.textContent = comics.rejected;
                failedCount.textContent = comics.failed;
                
                if (comics.pendingList.length === 0) {
                    listElement.innerHTML = '<div class="loading">暂无等待审计的漫画</div>';
                    return;
                }
                
                let html = '';
                comics.pendingList.forEach(comic => {
                    html += ` + "`" + `
                    <div class="comic-item">
                        <div class="comic-info">
                            <h3>漫画 CID: ${comic.cid}</h3>
                            <div class="comic-meta">
                                图片数: ${comic.total} | 已修复: ${comic.repaired} | 修改时间: ${new Date(comic.modified).toLocaleString()}
                            </div>
                        </div>
                        <div class="comic-actions">
                            <button class="btn btn-view" onclick="viewComic('${comic.cid}')">查看图片</button>
                            <button class="btn btn-audit" onclick="auditComic('${comic.cid}')">审计通过</button>
                        </div>
                    </div>
                    ` + "`" + `;
                });
                
                listElement.innerHTML = html;
            } catch (error) {
                console.error('加载漫画列表失败:', error);
                document.getElementById('comicList').innerHTML = '<div class="loading">加载失败，请刷新页面</div>';
            }
        }
        
        // 查看漫画图片
		async function viewComic(cid) {
            currentCID = cid;
            try {
                const response = await fetch(` + "`" + `/api/comic/${cid}` + "`" + `);
                currentComic = await response.json();
                
                // 重置预览状态
                currentImageIndex = 0;
                isZoomed = false;
                currentScale = 1;
                
                // 初始化拖动事件
                setTimeout(() => {
                    initDragEvents();
                }, 100);
                
                // 更新图片预览
                updatePreviewImage();
                
                // 生成缩略图
                generateThumbnails();
                
                document.getElementById('imageModal').style.display = 'block';
            } catch (error) {
                console.error('加载漫画图片失败:', error);
                alert('加载图片失败');
            }
        }
        
        // 审计通过
        async function auditComic(cid) {            
            try {
                const response = await fetch(` + "`" + `/api/comic/${cid}/audit` + "`" + `, {
                    method: 'POST'
                });
                
                const result = await response.json();
                
                if (result.success) {
                    console.log(` + "`" + `漫画 ${cid} 已标记为审计通过` + "`" + `);
                    loadComics(); // 刷新列表
                } else {
                    alert(` + "`" + `操作失败: ${result.error}` + "`" + `);
                }
            } catch (error) {
                console.error('审计操作失败:', error);
                alert('操作失败，请重试');
            }
        }

		// 新增：审核不通过
        async function rejectCurrentComic() {
            if (!currentCID) return;
            
            try {
                const response = await fetch(` + "`" + `/api/comic/${currentCID}/reject` + "`" + `, {
                    method: 'POST'
                });
                
                const result = await response.json();
                
                if (result.success) {
                    console.log(` + "`" + `漫画 ${currentCID} 已标记为审核不通过` + "`" + `);
                    closeModal();
                    loadComics();
                } else {
                    alert(` + "`" + `操作失败: ${result.error}` + "`" + `);
                }
            } catch (error) {
                console.error('审核不通过操作失败:', error);
                alert('操作失败，请重试');
            }
        }
        
        // 新增：键盘快捷键支持
        document.addEventListener('keydown', (e) => {
            if (document.getElementById('imageModal').style.display === 'none') return;
            
            switch(e.key.toLowerCase()) {
                case 'arrowleft':
                    prevImage();
                    break;
                case 'arrowright':
                    nextImage();
                    break;
                case 'escape':
                    if (isFullscreen) {
                        toggleFullscreen();
                    } else {
                        closeModal();
                    }
                    break;
                case ' ':
                    toggleZoom();
                    e.preventDefault();
                    break;
                case 'r':
                    resetZoom();
                    e.preventDefault();
                    break;
                case 'a':
                    auditCurrentComic();
                    e.preventDefault();
                    break;
                case 'd':
                case 'r':
                    rejectCurrentComic();
                    e.preventDefault();
                    break;
                case '+':
                case '=':
                    zoomIn();
                    e.preventDefault();
                    break;
                case '-':
                case '_':
                    zoomOut();
                    e.preventDefault();
                    break;
            }
        });

		// 新增：全屏/退出全屏切换
		function toggleFullscreen() {
            const modalContent = document.getElementById('modalContent');
            const fullscreenBtn = document.getElementById('fullscreenBtn');
            
            if (!isFullscreen) {
                modalContent.classList.add('fullscreen');
                fullscreenBtn.textContent = '退出全屏';
                document.body.style.overflow = 'hidden';
                // 重置图片位置
                resetImagePosition();
            } else {
                modalContent.classList.remove('fullscreen');
                fullscreenBtn.textContent = '全屏';
                document.body.style.overflow = '';
                // 重置图片位置
                resetImagePosition();
            }
            isFullscreen = !isFullscreen;
            
            // 更新图片显示
            setTimeout(updatePreviewImage, 100);
        }
        
        // 新增：重置图片位置
        function resetImagePosition() {
            const imageDisplay = document.getElementById('imageDisplay');
            if (imageDisplay) {
                imageDisplay.scrollLeft = 0;
                imageDisplay.scrollTop = 0;
            }
        }
        
        // 新增：图片加载完成
        function onImageLoaded() {
            document.getElementById('imageLoading').style.display = 'none';
            const img = document.getElementById('previewImage');
            img.style.opacity = 1;
            
            // 重置缩放
            currentScale = 1;
            updateImageTransform();
        }
        
        // 新增：图片加载失败
        function onImageError() {
            document.getElementById('imageLoading').style.display = 'none';
            const img = document.getElementById('previewImage');
            img.src = 'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMTAwIiBoZWlnaHQ9IjEwMCIgdmlld0JveD0iMCAwIDEwMCAxMDAiIGZpbGw9Im5vbmUiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PHJlY3Qgd2lkdGg9IjEwMCIgaGVpZ2h0PSIxMDAiIGZpbGw9IiMzMzMiLz48cGF0aCBkPSJNMzAgMzBoNDBtMCA0MEg0ME0zMCA3MGg0MCIgc3Ryb2tlPSIjNjY2IiBzdHJva2Utd2lkdGg9IjgiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIvPjx0ZXh0IHg9IjUwIiB5PSI2NSIgdGV4dC1hbmNob3I9Im1pZGRsZSIgZm9udC1zaXplPSIxMiIgZmlsbD0iIzk5OSI+SW1hZ2UgRXJyb3I8L3RleHQ+PC9zdmc+';
            img.style.opacity = 0.5;
        }
        
        // 新增：切换到上一张图片
        function prevImage() {
            if (!currentComic || currentComic.images.length === 0) return;
            currentImageIndex = (currentImageIndex - 1 + currentComic.images.length) % currentComic.images.length;
            updatePreviewImage();
        }
        
        // 新增：切换到下一张图片
        function nextImage() {
            if (!currentComic || currentComic.images.length === 0) return;
            currentImageIndex = (currentImageIndex + 1) % currentComic.images.length;
            updatePreviewImage();
        }
        
        // 新增：切换图片缩放
        function toggleZoom() {
            if (currentScale === 1) {
                // 计算适合屏幕的缩放比例
                const img = document.getElementById('previewImage');
                const container = document.getElementById('imageDisplay');
                
                if (img.naturalWidth && img.naturalHeight) {
                    const containerWidth = container.clientWidth - 20; // 减去padding
                    const containerHeight = container.clientHeight - 20;
                    const scaleX = containerWidth / img.naturalWidth;
                    const scaleY = containerHeight / img.naturalHeight;
                    currentScale = Math.min(scaleX, scaleY, 1.5); // 不超过1.5倍
                } else {
                    currentScale = 1.5;
                }
            } else {
                currentScale = 1;
            }
            
            updateImageTransform();
            isZoomed = currentScale !== 1;
            updateZoomState();
        }
        
        // 新增：鼠标事件处理
        function initDragEvents() {
            const img = document.getElementById('previewImage');
            const container = document.getElementById('imageDisplay');
            
            img.addEventListener('mousedown', startDrag);
            img.addEventListener('touchstart', startDragTouch);
            
            container.addEventListener('wheel', handleWheel, { passive: false });
        }
        
        // 新增：开始拖动
        function startDrag(e) {
            if (!isZoomed) return;
            
            e.preventDefault();
            isDragging = true;
            startX = e.pageX;
            startY = e.pageY;
            scrollLeft = document.getElementById('imageDisplay').scrollLeft;
            scrollTop = document.getElementById('imageDisplay').scrollTop;
            
            document.addEventListener('mousemove', doDrag);
            document.addEventListener('mouseup', stopDrag);
        }
        
        // 新增：触摸开始
        function startDragTouch(e) {
            if (!isZoomed) return;
            
            e.preventDefault();
            isDragging = true;
            const touch = e.touches[0];
            startX = touch.pageX;
            startY = touch.pageY;
            scrollLeft = document.getElementById('imageDisplay').scrollLeft;
            scrollTop = document.getElementById('imageDisplay').scrollTop;
            
            document.addEventListener('touchmove', doDragTouch);
            document.addEventListener('touchend', stopDrag);
        }
        
        // 新增：执行拖动
        function doDrag(e) {
            if (!isDragging) return;
            e.preventDefault();
            
            const x = e.pageX;
            const y = e.pageY;
            const walkX = (x - startX) * 1.5;
            const walkY = (y - startY) * 1.5;
            
            const container = document.getElementById('imageDisplay');
            container.scrollLeft = scrollLeft - walkX;
            container.scrollTop = scrollTop - walkY;
        }
        
        // 新增：触摸拖动
        function doDragTouch(e) {
            if (!isDragging) return;
            e.preventDefault();
            
            const touch = e.touches[0];
            const x = touch.pageX;
            const y = touch.pageY;
            const walkX = (x - startX) * 1.5;
            const walkY = (y - startY) * 1.5;
            
            const container = document.getElementById('imageDisplay');
            container.scrollLeft = scrollLeft - walkX;
            container.scrollTop = scrollTop - walkY;
        }
        
        // 新增：停止拖动
        function stopDrag() {
            isDragging = false;
            document.removeEventListener('mousemove', doDrag);
            document.removeEventListener('touchmove', doDragTouch);
            document.removeEventListener('mouseup', stopDrag);
            document.removeEventListener('touchend', stopDrag);
        }
        
        // 新增：鼠标滚轮缩放
        function handleWheel(e) {
            if (!isZoomed) return;
            
            e.preventDefault();
            if (e.deltaY < 0) {
                zoomIn();
            } else {
                zoomOut();
            }
        }

        // 新增：更新预览图片
		function updatePreviewImage() {
            if (!currentComic || !currentComic.images[currentImageIndex]) return;
            
            document.getElementById('imageLoading').style.display = 'block';
            const previewImage = document.getElementById('previewImage');
            previewImage.style.opacity = 0;
            
            const imageUrl = ` + "`" + `/api/comic/${currentComic.cid}/images/${currentComic.images[currentImageIndex]}` + "`" + `;
            previewImage.src = imageUrl;
            
            // 更新图片信息
            document.getElementById('currentImageInfo').textContent = 
                ` + "`" + `漫画 ${currentComic.cid} - 图片 ${currentImageIndex + 1}/${currentComic.images.length}` + "`" + `;
            document.getElementById('imageCounter').textContent = 
                ` + "`" + `${currentImageIndex + 1} / ${currentComic.images.length}` + "`" + `;
            
            // 重置缩放
            currentScale = 1;
            isZoomed = false;
            previewImage.classList.remove('zoomed');
            resetImagePosition();
            updateImageTransform();
            
            // 更新缩略图激活状态
            updateThumbnailActive();
        }
        
        // 新增：更新图片变换
        function updateImageTransform() {
            const img = document.getElementById('previewImage');
            img.style.transform = ` + "`" + `scale(${currentScale})` + "`" + `;
            img.style.transformOrigin = 'center center';
        }
        
        // 新增：放大
        function zoomIn() {
            if (currentScale < maxScale) {
                currentScale += scaleStep;
                updateImageTransform();
                isZoomed = currentScale !== 1;
                updateZoomState();
            }
        }
        
        // 新增：缩小
        function zoomOut() {
            if (currentScale > minScale) {
                currentScale -= scaleStep;
                updateImageTransform();
                isZoomed = currentScale !== 1;
                updateZoomState();
            }
        }
        
        // 新增：重置缩放
        function resetZoom() {
            currentScale = 1;
            updateImageTransform();
            isZoomed = false;
            updateZoomState();
            resetImagePosition();
        }
        
        // 新增：更新缩放状态
        function updateZoomState() {
            const img = document.getElementById('previewImage');
            if (currentScale !== 1) {
                img.classList.add('zoomed');
                img.style.cursor = 'move';
                document.getElementById('imageHints').textContent = '按住拖动查看细节 | 滚轮缩放 | R重置缩放';
            } else {
                img.classList.remove('zoomed');
                img.style.cursor = 'pointer';
                document.getElementById('imageHints').textContent = '点击图片缩放，按住拖动查看细节';
            }
        }
        
        // 生成缩略图
		function generateThumbnails() {
            const thumbnailGrid = document.getElementById('thumbnailGrid');
            thumbnailGrid.innerHTML = '';
            
            if (!currentComic || currentComic.images.length === 0) return;
            
            currentComic.images.forEach((image, index) => {
                const imageUrl = ` + "`" + `/api/comic/${currentComic.cid}/images/${image}` + "`" + `;
                const thumbnailItem = document.createElement('div');
                thumbnailItem.style.position = 'relative';
                thumbnailItem.style.display = 'inline-block';
                
                const thumbnail = document.createElement('img');
                thumbnail.src = imageUrl;
                thumbnail.className = 'thumbnail';
                thumbnail.dataset.index = index;
                thumbnail.onclick = () => {
                    currentImageIndex = index;
                    updatePreviewImage();
                };
                
                const indexBadge = document.createElement('div');
                indexBadge.textContent = index + 1;
                indexBadge.style.position = 'absolute';
                indexBadge.style.top = '2px';
                indexBadge.style.left = '2px';
                indexBadge.style.background = 'rgba(0,0,0,0.7)';
                indexBadge.style.color = 'white';
                indexBadge.style.padding = '1px 4px';
                indexBadge.style.fontSize = '10px';
                indexBadge.style.borderRadius = '3px';
                
                thumbnailItem.appendChild(thumbnail);
                thumbnailItem.appendChild(indexBadge);
                thumbnailGrid.appendChild(thumbnailItem);
            });
            
            updateThumbnailActive();
        }
        
        // 新增：更新激活的缩略图
		function updateThumbnailActive() {
            document.querySelectorAll('.thumbnail').forEach((thumb, index) => {
                if (index === currentImageIndex) {
                    thumb.style.border = '2px solid #1890ff';
                } else {
                    thumb.style.border = '2px solid transparent';
                }
            });
        }
        
        // 审计当前查看的漫画
        function auditCurrentComic() {
            if (currentCID) {
                auditComic(currentCID);
                closeModal();
            }
        }
        
        // 关闭模态框
        function closeModal() {
            document.getElementById('imageModal').style.display = 'none';
            if (isFullscreen) {
                toggleFullscreen();
            }
            currentCID = '';
            currentComic = null;
        }
        
        // 页面加载完成后执行
        window.onload = function() {
            loadComics();
            // 每30秒自动刷新一次
            setInterval(loadComics, 30000);
        };
        
        // 点击模态框外部关闭
        window.onclick = function(event) {
            const modal = document.getElementById('imageModal');
            if (event.target === modal) {
                closeModal();
            }
        };
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// 获取所有漫画列表
func getComicsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 获取等待审计的漫画
	pendingComics, err := getComicsFromDir(SuccDir)
	if err != nil {
		sendError(w, "获取等待审计漫画失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 获取已通过审计的漫画
	passedComics, err := getComicsFromDir(AuditDir)
	if err != nil {
		sendError(w, "获取已通过审计漫画失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 获取修复失败的漫画
	failedComics, err := getComicsFromDir(FailDir)
	if err != nil {
		sendError(w, "获取修复失败漫画失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rejectedComics, err := getComicsFromDir(AuditFailDir)
	if err != nil {
		sendError(w, "获取审核不通过漫画失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"total":        len(pendingComics) + len(passedComics) + len(failedComics) + len(rejectedComics),
		"pending":      len(pendingComics),
		"passed":       len(passedComics),
		"failed":       len(failedComics),
		"rejected":     len(rejectedComics),
		"pendingList":  pendingComics,
		"passedList":   passedComics,
		"failedList":   failedComics,
		"rejectedList": rejectedComics,
	}

	json.NewEncoder(w).Encode(response)
}

// 获取指定漫画的图片列表
func getComicImagesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid := vars["cid"]

	w.Header().Set("Content-Type", "application/json")

	// 检查漫画是否存在
	comicPath := filepath.Join(SuccDir, cid)
	if _, err := os.Stat(comicPath); os.IsNotExist(err) {
		sendError(w, "漫画不存在: "+cid, http.StatusNotFound)
		return
	}

	// 获取图片列表
	images, err := getImagesFromComicDir(comicPath)
	if err != nil {
		sendError(w, "获取图片列表失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 获取图片尺寸信息
	var imageInfos []map[string]any
	for _, img := range images {
		imgPath := filepath.Join(comicPath, img)
		imgInfo := map[string]any{
			"filename": img,
			"size":     0,
			"width":    0,
			"height":   0,
		}

		// 获取文件大小
		if info, err := os.Stat(imgPath); err == nil {
			imgInfo["size"] = info.Size()
		}

		// 获取图片尺寸
		if width, height, err := getImageDimensions(imgPath); err == nil {
			imgInfo["width"] = width
			imgInfo["height"] = height
		}

		imageInfos = append(imageInfos, imgInfo)
	}

	response := map[string]any{
		"cid":        cid,
		"images":     images,
		"imageInfos": imageInfos,
		"total":      len(images),
	}

	json.NewEncoder(w).Encode(response)
}

// 新增：获取图片尺寸
func getImageDimensions(fp string) (int, int, error) {
	file, err := os.Open(fp)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	// 根据文件扩展名使用不同的解析方法
	ext := strings.ToLower(filepath.Ext(fp))

	switch ext {
	case ".jpg", ".jpeg":
		return getJPEGDimensions(file)
	case ".png":
		return getPNGDimensions(file)
	case ".webp":
		return getWebPDimensions(file)
	default:
		return 0, 0, fmt.Errorf("unsupported image format: %s", ext)
	}
}

// 简化版本：只读取文件头获取尺寸
func getJPEGDimensions(file *os.File) (int, int, error) {
	// JPEG文件标记
	header := make([]byte, 2)
	if _, err := file.Read(header); err != nil {
		return 0, 0, err
	}

	if header[0] != 0xFF || header[1] != 0xD8 {
		return 0, 0, fmt.Errorf("not a valid JPEG file")
	}

	// 简化处理，实际应用中可能需要更复杂的解析
	return 0, 0, nil
}

func getPNGDimensions(file *os.File) (int, int, error) {
	return 0, 0, nil
}

func getWebPDimensions(file *os.File) (int, int, error) {
	return 0, 0, nil
}

// 提供图片文件
func serveImageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid := vars["cid"]
	filename := vars["filename"]

	// 检查文件是否存在
	filePath := filepath.Join(SuccDir, cid, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "图片不存在", http.StatusNotFound)
		return
	}

	// 设置缓存头
	w.Header().Set("Cache-Control", "public, max-age=86400") // 缓存1天

	// 设置正确的Content-Type
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	// 设置文件大小
	if info, err := os.Stat(filePath); err == nil {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	}

	// 提供文件
	http.ServeFile(w, r, filePath)
}

// 审计通过，移动到审计目录
func auditComicHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid := vars["cid"]

	w.Header().Set("Content-Type", "application/json")

	// 检查漫画是否存在
	srcPath := filepath.Join(SuccDir, cid)
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		sendError(w, "漫画不存在: "+cid, http.StatusNotFound)
		return
	}

	// 移动到审计目录
	dstPath := filepath.Join(AuditDir, cid)
	if err := os.Rename(srcPath, dstPath); err != nil {
		sendError(w, "移动文件失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": fmt.Sprintf("漫画 %s 已审计通过", cid),
		"oldPath": srcPath,
		"newPath": dstPath,
	})
}

// 新增：审核不通过，移动到审计不通过目录
func rejectComicHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid := vars["cid"]

	w.Header().Set("Content-Type", "application/json")

	// 检查漫画是否存在
	srcPath := filepath.Join(SuccDir, cid)
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		sendError(w, "漫画不存在: "+cid, http.StatusNotFound)
		return
	}

	// 移动到审计不通过目录
	dstPath := filepath.Join("./audit_failed", cid)
	if err := os.Rename(srcPath, dstPath); err != nil {
		sendError(w, "移动文件失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": fmt.Sprintf("漫画 %s 已标记为审核不通过", cid),
		"oldPath": srcPath,
		"newPath": dstPath,
	})
}

// 新增：获取审核不通过的漫画图片列表
func getRejectedComicImagesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid := vars["cid"]

	w.Header().Set("Content-Type", "application/json")

	// 检查漫画是否存在
	comicPath := filepath.Join("./audit_failed", cid)
	if _, err := os.Stat(comicPath); os.IsNotExist(err) {
		sendError(w, "漫画不存在: "+cid, http.StatusNotFound)
		return
	}

	// 获取图片列表
	images, err := getImagesFromComicDir(comicPath)
	if err != nil {
		sendError(w, "获取图片列表失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"cid":    cid,
		"images": images,
		"total":  len(images),
	}

	json.NewEncoder(w).Encode(response)
}

// 新增：提供审核不通过的图片文件
func serveRejectedImageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid := vars["cid"]
	filename := vars["filename"]

	// 检查文件是否存在
	filePath := filepath.Join("./audit_failed", cid, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "图片不存在", http.StatusNotFound)
		return
	}

	// 设置正确的Content-Type
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	// 提供文件
	http.ServeFile(w, r, filePath)
}

// ==================== 辅助函数 ====================

// 从目录获取漫画列表
func getComicsFromDir(dirPath string) ([]ComicInfo, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		// 目录可能不存在
		return []ComicInfo{}, nil
	}

	var comics []ComicInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		cid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		comicPath := filepath.Join(dirPath, entry.Name())

		// 获取目录信息
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 获取图片列表
		images, err := getImagesFromComicDir(comicPath)
		if err != nil {
			continue
		}

		// 获取修复后的图片数量
		repairedCount := 0
		for _, img := range images {
			if strings.Contains(img, "-repaired") {
				repairedCount++
			}
		}

		comics = append(comics, ComicInfo{
			CID:      cid,
			Images:   images,
			Total:    len(images),
			Repaired: repairedCount,
			Modified: info.ModTime(),
			Status:   filepath.Base(dirPath),
		})
	}

	return comics, nil
}

// 从漫画目录获取图片列表
func getImagesFromComicDir(dirPath string) ([]string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var images []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		ext := strings.ToLower(filepath.Ext(filename))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" {
			images = append(images, filename)
		}
	}

	return images, nil
}

// 确保静态目录存在
func ensureStaticDir() {
	if _, err := os.Stat("static"); os.IsNotExist(err) {
		os.MkdirAll("static", 0o755)
	}
}

// 发送错误响应
func sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error":   message,
	})
}

// ==================== MongoDB相关函数 ====================

func getCIDsFromMongo() ([]int, error) {
	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.Background())

	collection := client.Database(DatabaseName).Collection(CollectionName)
	filter := bson.M{"archive": bson.M{"$exists": false}}

	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var cidList []int
	for cursor.Next(context.Background()) {
		var result struct {
			CID int `bson:"cid"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		cidList = append(cidList, result.CID)
	}

	return cidList, nil
}

// ==================== API调用相关函数 ====================

func callArchiveAPI(client *http.Client, cid int) (*APIResponse, error) {
	apiURL := fmt.Sprintf("%s/v2/api/nhcomic/%d/archive", serverAddr, cid)

	req, err := http.NewRequest("POST", apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("archive API返回非200状态码: %d", resp.StatusCode)
	}

	// 解析响应
	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return &apiResp, nil
}

func queryImageSaveDir(client *http.Client, cid int) (string, error) {
	apiURL := fmt.Sprintf("%s/v2/api/nhcomic/%d/cover", serverAddr, cid)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cover API返回非200状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if len(body) == 0 {
		return "", fmt.Errorf("cover API返回空响应")
	}

	return filepath.Dir(string(body)), nil
}

func callCommitAPI(client *http.Client, cid int) (*APIResponse, error) {
	apiURL := fmt.Sprintf("%s/v2/api/nhcomic/verify", serverAddr)

	body := strings.NewReader(fmt.Sprintf(`{"id":"%d","autoFix":true,"maxWorkers":1}`, cid))

	req, err := http.NewRequest("POST", apiURL, body)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("commit API返回非200状态码: %d", resp.StatusCode)
	}

	// 解析响应
	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return &apiResp, nil
}

// ==================== Worker相关函数 ====================

func worker(id int, taskChan <-chan Task, wg *sync.WaitGroup) {
	defer wg.Done()

	for task := range taskChan {
		fmt.Printf("[Worker %d] 开始处理CID: %d\n", id, task.CID)

		// 创建CID目录
		cidDir := fmt.Sprintf("./%d", task.CID)
		if err := os.MkdirAll(cidDir, 0o755); err != nil {
			fmt.Printf("[CID:%d] 创建目录失败: %v\n", task.CID, err)
			moveToFail(task.CID)
			continue
		}

		allSuccess := true
		var failedImages []string

		// 下载每个异常图片
		for _, img := range task.ResBody.Body.InvalidImages {
			imgPath := fmt.Sprintf("%s/%d", cidDir, img.Index)
			imgURL := fmt.Sprintf("%s/g/%d/%d", serverAddr, task.CID, img.Index)

			// 下载图片
			imageType, err := downloadImage(imgURL, imgPath)
			if err != nil {
				fmt.Printf("[CID:%d] 下载图片 %d 失败: %v\n", task.CID, img.Index, err)
				allSuccess = false
				failedImages = append(failedImages, fmt.Sprintf("index:%d", img.Index))
				continue
			}

			// 如果是jpg图片，执行修复
			if strings.ToLower(imageType) == "jpg" || strings.ToLower(imageType) == "jpeg" {
				originalFile := fmt.Sprintf("%s.jpg", imgPath)
				repairedFile := fmt.Sprintf("%s-repaired.jpg", imgPath)

				if err := repairJPEG(originalFile, repairedFile); err != nil {
					fmt.Printf("[CID:%d] 修复图片 %d 失败: %v\n", task.CID, img.Index, err)
					allSuccess = false
					failedImages = append(failedImages, fmt.Sprintf("index:%d(修复失败)", img.Index))
				} else {
					fmt.Printf("[CID:%d] 图片 %d 修复成功\n", task.CID, img.Index)
				}
			} else {
				fmt.Printf("[CID:%d] 图片 %d 类型为 %s, 无需修复\n", task.CID, img.Index, imageType)
			}
		}

		// 根据处理结果移动目录
		if allSuccess {
			if err := moveToSucc(task.CID); err != nil {
				fmt.Printf("[CID:%d] 移动到succ目录失败: %v\n", task.CID, err)
			} else {
				fmt.Printf("[CID:%d] 所有图片处理成功，已移动到succ目录\n", task.CID)
			}
		} else {
			if err := moveToFail(task.CID); err != nil {
				fmt.Printf("[CID:%d] 移动到fail目录失败: %v\n", task.CID, err)
			} else {
				fmt.Printf("[CID:%d] 部分图片处理失败，已移动到fail目录。失败项: %v\n", task.CID, failedImages)
			}
		}
	}
}

func downloadImage(url, basePath string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	var extension string

	switch {
	case strings.Contains(contentType, "image/jpeg"):
		extension = "jpg"
	case strings.Contains(contentType, "image/png"):
		extension = "png"
	case strings.Contains(contentType, "image/webp"):
		extension = "webp"
	default:
		extension = "jpg"
	}

	filePath := fmt.Sprintf("%s.%s", basePath, extension)
	out, err := os.Create(filePath)
	if err != nil {
		return extension, err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return extension, err
}

func repairJPEG(inputFile, outputFile string) error {
	cmd := exec.Command("jpegtran", "-copy", "none", "-optimize", inputFile)

	out, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer out.Close()

	cmd.Stdout = out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("执行jpegtran失败: %v", err)
	}

	return nil
}

func moveToSucc(cid int) error {
	src := fmt.Sprintf("./%d", cid)
	dst := fmt.Sprintf("%s/%d", SuccDir, cid)

	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}

	return os.Rename(src, dst)
}

func moveToFail(cid int) error {
	src := fmt.Sprintf("./%d", cid)
	dst := fmt.Sprintf("%s/%d", FailDir, cid)

	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}

	return os.Rename(src, dst)
}
