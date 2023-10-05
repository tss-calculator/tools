package command

import (
	"context"
	"errors"
	"os/exec"

	applogger "github.com/tss-calculator/go-lib/pkg/application/logger"
)

type Command struct {
	WorkDir    string
	Executable string
	Args       []string
}

type Runner interface {
	Execute(ctx context.Context, command Command) (string, error)
}

func NewCommandRunner(logger applogger.Logger) Runner {
	return &runner{
		logger: logger,
	}
}

type runner struct {
	logger applogger.Logger
}

func (r runner) Execute(ctx context.Context, command Command) (string, error) {
	if command.Executable == "" {
		return "", errors.New("command executable can not be empty")
	}
	// nolint:gosec
	cmd := exec.CommandContext(ctx, command.Executable, command.Args...)
	cmd.Dir = command.WorkDir
	r.logger.Debug(cmd.String())
	result, err := cmd.Output()
	return string(result), err
}
