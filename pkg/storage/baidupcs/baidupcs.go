// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package baidupcs

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	bdlib "github.com/qjfoidnh/BaiduPCS-Go/baidupcs"
	pcserr "github.com/qjfoidnh/BaiduPCS-Go/baidupcs/pcserror"

	"github.com/cocomhub/cocom/pkg/storage"
)

const Type = "baidupcs"

type Config struct {
	Root         string
	TempDir      string
	BDUSS        string
	Cookies      string
	SToken       string
	SBoxTKN      string
	AppID        int
	PCSAddr      string
	PCSUserAgent string
	PanUserAgent string
	Adapter      Adapter
	UID          uint64

	// Deprecated legacy command fields kept to avoid breaking in-package tests
	// and older programmatic call sites during migration.
	Command string
	WorkDir string
	Timeout time.Duration
	Args    []string
}

type Storage struct {
	name    string
	root    string
	config  Config
	adapter Adapter
	runner  runner
}

type runner interface {
	Run(ctx context.Context, op string, args ...string) (commandResult, error)
}

type commandRunner struct {
	command string
	workDir string
	timeout time.Duration
	args    []string
}

type commandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type remoteEntry struct {
	Path    string
	Size    int64
	ModTime time.Time
	ETag    string
	IsDir   bool
}

func New(name string, config Config) (*Storage, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: storage name is empty", storage.ErrInvalidParam)
	}
	if config.Root == "" {
		return nil, fmt.Errorf("%w: root is empty", storage.ErrInvalidParam)
	}
	root, err := storage.Path(config.Root)
	if err != nil {
		return nil, fmt.Errorf("%w: root %q: %v", storage.ErrInvalidParam, config.Root, err)
	}
	if config.TempDir == "" {
		config.TempDir = os.TempDir()
	}
	if err := os.MkdirAll(config.TempDir, 0o755); err != nil {
		return nil, err
	}
	config.Root = root

	adapter := config.Adapter
	if adapter == nil {
		if strings.TrimSpace(config.BDUSS) == "" && strings.TrimSpace(config.Cookies) == "" {
			return nil, fmt.Errorf("%w: either bduss or cookies is required", storage.ErrInvalidParam)
		}
		adapter, err = newLibraryAdapter(config)
		if err != nil {
			return nil, fmt.Errorf("create baidupcs client: %w", err)
		}
	}

	return &Storage{
		name:    name,
		root:    root,
		config:  config,
		adapter: adapter,
		runner: commandRunner{
			command: config.Command,
			workDir: config.WorkDir,
			timeout: config.Timeout,
			args:    append([]string(nil), config.Args...),
		},
	}, nil
}

func (s *Storage) Type() string {
	return Type
}

func (s *Storage) Name() string {
	return s.name
}

func (s *Storage) Put(ctx context.Context, key string, r io.Reader, opts ...storage.Option) (storage.ObjectMeta, error) {
	var po storage.PutOptions
	for _, opt := range opts {
		opt(&po)
	}
	if !po.Overwrite {
		exists, err := s.Exists(ctx, key)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return storage.ObjectMeta{}, err
		}
		if exists {
			return storage.ObjectMeta{}, storage.ErrAlreadyExists
		}
	}

	tmp, err := os.CreateTemp(s.config.TempDir, "cocom-baidupcs-put-*")
	if err != nil {
		return storage.ObjectMeta{}, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	localMeta, err := writeTempFile(tmp, key, r, po.Hash)
	if err != nil {
		return storage.ObjectMeta{}, err
	}
	remote, err := s.remotePath(key)
	if err != nil {
		return storage.ObjectMeta{}, err
	}
	if err := s.adapter.Upload(ctx, tmpPath, remote, po.Overwrite); err != nil {
		return storage.ObjectMeta{}, s.mapError("put", err)
	}

	meta, statErr := s.Stat(ctx, key)
	if statErr == nil {
		if meta.ETag == "" {
			meta.ETag = localMeta.ETag
		}
		return meta, nil
	}
	return localMeta, nil
}

func (s *Storage) Get(ctx context.Context, key string) (io.ReadCloser, storage.ObjectMeta, error) {
	meta, err := s.Stat(ctx, key)
	if err != nil {
		return nil, storage.ObjectMeta{}, err
	}

	tmp, err := os.CreateTemp(s.config.TempDir, "cocom-baidupcs-get-*")
	if err != nil {
		return nil, storage.ObjectMeta{}, err
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return nil, storage.ObjectMeta{}, err
	}

	remote, err := s.remotePath(key)
	if err != nil {
		_ = os.Remove(tmpPath)
		return nil, storage.ObjectMeta{}, err
	}
	if err := s.adapter.Download(ctx, remote, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return nil, storage.ObjectMeta{}, s.mapError("get", err)
	}

	fd, err := os.Open(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return nil, storage.ObjectMeta{}, err
	}
	return &tempReadCloser{ReadCloser: fd, path: tmpPath}, meta, nil
}

func (s *Storage) Stat(ctx context.Context, key string) (storage.ObjectMeta, error) {
	if err := ctx.Err(); err != nil {
		return storage.ObjectMeta{}, s.mapError("stat", err)
	}
	remote, err := s.remotePath(key)
	if err != nil {
		return storage.ObjectMeta{}, err
	}
	fd, err := s.adapter.Meta(remote)
	if err != nil {
		return storage.ObjectMeta{}, s.mapError("stat", err)
	}
	meta, err := s.metaFromFileDirectory(fd)
	if err != nil {
		return storage.ObjectMeta{}, err
	}
	return meta, nil
}

func (s *Storage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.Stat(ctx, key)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, storage.ErrNotFound) {
		return false, nil
	}
	return false, err
}

func (s *Storage) List(ctx context.Context, prefix string) ([]storage.ObjectMeta, error) {
	if err := ctx.Err(); err != nil {
		return nil, s.mapError("list", err)
	}
	remote, err := s.remotePath(prefix)
	if err != nil {
		return nil, err
	}
	fd, err := s.adapter.Meta(remote)
	if err != nil {
		return nil, s.mapError("list", err)
	}
	if !fd.Isdir {
		meta, err := s.metaFromFileDirectory(fd)
		if err != nil {
			return nil, err
		}
		return []storage.ObjectMeta{meta}, nil
	}

	stack := []string{fd.Path}
	out := make([]storage.ObjectMeta, 0)
	for len(stack) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, s.mapError("list", err)
		}
		dir := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		children, err := s.adapter.List(dir)
		if err != nil {
			return nil, s.mapError("list", err)
		}
		for _, child := range children {
			if child == nil {
				continue
			}
			if child.Isdir {
				stack = append(stack, child.Path)
				continue
			}
			meta, err := s.metaFromFileDirectory(child)
			if err != nil {
				return nil, err
			}
			out = append(out, meta)
		}
	}
	return out, nil
}

func (s *Storage) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return s.mapError("delete", err)
	}
	remote, err := s.remotePath(key)
	if err != nil {
		return err
	}
	if err := s.adapter.Delete(remote); err != nil {
		return s.mapError("delete", err)
	}
	return nil
}

func (s *Storage) Copy(ctx context.Context, srcKey, dstKey string, opts ...storage.Option) (storage.ObjectMeta, error) {
	var po storage.PutOptions
	for _, opt := range opts {
		opt(&po)
	}
	if !po.Overwrite {
		exists, err := s.Exists(ctx, dstKey)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return storage.ObjectMeta{}, err
		}
		if exists {
			return storage.ObjectMeta{}, storage.ErrAlreadyExists
		}
	}
	src, err := s.remotePath(srcKey)
	if err != nil {
		return storage.ObjectMeta{}, err
	}
	dst, err := s.remotePath(dstKey)
	if err != nil {
		return storage.ObjectMeta{}, err
	}
	if err := s.adapter.Copy(&bdlib.CpMvJSON{From: src, To: dst}); err != nil {
		return storage.ObjectMeta{}, s.mapError("copy", err)
	}
	return s.Stat(ctx, dstKey)
}

func (s *Storage) Move(ctx context.Context, srcKey, dstKey string, opts ...storage.Option) (storage.ObjectMeta, error) {
	var po storage.PutOptions
	for _, opt := range opts {
		opt(&po)
	}
	if !po.Overwrite {
		exists, err := s.Exists(ctx, dstKey)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return storage.ObjectMeta{}, err
		}
		if exists {
			return storage.ObjectMeta{}, storage.ErrAlreadyExists
		}
	}
	src, err := s.remotePath(srcKey)
	if err != nil {
		return storage.ObjectMeta{}, err
	}
	dst, err := s.remotePath(dstKey)
	if err != nil {
		return storage.ObjectMeta{}, err
	}
	if err := s.adapter.Move(&bdlib.CpMvJSON{From: src, To: dst}); err != nil {
		return storage.ObjectMeta{}, s.mapError("move", err)
	}
	return s.Stat(ctx, dstKey)
}

func (s *Storage) remotePath(key string) (string, error) {
	p, err := storage.Path(key)
	if err != nil {
		return "", fmt.Errorf("%w: key %q: %v", storage.ErrInvalidParam, key, err)
	}
	if p == "/" {
		return s.root, nil
	}
	if s.root == "/" {
		return p, nil
	}
	return storage.Path(s.root, strings.TrimPrefix(p, "/"))
}

func (s *Storage) metaFromFileDirectory(fd *bdlib.FileDirectory) (storage.ObjectMeta, error) {
	if fd == nil {
		return storage.ObjectMeta{}, fmt.Errorf("%w: file metadata is nil", storage.ErrInvalidParam)
	}
	key, err := s.logicalKey(fd.Path)
	if err != nil {
		return storage.ObjectMeta{}, err
	}
	meta := storage.ObjectMeta{
		Key:  key,
		Size: fd.Size,
		ETag: fd.MD5,
	}
	if fd.Mtime > 0 {
		meta.ModTime = time.Unix(fd.Mtime, 0).UTC()
	}
	return meta, nil
}

func (s *Storage) logicalKey(remote string) (string, error) {
	normalized, err := storage.Path(remote)
	if err != nil {
		return "", fmt.Errorf("%w: remote path %q: %v", storage.ErrInvalidParam, remote, err)
	}
	if s.root == "/" {
		return normalized, nil
	}
	if normalized == s.root {
		return "/", nil
	}
	prefix := s.root + "/"
	if !strings.HasPrefix(normalized, prefix) {
		return "", fmt.Errorf("%w: remote path %q is outside root %q", storage.ErrInvalidParam, remote, s.root)
	}
	return storage.Path(strings.TrimPrefix(normalized, prefix))
}

func (s *Storage) mapError(op string, err error) error {
	if err == nil {
		return nil
	}
	diagnostic := trimSummary(err.Error())

	switch {
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("baidupcs backend %q %s canceled: %s: %w", s.name, op, diagnostic, storage.ErrTransient)
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("baidupcs backend %q %s timeout: %s: %w", s.name, op, diagnostic, storage.ErrTransient)
	}

	var pcsErr pcserr.Error
	if errors.As(err, &pcsErr) {
		if mapped := mapPCSErrorCategory(pcsErr); mapped != nil {
			return fmt.Errorf("baidupcs backend %q %s failed: %s: %w", s.name, op, diagnostic, mapped)
		}
	}

	if mapped := mapMessageCategory(diagnostic); mapped != nil {
		return fmt.Errorf("baidupcs backend %q %s failed: %s: %w", s.name, op, diagnostic, mapped)
	}
	return fmt.Errorf("baidupcs backend %q %s failed: %s: %w", s.name, op, diagnostic, err)
}

func mapPCSErrorCategory(err pcserr.Error) error {
	switch err.GetErrType() {
	case pcserr.ErrTypeNetError:
		return storage.ErrTransient
	case pcserr.ErrTypeRemoteError:
		switch err.GetRemoteErrCode() {
		case 31066, -3, -9:
			return storage.ErrNotFound
		case 31061, -8, -30, 114514, 1919810:
			return storage.ErrAlreadyExists
		case 3, -4, -6, -11, 31045, 9019:
			return storage.ErrPermissionDenied
		case 2, 4, 112, 113:
			return storage.ErrTransient
		}
		if mapped := mapMessageCategory(err.GetRemoteErrMsg()); mapped != nil {
			return mapped
		}
	default:
		if mapped := mapMessageCategory(err.Error()); mapped != nil {
			return mapped
		}
	}
	if raw := err.GetError(); raw != nil {
		if errors.Is(raw, context.Canceled) || errors.Is(raw, context.DeadlineExceeded) {
			return storage.ErrTransient
		}
	}
	return nil
}

func mapMessageCategory(message string) error {
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "deadline exceeded"),
		strings.Contains(lower, "timeout"),
		strings.Contains(lower, "timed out"),
		strings.Contains(lower, "超时"),
		strings.Contains(lower, "请稍后再试"):
		return storage.ErrTransient
	case strings.Contains(lower, "not found"),
		strings.Contains(lower, "no such file"),
		strings.Contains(lower, "does not exist"),
		strings.Contains(lower, "文件不存在"),
		strings.Contains(lower, "目录不存在"):
		return storage.ErrNotFound
	case strings.Contains(lower, "already exists"),
		strings.Contains(lower, "file exists"),
		strings.Contains(lower, "文件已存在"),
		strings.Contains(lower, "同名文件"):
		return storage.ErrAlreadyExists
	case strings.Contains(lower, "permission denied"),
		strings.Contains(lower, "access denied"),
		strings.Contains(lower, "未登录"),
		strings.Contains(lower, "cookie"),
		strings.Contains(lower, "权限"),
		strings.Contains(lower, "登录"):
		return storage.ErrPermissionDenied
	}
	return nil
}

func (s *Storage) commandError(op string, result commandResult, err error) error {
	if err == nil {
		return nil
	}
	diagnostic := summarizeDiagnostic(result, err)
	if mapped := mapMessageCategory(diagnostic); mapped != nil {
		return fmt.Errorf("baidupcs backend %q %s failed: %s: %w", s.name, op, diagnostic, mapped)
	}
	return fmt.Errorf("baidupcs backend %q %s failed: %s: %w", s.name, op, diagnostic, err)
}

func (r commandRunner) Run(ctx context.Context, op string, args ...string) (commandResult, error) {
	return commandResult{}, errors.New("command runner is deprecated for baidupcs storage")
}

func parseSingleEntry(output string) (remoteEntry, error) {
	entries, err := parseEntries(output)
	if err != nil {
		return remoteEntry{}, err
	}
	if len(entries) == 0 {
		return remoteEntry{}, io.EOF
	}
	return entries[0], nil
}

func parseEntries(output string) ([]remoteEntry, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil, nil
	}
	switch output[0] {
	case '{':
		var payload struct {
			Entries []remoteEntryPayload `json:"entries"`
		}
		if err := json.Unmarshal([]byte(output), &payload); err == nil && len(payload.Entries) > 0 {
			return payloadEntries(payload.Entries)
		}
		var single remoteEntryPayload
		if err := json.Unmarshal([]byte(output), &single); err == nil && single.Path() != "" {
			entry, err := single.Entry()
			if err != nil {
				return nil, err
			}
			return []remoteEntry{entry}, nil
		}
	case '[':
		var payload []remoteEntryPayload
		if err := json.Unmarshal([]byte(output), &payload); err == nil {
			return payloadEntries(payload)
		}
	}
	lines := strings.Split(output, "\n")
	entries := make([]remoteEntry, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entry, err := parseLineEntry(line)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

type remoteEntryPayload struct {
	RemotePath string `json:"path"`
	Key        string `json:"key"`
	Name       string `json:"name"`
	SizeValue  int64  `json:"size"`
	ModValue   any    `json:"mod_time"`
	ModAlt     any    `json:"modTime"`
	MTime      any    `json:"mtime"`
	Dir        bool   `json:"is_dir"`
	DirAlt     bool   `json:"isDir"`
	ETag       string `json:"etag"`
	MD5        string `json:"md5"`
}

func (p remoteEntryPayload) Path() string {
	switch {
	case p.RemotePath != "":
		return p.RemotePath
	case p.Key != "":
		return p.Key
	default:
		return p.Name
	}
}

func (p remoteEntryPayload) Entry() (remoteEntry, error) {
	modTime, err := parseTimeValue(p.ModValue, p.ModAlt, p.MTime)
	if err != nil {
		return remoteEntry{}, err
	}
	etag := p.ETag
	if etag == "" {
		etag = p.MD5
	}
	return remoteEntry{
		Path:    p.Path(),
		Size:    p.SizeValue,
		ModTime: modTime,
		ETag:    etag,
		IsDir:   p.Dir || p.DirAlt,
	}, nil
}

func payloadEntries(payload []remoteEntryPayload) ([]remoteEntry, error) {
	entries := make([]remoteEntry, 0, len(payload))
	for _, item := range payload {
		entry, err := item.Entry()
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func parseLineEntry(line string) (remoteEntry, error) {
	fields := strings.Split(line, "\t")
	if len(fields) < 4 {
		return remoteEntry{}, fmt.Errorf("unsupported baidupcs output line %q", line)
	}
	size, err := strconv.ParseInt(strings.TrimSpace(fields[2]), 10, 64)
	if err != nil {
		return remoteEntry{}, err
	}
	modTime, err := parseTimeString(fields[3])
	if err != nil {
		return remoteEntry{}, err
	}
	etag := ""
	if len(fields) >= 5 {
		etag = strings.TrimSpace(fields[4])
	}
	return remoteEntry{
		IsDir:   strings.EqualFold(strings.TrimSpace(fields[0]), "D"),
		Path:    strings.TrimSpace(fields[1]),
		Size:    size,
		ModTime: modTime,
		ETag:    etag,
	}, nil
}

func parseTimeValue(values ...any) (time.Time, error) {
	for _, value := range values {
		switch v := value.(type) {
		case nil:
			continue
		case string:
			if strings.TrimSpace(v) == "" {
				continue
			}
			return parseTimeString(v)
		case float64:
			return time.Unix(int64(v), 0).UTC(), nil
		}
	}
	return time.Time{}, nil
}

func parseTimeString(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	if unix, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(unix, 0).UTC(), nil
	}
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"}
	for _, layout := range layouts {
		if tm, err := time.Parse(layout, value); err == nil {
			return tm.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time value %q", value)
}

func summarizeDiagnostic(result commandResult, err error) string {
	parts := make([]string, 0, 4)
	if result.ExitCode != 0 {
		parts = append(parts, fmt.Sprintf("exit=%d", result.ExitCode))
	}
	if text := strings.TrimSpace(result.Stderr); text != "" {
		parts = append(parts, "stderr="+trimSummary(text))
	}
	if text := strings.TrimSpace(result.Stdout); text != "" {
		parts = append(parts, "stdout="+trimSummary(text))
	}
	if err != nil {
		parts = append(parts, "err="+trimSummary(err.Error()))
	}
	if len(parts) == 0 {
		return "no diagnostic output"
	}
	return strings.Join(parts, " ")
}

func trimSummary(text string) string {
	text = strings.ReplaceAll(text, "\n", " | ")
	if len(text) > 240 {
		return text[:240] + "..."
	}
	return text
}

func writeTempFile(tmp *os.File, key string, r io.Reader, h hash.Hash) (storage.ObjectMeta, error) {
	var (
		size int64
		err  error
	)
	if h != nil {
		size, err = io.Copy(tmp, io.TeeReader(r, h))
	} else {
		size, err = io.Copy(tmp, r)
	}
	if err != nil {
		_ = tmp.Close()
		return storage.ObjectMeta{}, err
	}
	if err := tmp.Close(); err != nil {
		return storage.ObjectMeta{}, err
	}
	meta := storage.ObjectMeta{
		Key:     storage.MustPath(key),
		Size:    size,
		ModTime: time.Now(),
	}
	if h != nil {
		meta.ETag = hex.EncodeToString(h.Sum(nil))
	}
	return meta, nil
}

type tempReadCloser struct {
	io.ReadCloser
	path string
}

func (r *tempReadCloser) Close() error {
	err := r.ReadCloser.Close()
	removeErr := os.Remove(r.path)
	if err != nil {
		return err
	}
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return removeErr
	}
	return nil
}
