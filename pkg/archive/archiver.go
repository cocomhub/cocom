package archive

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func New(t Type) Algorithm {
	switch t {
	case TypeDouble:
		return &double{}
	default:
		return &single{}
	}
}

type single struct{}

func (s *single) Type() Type { return TypeSingle }

func (s *single) Archive(ctx context.Context, srcDir string, destArchivePath string, cfg Config, cid int) error {
	args := []string{"a", "-mhe=on", "-p" + cfg.Password}
	args = append(args, destArchivePath, srcDir)
	cmd := exec.CommandContext(ctx, cfg.CmdPath, args...)
	return cmd.Run()
}

func (s *single) Restore(ctx context.Context, archivePath string, destDir string, cfg Config, cid int) error {
	args := []string{"x", "-y", "-p" + cfg.Password, "-o" + destDir, archivePath}
	cmd := exec.CommandContext(ctx, cfg.CmdPath, args...)
	return cmd.Run()
}

type double struct{}

func (d *double) Type() Type { return TypeDouble }

func (d *double) Archive(ctx context.Context, srcDir string, destArchivePath string, cfg Config, cid int) error {
	stage := destArchivePath + ".stage1"
	if err := (&single{}).Archive(ctx, srcDir, stage, cfg, cid); err != nil {
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
	if err := (&single{}).Restore(ctx, nestedFile, destDir, cfg, cid); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	return os.RemoveAll(tmpDir)
}
