package archive

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/spf13/viper"
)

type Type string

const (
	TypeSingle Type = "single"
	TypeDouble Type = "double"
)

type Config struct {
	CmdPath  string
	Password string
}

type Algorithm interface {
	Type() Type
	Archive(ctx context.Context, srcDir string, destArchivePath string, cfg Config, cid int) error
	Restore(ctx context.Context, archivePath string, destDir string, cfg Config, cid int) error
}

var (
	onceSingle = sync.OnceValue(newSingle)
	onceDouble = sync.OnceValue(newDouble)
)

func Get(t Type) Algorithm {
	switch t {
	case TypeDouble:
		return onceDouble()
	default:
		return onceSingle()
	}
}

func newSingle() *single {
	return &single{ch: make(chan struct{}, viper.GetInt("archive.algorithm.single.concurrency"))}
}

type single struct {
	ch chan struct{}
}

func (s *single) Type() Type { return TypeSingle }

func (s *single) Archive(ctx context.Context, srcDir string, destArchivePath string, cfg Config, cid int) error {
	s.ch <- struct{}{}
	defer func() { <-s.ch }()

	args := []string{"a", "-mhe=on", "-p" + cfg.Password}
	args = append(args, destArchivePath, srcDir)
	cmd := exec.CommandContext(ctx, cfg.CmdPath, args...)
	return cmd.Run()
}

func (s *single) Restore(ctx context.Context, archivePath string, destDir string, cfg Config, cid int) error {
	s.ch <- struct{}{}
	defer func() { <-s.ch }()

	args := []string{"x", "-y", "-p" + cfg.Password, "-o" + destDir, archivePath}
	cmd := exec.CommandContext(ctx, cfg.CmdPath, args...)
	return cmd.Run()
}

func newDouble() *double {
	return &double{
		ch:     make(chan struct{}, viper.GetInt("archive.algorithm.double.concurrency")),
		single: onceSingle(),
	}
}

type double struct {
	ch     chan struct{}
	single *single
}

func (d *double) Type() Type { return TypeDouble }

func (d *double) Archive(ctx context.Context, srcDir string, destArchivePath string, cfg Config, cid int) error {
	d.ch <- struct{}{}
	defer func() { <-d.ch }()

	stage := destArchivePath + ".stage1"
	if err := d.single.Archive(ctx, srcDir, stage, cfg, cid); err != nil {
		return err
	}
	nestedDir := filepath.Join(filepath.Dir(destArchivePath), fmt.Sprintf("%d", cid))
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		return err
	}
	nestedFile := filepath.Join(nestedDir, filepath.Base(destArchivePath))
	if err := os.Rename(stage, nestedFile); err != nil {
		return err
	}
	args := []string{"a", "-mhe=on", "-p" + cfg.Password}
	args = append(args, destArchivePath, nestedDir)
	cmd := exec.CommandContext(ctx, cfg.CmdPath, args...)
	if err := cmd.Run(); err != nil {
		return err
	}
	return os.RemoveAll(nestedDir)
}

func (d *double) Restore(ctx context.Context, archivePath string, destDir string, cfg Config, cid int) error {
	d.ch <- struct{}{}
	defer func() { <-d.ch }()

	tmpDir, err := os.MkdirTemp(filepath.Dir(archivePath), "restore-*")
	if err != nil {
		return err
	}
	args := []string{"x", "-y", "-p" + cfg.Password, "-o" + tmpDir, archivePath}
	cmd := exec.CommandContext(ctx, cfg.CmdPath, args...)
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	nestedFile := filepath.Join(tmpDir, fmt.Sprintf("%d", cid), filepath.Base(archivePath))
	if err := d.single.Restore(ctx, nestedFile, destDir, cfg, cid); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	return os.RemoveAll(tmpDir)
}
