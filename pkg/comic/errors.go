package comic

import "github.com/suixibing/cocom/pkg/errwrap"

var (
	// ErrComicNotFound 漫画不存在
	ErrComicNotFound = errwrap.New(3000, "comic not found")

	// ErrComicInvalid 漫画无效
	ErrComicInvalid = errwrap.New(3001, "invalid comic")

	// ErrComicVerify 漫画验证失败
	ErrComicVerify = errwrap.New(3002, "comic verify failed")

	// ErrComicDownload 漫画下载失败
	ErrComicDownload = errwrap.New(3003, "comic download failed")

	// ErrComicDB 数据库操作失败
	ErrComicDB = errwrap.New(3004, "comic database operation failed")

	// ErrComicPattern 无效的匹配模式
	ErrComicPattern = errwrap.New(3005, "invalid comic pattern")

	// ErrTaskNotFound 任务不存在
	ErrTaskNotFound = errwrap.New(3006, "task not found")

	// ErrComicDB      = errwrap.New(3000, "comic database error")
	// ErrComicVerify  = errwrap.New(3001, "comic verify error")
	// ErrComicDownload = errwrap.New(3002, "comic download error")
	// ErrComicState   = errwrap.New(3003, "comic state error")
	// ErrComicMetrics = errwrap.New(3004, "comic metrics error")
	// ErrComicReport  = errwrap.New(3005, "comic report error")
)
