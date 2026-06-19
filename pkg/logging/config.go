// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"os"
	"path/filepath"
)

var AppName = filepath.Base(os.Args[0])

// old init() 中的 SetDefault 已集中迁移到 internal/config/manager.go setDefaultsOn()。
func init() {
}

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

	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time.  The default is to use UTC
	// time.
	LocalTime bool `json:"localtime" yaml:"localtime"`

	// Compress determines if the rotated log files should be compressed
	// using gzip.
	Compress bool `json:"compress" yaml:"compress"`

	// EnableConsole determines if the log should be displayed in stderr.
	EnableConsole bool `json:"enableConsole" yaml:"enableConsole"`

	// EnableCaller determines if the log should contain the caller
	EnableCaller bool `json:"enableCaller" yaml:"enableCaller"`

	// EnableSourceIP determines if the log should contain the sourceIP
	EnableSourceIP bool `json:"enableSourceIP" yaml:"enableSourceIP"`

	// EnablePID determines if the log should contain the PID
	EnablePID bool `json:"enablePID" yaml:"enablePID"`

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

	// GlobalCallerSkip increases the number of callers skipped
	GlobalCallerSkip int `json:"-" yaml:"-"`
}

func NewDevelopmentConfig(appname string, filename string) Config {
	return Config{
		EnableFile:      true,
		Filename:        filename,
		MaxSize:         0,
		MaxAge:          0,
		MaxBackups:      0,
		LocalTime:       true,
		Compress:        false,
		EnableConsole:   true,
		EnableCaller:    true,
		EnableSourceIP:  true,
		EnablePID:       true,
		FileLevel:       "debug",
		ConsoleLevel:    "debug",
		FileEncoding:    "json",
		ConsoleEncoding: "console",
		AppName:         appname,
		SourceEth:       "en0",
		DisableTraceID:  false,
	}
}

func NewProductionConfig(appname string, filename string) Config {
	return Config{
		EnableFile:      true,
		Filename:        filename,
		MaxSize:         0,
		MaxAge:          0,
		MaxBackups:      0,
		LocalTime:       true,
		Compress:        false,
		EnableConsole:   false,
		EnableCaller:    true,
		EnableSourceIP:  true,
		EnablePID:       true,
		FileLevel:       "info",
		ConsoleLevel:    "info",
		FileEncoding:    "json",
		ConsoleEncoding: "console",
		AppName:         appname,
		SourceEth:       "en0",
		DisableTraceID:  false,
	}
}

func NewStdConfig() Config {
	return Config{
		EnableFile:      false,
		Filename:        "",
		MaxSize:         0,
		MaxAge:          0,
		MaxBackups:      0,
		LocalTime:       true,
		Compress:        false,
		EnableConsole:   true,
		EnableCaller:    true,
		EnableSourceIP:  false,
		EnablePID:       true,
		FileLevel:       "info",
		ConsoleLevel:    "debug",
		FileEncoding:    "json",
		ConsoleEncoding: "console",
		AppName:         "",
		SourceEth:       "en0",
		DisableTraceID:  false,
	}
}
