// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"time"

	"github.com/cocomhub/cocom/pkg/download"
	"github.com/cocomhub/cocom/pkg/logging"
	"github.com/cocomhub/cocom/pkg/mongowrap"
	"github.com/cocomhub/cocom/pkg/storage"
)

// Config 是整个应用的 Viper 配置映射结构体。
// 所有字段均通过 mapstructure 标签与 viper 键一一对应。
type Config struct {
	Cocom     Cocom            `mapstructure:"cocom"`
	Archive   Archive          `mapstructure:"archive"`
	Server    Server           `mapstructure:"server"`
	Log       logging.Config   `mapstructure:"log"`
	Mongo     mongowrap.Config `mapstructure:"mongo"`
	Comic     Comic            `mapstructure:"comic"`
	Download  download.Config  `mapstructure:"download"`
	Recommend Recommend        `mapstructure:"recommend"`
	Client    Client           `mapstructure:"client"`
}

// Client 客户端配置。
type Client struct {
	ServerAddr string `mapstructure:"server_addr"`
}

// Cocom 顶层 cocom 相关子配置。
type Cocom struct {
	Storage CocomStorage `mapstructure:"storage"`
	Archive CocomArchive `mapstructure:"archive"`
	Cache   CocomCache   `mapstructure:"cache"`
}

type CocomStorage struct {
	Path     string           `mapstructure:"path"`
	Backends []storage.Config `mapstructure:"backends"`
}

type CocomArchive struct {
	Path      string      `mapstructure:"path"`
	TempPath  string      `mapstructure:"temp_path"`
	Password  string      `mapstructure:"password"`
	Cmd       string      `mapstructure:"cmd"`
	Replicate bool        `mapstructure:"replicate"`
	Algorithm ArchiveAlgo `mapstructure:"algorithm"`
}

// ArchiveAlgo 归档算法并发数配置。
type ArchiveAlgo struct {
	Single ArchiveAlgoConcurrency `mapstructure:"single"`
	Double ArchiveAlgoConcurrency `mapstructure:"double"`
}

// ArchiveAlgoConcurrency 单种算法的并发数。
type ArchiveAlgoConcurrency struct {
	Concurrency int `mapstructure:"concurrency"`
}

type CocomCache struct {
	CleanInterval    time.Duration `mapstructure:"cleanInterval"`
	EvictionInterval time.Duration `mapstructure:"evictionInterval"`
}

// Server 服务端配置。
// 唯一监听入口为 Server.Listen.HTTP.Addr，不再使用 Port/Host 双字段。
type Server struct {
	Listen          Listen    `mapstructure:"listen"`
	AccessLog       AccessLog `mapstructure:"access_log"`
	CORS            CORS      `mapstructure:"cors"`
	Gzip            Gzip      `mapstructure:"gzip"`
	RateLimit       RateLimit `mapstructure:"ratelimit"`
	Admin           Admin     `mapstructure:"admin"`
	ShutdownTimeout string    `mapstructure:"shutdown_timeout"`
	Scheduler       Scheduler `mapstructure:"scheduler"`
}

// Listen 监听配置组。
type Listen struct {
	HTTP ListenAddr `mapstructure:"http"`
	TLS  ListenTLS  `mapstructure:"tls"`
	Unix ListenUnix `mapstructure:"unix"`
}

type ListenAddr struct {
	Addr string `mapstructure:"addr"`
}

type ListenTLS struct {
	Cert string `mapstructure:"cert"`
	Key  string `mapstructure:"key"`
}

type ListenUnix struct {
	Path string `mapstructure:"path"`
}

type AccessLog struct {
	Patterns []string `mapstructure:"patterns"`
}

type CORS struct {
	Enabled      bool   `mapstructure:"enabled"`
	AllowOrigins string `mapstructure:"allow_origins"`
	AllowMethods string `mapstructure:"allow_methods"`
	AllowHeaders string `mapstructure:"allow_headers"`
}

type Gzip struct {
	Enabled bool `mapstructure:"enabled"`
	Level   int  `mapstructure:"level"`
}

type RateLimit struct {
	Enabled bool `mapstructure:"enabled"`
	RPS     int  `mapstructure:"rps"`
	Burst   int  `mapstructure:"burst"`
}

type Admin struct {
	Token       string `mapstructure:"token"`
	AllowRemote bool   `mapstructure:"allow_remote"`
}

// Scheduler 调度器及其子任务配置。
type Scheduler struct {
	Enabled            bool           `mapstructure:"enabled"`
	Timezone           string         `mapstructure:"timezone"`
	ProbeComic         SchedulerTask  `mapstructure:"probe_comic"`
	ArchiveStatusCheck SchedulerTask  `mapstructure:"archive_status_check"`
	CocomaArchiver     CocomaArchiver `mapstructure:"cocoma_archiver"`
}

// SchedulerTask 调度子任务配置（probe_comic 和 archive_status_check 通用）。
type SchedulerTask struct {
	Enabled  bool     `mapstructure:"enabled"`
	Name     string   `mapstructure:"name"`
	Cron     string   `mapstructure:"cron"`
	Tags     []string `mapstructure:"tags"`
	Limit    int      `mapstructure:"limit"`
	MaxConn  int      `mapstructure:"max_conn"`
	Backends []string `mapstructure:"backends"`
}

// CocomaArchiver 中的 notmatch_dir 字段，在 mapstructure 和字段名前缀上保持一致。
type CocomaArchiver struct {
	Enabled     bool   `mapstructure:"enabled"`
	Cron        string `mapstructure:"cron"`
	Limit       int    `mapstructure:"limit"`
	CIDRegex    string `mapstructure:"cid_regex"`
	ScanDir     string `mapstructure:"scan_dir"`
	ArchiveDir  string `mapstructure:"archive_dir"`
	NotMatchDir string `mapstructure:"notmatch_dir"`
}

// Archive 归档压缩配置。
type Archive struct {
	Password  string              `mapstructure:"password"`
	Cmd       string              `mapstructure:"cmd"`
	Replicate bool                `mapstructure:"replicate"`
	RootDir   string              `mapstructure:"root_dir"`
	Algorithm ArchiveAlgorithmSet `mapstructure:"algorithm"`
	Manager   ArchiveManager      `mapstructure:"manager"`
}

type ArchiveAlgorithmSet struct {
	Single ArchiveAlgorithm `mapstructure:"single"`
	Double ArchiveAlgorithm `mapstructure:"double"`
}

type ArchiveAlgorithm struct {
	Concurrency int `mapstructure:"concurrency"`
}

// ArchiveManager 归档管理器配置 — 不能直接引用 pkg/archive/manager 以免循环依赖，
// 因此保持本地定义，与 manager.Config 保持同步。
type ArchiveManager struct {
	Algorithm          string       `mapstructure:"algorithm"`
	MetaRecordFileList bool         `mapstructure:"meta_record_file_list"`
	Replicates         []string     `mapstructure:"replicates"`
	Index              ArchiveIndex `mapstructure:"index"`
}

type ArchiveIndex struct {
	Type            string `mapstructure:"type"`
	FileStoreName   string `mapstructure:"file_store_name"`
	FileStorePrefix string `mapstructure:"file_store_prefix"`
	MongoDatabase   string `mapstructure:"mongo_database"`
	MongoCollection string `mapstructure:"mongo_collection"`
	MongoPrefix     string `mapstructure:"mongo_prefix"`
	MongoIDField    string `mapstructure:"mongo_id_field"`
	MongoNameField  string `mapstructure:"mongo_name_field"`
}

// Log 日志配置（注意：Localtime 字段对应 viper 键 log.localtime，全小写）。
// 已迁移到 pkg/logging.Config。此处保留本地副本确保 mapstructure 对齐。
type Log struct {
	EnableFile      bool   `mapstructure:"enableFile"`
	Filename        string `mapstructure:"filename"`
	MaxSize         int    `mapstructure:"maxSize"`
	MaxAge          int    `mapstructure:"maxAge"`
	MaxBackups      int    `mapstructure:"maxBackups"`
	Localtime       bool   `mapstructure:"localtime"`
	Compress        bool   `mapstructure:"compress"`
	EnableConsole   bool   `mapstructure:"enableConsole"`
	EnableCaller    bool   `mapstructure:"enableCaller"`
	EnableSourceIP  bool   `mapstructure:"enableSourceIP"`
	EnablePID       bool   `mapstructure:"enablePID"`
	FileLevel       string `mapstructure:"fileLevel"`
	ConsoleLevel    string `mapstructure:"consoleLevel"`
	FileEncoding    string `mapstructure:"fileEncoding"`
	ConsoleEncoding string `mapstructure:"consoleEncoding"`
	AppName         string `mapstructure:"appName"`
	SourceEth       string `mapstructure:"sourceEth"`
	DisableTraceID  bool   `mapstructure:"disableTraceID"`
}

// Mongo MongoDB 连接配置。
// 已迁移到 pkg/mongowrap.Config。此处保留本地副本确保 mapstructure 对齐。
type Mongo struct {
	User       string `mapstructure:"user"`
	Password   string `mapstructure:"password"`
	Host       string `mapstructure:"host"`
	Database   string `mapstructure:"database"`
	AuthSource string `mapstructure:"authSource"`
}

// Comic 漫画业务配置。
type Comic struct {
	Verify   ComicVerify   `mapstructure:"verify"`
	Download ComicDownload `mapstructure:"download"`
	Mongo    ComicMongo    `mapstructure:"mongo"`
}

type ComicVerify struct {
	Concurrent     int `mapstructure:"concurrent"`
	TaskBufferSize int `mapstructure:"task_buffer_size"`
}

type ComicDownload struct {
	MaxDownloadSize int32 `mapstructure:"maxDownloadSize"`
}

type ComicMongo struct {
	Database    string         `mapstructure:"database"`
	Collections ComicMongoColl `mapstructure:"collections"`
}

type ComicMongoColl struct {
	ComicInfo    string `mapstructure:"comicInfo"`
	OneComicInfo string `mapstructure:"oneComicInfo"`
	VideoInfo    string `mapstructure:"videoInfo"`
	Settings     string `mapstructure:"settings"`
	Custom       string `mapstructure:"custom"`
	ComicTag     string `mapstructure:"comicTag"`
	TagRelation  string `mapstructure:"tagRelation"`
}

// Download 下载模块配置。
// 已迁移到 pkg/download.Config。此处保留本地副本确保 mapstructure 对齐。
type Download struct {
	MaxRunning  int    `mapstructure:"maxRunning"`
	DownloadDir string `mapstructure:"downloadDir"`
}

// Recommend 推荐系统配置。
type Recommend struct {
	Limit int `mapstructure:"limit"`
}
