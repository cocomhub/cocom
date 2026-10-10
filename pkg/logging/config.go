// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package logging

// Config is the configuration for the logging package.
type Config struct {
	// EnableFile determines if the log should be writed to local file.
	EnableFile bool `json:"enableFile" yaml:"enableFile"`

	// Filename is the file to write logs to.  Backup log files will be retained
	// in the same directory.  It uses <processname>-lumberjack.log in
	// os.TempDir() if empty.
	Filename string `json:"filename" yaml:"filename"`

	// MaxSize is the maximum size in megabytes of the log file before it gets
	// rotated. It defaults to 100 megabytes.
	MaxSize int `json:"maxSize" yaml:"maxSize"`

	// MaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	MaxAge int `json:"maxAge" yaml:"maxAge"`

	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files (though MaxAge may still cause them to get
	// deleted.)
	MaxBackups int `json:"maxBackups" yaml:"maxBackups"`

	// DisableLocalTime 为 true 时使用 UTC（默认 false：使用本机本地时间）。
	// 命名约定：bool 配置一律以「关闭某能力」表达并默认 false（见 AGENTS 7c），
	// 避免「未配置」与「显式关闭」语义混淆。
	DisableLocalTime bool `json:"disableLocalTime" yaml:"disableLocalTime"`

	// DisableCompress 为 true 时不对轮转日志做 gzip 压缩（默认 false：压缩开启）。
	DisableCompress bool `json:"disableCompress" yaml:"disableCompress"`

	// DisableConsole 为 true 时不输出到 stderr（默认 false：控制台输出开启）。
	DisableConsole bool `json:"disableConsole" yaml:"disableConsole"`

	// DisableCaller 为 true 时日志不携带调用方信息（默认 false：携带）。
	DisableCaller bool `json:"disableCaller" yaml:"disableCaller"`

	// EnableSourceIP determines if the log should contain the sourceIP
	EnableSourceIP bool `json:"enableSourceIP" yaml:"enableSourceIP"`

	// DisablePID 为 true 时日志不携带进程 PID（默认 false：携带）。
	DisablePID bool `json:"disablePID" yaml:"disablePID"`

	// log level in log file
	FileLevel string `json:"fileLevel" yaml:"fileLevel"`

	// log level in console
	ConsoleLevel string `json:"consoleLevel" yaml:"consoleLevel"`

	// encoding in log file. Valid values are "json" and
	// "console"
	FileEncoding string `json:"fileEncoding" yaml:"fileEncoding"`

	// encoding in console. Valid values are "json" and
	// "console"
	ConsoleEncoding string `json:"consoleEncoding" yaml:"consoleEncoding"`

	// application name
	// default is app
	AppName string `json:"appName" yaml:"appName"`

	// SourceEth determine which eth to get SourceIp
	// defautl is en0
	SourceEth string `json:"sourceEth" yaml:"sourceEth"`

	// DisableTraceID disable trace id
	DisableTraceID bool `json:"disableTraceID" yaml:"disableTraceID"`
}
