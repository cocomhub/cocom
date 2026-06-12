// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import "time"

// Config 是 cocom 所有 Viper 配置的结构化映射。
// 使用 viper.Unmarshal() 反序列化，与全局 viper 单例同步更新。
type Config struct {
	Cocom     CocomConfig     `mapstructure:"cocom"`
	Server    ServerConfig    `mapstructure:"server"`
	Archive   ArchiveConfig   `mapstructure:"archive"`
	Recommend RecommendConfig `mapstructure:"recommend"`
	Comic     ComicConfig     `mapstructure:"comic"`
	Mongo     MongoConfig     `mapstructure:"mongo"`
	Log       LogConfig       `mapstructure:"log"`
	Download  DownloadConfig  `mapstructure:"download"`
}

// ======================== Cocom ========================

type CocomConfig struct {
	Storage CocomStorageConfig `mapstructure:"storage"`
	Archive CocomArchiveConfig `mapstructure:"archive"`
	Cache   CacheConfig        `mapstructure:"cache"`
}

type CocomStorageConfig struct {
	Path string `mapstructure:"path"`
}

type CocomArchiveConfig struct {
	Path      string `mapstructure:"path"`
	TempPath  string `mapstructure:"temp_path"`
	Password  string `mapstructure:"password"`
	Cmd       string `mapstructure:"cmd"`
	Replicate bool   `mapstructure:"replicate"`
}

// ======================== Cache ========================

type CacheConfig struct {
	CleanInterval    time.Duration `mapstructure:"cleanInterval"`
	EvictionInterval time.Duration `mapstructure:"evictionInterval"`
}

// ======================== Server ========================

type ServerConfig struct {
	Port            int32        `mapstructure:"port"`
	AccessLog       AccessLogCfg `mapstructure:"access_log"`
	CORS            CORSCfg      `mapstructure:"cors"`
	Gzip            GzipCfg      `mapstructure:"gzip"`
	RateLimit       RateLimitCfg `mapstructure:"ratelimit"`
	Scheduler       SchedulerCfg `mapstructure:"scheduler"`
	Listen          ListenCfg    `mapstructure:"listen"`
	ShutdownTimeout string       `mapstructure:"shutdown_timeout"`
}

type AccessLogCfg struct {
	Patterns []string `mapstructure:"patterns"`
}

type CORSCfg struct {
	Enabled      bool   `mapstructure:"enabled"`
	AllowOrigins string `mapstructure:"allow_origins"`
	AllowMethods string `mapstructure:"allow_methods"`
	AllowHeaders string `mapstructure:"allow_headers"`
}

type GzipCfg struct {
	Enabled bool `mapstructure:"enabled"`
	Level   int  `mapstructure:"level"`
}

type RateLimitCfg struct {
	Enabled bool `mapstructure:"enabled"`
	RPS     int  `mapstructure:"rps"`
	Burst   int  `mapstructure:"burst"`
}

type SchedulerCfg struct {
	Enabled            bool              `mapstructure:"enabled"`
	Timezone           string            `mapstructure:"timezone"`
	ProbeComic         SchedulerTaskCfg  `mapstructure:"probe_comic"`
	ArchiveStatusCheck SchedulerTaskCfg  `mapstructure:"archive_status_check"`
	CocomaArchiver     CocomaArchiverCfg `mapstructure:"cocoma_archiver"`
}

type SchedulerTaskCfg struct {
	Enabled  bool     `mapstructure:"enabled"`
	Name     string   `mapstructure:"name"`
	Cron     string   `mapstructure:"cron"`
	Tags     []string `mapstructure:"tags"`
	Limit    int      `mapstructure:"limit"`
	MaxConn  int      `mapstructure:"max_conn"`
	Backends []string `mapstructure:"backends"`
}

type CocomaArchiverCfg struct {
	Enabled     bool   `mapstructure:"enabled"`
	Cron        string `mapstructure:"cron"`
	Limit       int    `mapstructure:"limit"`
	CIDRegex    string `mapstructure:"cid_regex"`
	ScanDir     string `mapstructure:"scan_dir"`
	ArchiveDir  string `mapstructure:"archive_dir"`
	NotmatchDir string `mapstructure:"notmatch_dir"`
}

type ListenCfg struct {
	HTTP ListenAddrCfg `mapstructure:"http"`
	TLS  ListenTLSCfg  `mapstructure:"tls"`
	Unix ListenUnixCfg `mapstructure:"unix"`
}

type ListenAddrCfg struct {
	Addr string `mapstructure:"addr"`
}

type ListenTLSCfg struct {
	Cert string `mapstructure:"cert"`
	Key  string `mapstructure:"key"`
}

type ListenUnixCfg struct {
	Path string `mapstructure:"path"`
}

// ======================== Archive ========================

type ArchiveConfig struct {
	Password  string           `mapstructure:"password"`
	Cmd       string           `mapstructure:"cmd"`
	Replicate bool             `mapstructure:"replicate"`
	RootDir   string           `mapstructure:"root_dir"`
	Algorithm ArchiveAlgoCfg   `mapstructure:"algorithm"`
	Manager   ArchiveMgrConfig `mapstructure:"manager"`
}

type ArchiveAlgoCfg struct {
	Single AlgoCfg `mapstructure:"single"`
	Double AlgoCfg `mapstructure:"double"`
}

type AlgoCfg struct {
	Concurrency int `mapstructure:"concurrency"`
}

// ======================== Archive Manager ========================

type ArchiveMgrConfig struct {
	Algorithm          string           `mapstructure:"algorithm"`
	MetaRecordFileList bool             `mapstructure:"meta_record_file_list"`
	Replicates         []string         `mapstructure:"replicates"`
	Index              ArchiveMgrIdxCfg `mapstructure:"index"`
}

type ArchiveMgrIdxCfg struct {
	Type            string `mapstructure:"type"`
	FileStoreName   string `mapstructure:"file_store_name"`
	FileStorePrefix string `mapstructure:"file_store_prefix"`
	MongoDatabase   string `mapstructure:"mongo_database"`
	MongoCollection string `mapstructure:"mongo_collection"`
	MongoPrefix     string `mapstructure:"mongo_prefix"`
	MongoIDField    string `mapstructure:"mongo_id_field"`
	MongoNameField  string `mapstructure:"mongo_name_field"`
}

// ======================== Recommend ========================

type RecommendConfig struct {
	Limit int `mapstructure:"limit"`
}

// ======================== Comic ========================

type ComicConfig struct {
	Mongo    ComicMongoCfg    `mapstructure:"mongo"`
	Verify   ComicVerifyCfg   `mapstructure:"verify"`
	Download ComicDownloadCfg `mapstructure:"download"`
}

type ComicMongoCfg struct {
	Database    string                   `mapstructure:"database"`
	Collections ComicMongoCollectionsCfg `mapstructure:"collections"`
}

type ComicMongoCollectionsCfg struct {
	ComicInfo    string `mapstructure:"comicInfo"`
	OneComicInfo string `mapstructure:"oneComicInfo"`
	VideoInfo    string `mapstructure:"videoInfo"`
	Settings     string `mapstructure:"settings"`
	Custom       string `mapstructure:"custom"`
	ComicTag     string `mapstructure:"comicTag"`
	TagRelation  string `mapstructure:"tagRelation"`
}

type ComicVerifyCfg struct {
	Concurrent     int `mapstructure:"concurrent"`
	TaskBufferSize int `mapstructure:"task_buffer_size"`
}

type ComicDownloadCfg struct {
	MaxDownloadSize int32 `mapstructure:"maxDownloadSize"`
}

// ======================== Mongo ========================

type MongoConfig struct {
	User       string `mapstructure:"user"`
	Password   string `mapstructure:"password"`
	Host       string `mapstructure:"host"`
	Database   string `mapstructure:"database"`
	AuthSource string `mapstructure:"authSource"`
}

// ======================== Log ========================

type LogConfig struct {
	EnableFile      bool   `mapstructure:"enableFile"`
	Filename        string `mapstructure:"filename"`
	MaxSize         int    `mapstructure:"maxSize"`
	MaxAge          int    `mapstructure:"maxAge"`
	MaxBackups      int    `mapstructure:"maxBackups"`
	LocalTime       bool   `mapstructure:"localtime"`
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

// ======================== Download ========================

type DownloadConfig struct {
	MaxRunning  int    `mapstructure:"maxRunning"`
	DownloadDir string `mapstructure:"downloadDir"`
}
