// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

// Config 是 cocom 所有 Viper 配置的结构化映射。
// 使用 viper.Unmarshal() 反序列化，与全局 viper 单例同步更新。
type Config struct {
	Cocom     CocomConfig     `mapstructure:"cocom"`
	Server    ServerConfig    `mapstructure:"server"`
	Archive   ArchiveConfig   `mapstructure:"archive"`
	Recommend RecommendConfig `mapstructure:"recommend"`
	Comic     ComicConfig     `mapstructure:"comic"`
	Mongo     MongoConfig     `mapstructure:"mongo"`
}

type CocomConfig struct {
	Storage CocomStorageConfig `mapstructure:"storage"`
	Archive CocomArchiveConfig `mapstructure:"archive"`
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

type ArchiveConfig struct {
	Password  string         `mapstructure:"password"`
	Cmd       string         `mapstructure:"cmd"`
	Replicate bool           `mapstructure:"replicate"`
	RootDir   string         `mapstructure:"root_dir"`
	Algorithm ArchiveAlgoCfg `mapstructure:"algorithm"`
}

type ArchiveAlgoCfg struct {
	Single AlgoCfg `mapstructure:"single"`
	Double AlgoCfg `mapstructure:"double"`
}

type AlgoCfg struct {
	Concurrency int `mapstructure:"concurrency"`
}

type RecommendConfig struct {
	Limit int `mapstructure:"limit"`
}

type ComicConfig struct {
	Mongo ComicMongoCfg `mapstructure:"mongo"`
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

type MongoConfig struct {
	User       string `mapstructure:"user"`
	Password   string `mapstructure:"password"`
	Host       string `mapstructure:"host"`
	Database   string `mapstructure:"database"`
	AuthSource string `mapstructure:"authSource"`
}
