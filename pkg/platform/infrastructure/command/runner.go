package command

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os/exec"

	applogger "github.com/tss-calculator/go-lib/pkg/application/logger"
)

type Command struct {
	WorkDir    string
	Executable string
	Args       []string
	Verbose    bool
}

type Runner interface {
	Execute(ctx context.Context, command Command) (string, error)
}

func NewCommandRunner(logger applogger.Logger, silent bool) Runner {
	return &runner{
		logger: logger,
		silent: silent,
	}
}

type runner struct {
	logger applogger.Logger
	silent bool
}

func (r runner) Execute(ctx context.Context, command Command) (string, error) {
	if command.Executable == "" {
		return "", errors.New("command executable can not be empty")
	}
	// nolint:gosec
	cmd := exec.CommandContext(ctx, command.Executable, command.Args...)
	cmd.Dir = command.WorkDir
	r.logger.Debug(cmd.String())
	if command.Verbose && !r.silent {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return "", err
		}
		go r.verboseLogger(stdout)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return "", err
		}
		go r.verboseLogger(stderr)
		return "", cmd.Run()
	}
	result, err := cmd.CombinedOutput()
	return string(result), err
}

func (r runner) verboseLogger(pipe io.Reader) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		r.logger.Info(scanner.Text())
	}
}
