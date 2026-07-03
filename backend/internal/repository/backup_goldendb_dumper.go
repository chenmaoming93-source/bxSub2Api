package repository

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// GoldenDBDumper implements service.DBDumper using MySQL-compatible tools.
type GoldenDBDumper struct {
	cfg *config.DatabaseConfig
}

func NewGoldenDBDumper(cfg *config.Config) service.DBDumper {
	return &GoldenDBDumper{cfg: &cfg.Database}
}

func NewMySQLDumper(cfg *config.Config) service.DBDumper {
	return NewGoldenDBDumper(cfg)
}

func (d *GoldenDBDumper) Dump(ctx context.Context) (io.ReadCloser, error) {
	args := []string{
		"-h", d.cfg.Host,
		"-P", fmt.Sprintf("%d", d.cfg.Port),
		"-u", d.cfg.User,
		"--single-transaction",
		"--routines",
		"--triggers",
		"--set-gtid-purged=OFF",
		d.cfg.DBName,
	}
	if d.cfg.Password != "" {
		args = append([]string{fmt.Sprintf("-p%s", d.cfg.Password)}, args...)
	}

	cmd := exec.CommandContext(ctx, "mysqldump", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start mysqldump: %w", err)
	}
	return &cmdReadCloser{ReadCloser: stdout, cmd: cmd, name: "mysqldump"}, nil
}

func (d *GoldenDBDumper) Restore(ctx context.Context, data io.Reader) error {
	args := []string{
		"-h", d.cfg.Host,
		"-P", fmt.Sprintf("%d", d.cfg.Port),
		"-u", d.cfg.User,
		d.cfg.DBName,
	}
	if d.cfg.Password != "" {
		args = append([]string{fmt.Sprintf("-p%s", d.cfg.Password)}, args...)
	}

	cmd := exec.CommandContext(ctx, "mysql", args...)
	cmd.Stdin = data
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}
	return nil
}

type cmdReadCloser struct {
	io.ReadCloser
	cmd  *exec.Cmd
	name string
}

func (c *cmdReadCloser) Close() error {
	_ = c.ReadCloser.Close()
	if err := c.cmd.Wait(); err != nil {
		return fmt.Errorf("%s exited with error: %w", c.name, err)
	}
	return nil
}
