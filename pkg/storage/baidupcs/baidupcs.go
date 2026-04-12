// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package baidupcs

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/cocomhub/cocom/pkg/storage"
)

const Type = "baidupcs"

type Config struct {
	Command string
	Root    string
	TempDir string
	WorkDir string
	Timeout time.Duration
	Args    []string
}

type Storage struct {
	name   string
	root   string
	config Config
	runner runner
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
	if config.Command == "" {
		return nil, fmt.Errorf("%w: command is empty", storage.ErrInvalidParam)
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
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	return &Storage{
		name:   name,
		root:   root,
		config: config,
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
	args := []string{"upload"}
	if po.Overwrite {
		args = append(args, "--overwrite")
	}
	args = append(args, tmpPath, remote)
	result, err := s.runner.Run(ctx, "put", args...)
	if err != nil {
		return storage.ObjectMeta{}, s.commandError("put", result, err)
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
	result, err := s.runner.Run(ctx, "get", "download", remote, tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return nil, storage.ObjectMeta{}, s.commandError("get", result, err)
	}
	fd, err := os.Open(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return nil, storage.ObjectMeta{}, err
	}
	return &tempReadCloser{ReadCloser: fd, path: tmpPath}, meta, nil
}

func (s *Storage) Stat(ctx context.Context, key string) (storage.ObjectMeta, error) {
	remote, err := s.remotePath(key)
	if err != nil {
		return storage.ObjectMeta{}, err
	}
	result, err := s.runner.Run(ctx, "stat", "meta", remote)
	if err != nil {
		return storage.ObjectMeta{}, s.commandError("stat", result, err)
	}
	entry, err := parseSingleEntry(result.Stdout)
	if err != nil {
		return storage.ObjectMeta{}, fmt.Errorf("baidupcs backend %q stat parse output: %w", s.name, err)
	}
	meta, err := s.metaFromEntry(entry)
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
	remote, err := s.remotePath(prefix)
	if err != nil {
		return nil, err
	}
	result, err := s.runner.Run(ctx, "list", "ls", remote)
	if err != nil {
		return nil, s.commandError("list", result, err)
	}
	entries, err := parseEntries(result.Stdout)
	if err != nil {
		return nil, fmt.Errorf("baidupcs backend %q list parse output: %w", s.name, err)
	}
	out := make([]storage.ObjectMeta, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir {
			continue
		}
		meta, err := s.metaFromEntry(entry)
		if err != nil {
			return nil, err
		}
		out = append(out, meta)
	}
	return out, nil
}

func (s *Storage) Delete(ctx context.Context, key string) error {
	remote, err := s.remotePath(key)
	if err != nil {
		return err
	}
	result, err := s.runner.Run(ctx, "delete", "rm", remote)
	if err != nil {
		return s.commandError("delete", result, err)
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
	args := []string{"cp"}
	if po.Overwrite {
		args = append(args, "--overwrite")
	}
	args = append(args, src, dst)
	result, err := s.runner.Run(ctx, "copy", args...)
	if err != nil {
		return storage.ObjectMeta{}, s.commandError("copy", result, err)
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
	args := []string{"mv"}
	if po.Overwrite {
		args = append(args, "--overwrite")
	}
	args = append(args, src, dst)
	result, err := s.runner.Run(ctx, "move", args...)
	if err != nil {
		return storage.ObjectMeta{}, s.commandError("move", result, err)
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

func (s *Storage) metaFromEntry(entry remoteEntry) (storage.ObjectMeta, error) {
	key, err := s.logicalKey(entry.Path)
	if err != nil {
		return storage.ObjectMeta{}, err
	}
	return storage.ObjectMeta{
		Key:     key,
		Size:    entry.Size,
		ETag:    entry.ETag,
		ModTime: entry.ModTime,
	}, nil
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

func (s *Storage) commandError(op string, result commandResult, err error) error {
	diagnostic := summarizeDiagnostic(result, err)
	lower := strings.ToLower(diagnostic)
	switch {
	case errors.Is(err, context.DeadlineExceeded), strings.Contains(lower, "deadline exceeded"), strings.Contains(lower, "timeout"), strings.Contains(lower, "timed out"), strings.Contains(lower, "超时"):
		return fmt.Errorf("baidupcs backend %q %s timeout: %s: %w", s.name, op, diagnostic, storage.ErrTransient)
	case strings.Contains(lower, "not found"), strings.Contains(lower, "no such file"), strings.Contains(lower, "does not exist"), strings.Contains(lower, "文件不存在"), strings.Contains(lower, "找不到文件"):
		return fmt.Errorf("baidupcs backend %q %s failed: %s: %w", s.name, op, diagnostic, storage.ErrNotFound)
	case strings.Contains(lower, "already exists"), strings.Contains(lower, "file exists"), strings.Contains(lower, "文件已存在"):
		return fmt.Errorf("baidupcs backend %q %s failed: %s: %w", s.name, op, diagnostic, storage.ErrAlreadyExists)
	case strings.Contains(lower, "permission denied"), strings.Contains(lower, "权限"), strings.Contains(lower, "access denied"):
		return fmt.Errorf("baidupcs backend %q %s failed: %s: %w", s.name, op, diagnostic, storage.ErrPermissionDenied)
	}
	return fmt.Errorf("baidupcs backend %q %s failed: %s: %w", s.name, op, diagnostic, err)
}

func (r commandRunner) Run(ctx context.Context, op string, args ...string) (commandResult, error) {
	runCtx := ctx
	cancel := func() {}
	if r.timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, r.timeout)
	}
	defer cancel()

	cmdArgs := append(append([]string(nil), r.args...), args...)
	cmd := exec.CommandContext(runCtx, r.command, cmdArgs...)
	if r.workDir != "" {
		cmd.Dir = r.workDir
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := commandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	if err != nil {
		if result.ExitCode == 0 {
			result.ExitCode = -1
		}
		if runCtx.Err() != nil {
			return result, runCtx.Err()
		}
		return result, err
	}
	return result, nil
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
