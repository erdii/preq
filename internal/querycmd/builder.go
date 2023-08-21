package querycmd

import (
	"context"
	"os/exec"
	"strings"
)

type Builder struct {
	program      string
	argsPatterns []string
}

func NewBuilder(program string, argsPatterns ...string) Builder {
	return Builder{
		program:      program,
		argsPatterns: argsPatterns,
	}
}

func (b Builder) Cmd(ctx context.Context, query, data string) (*exec.Cmd, error) {
	replacedArgs := make([]string, len(b.argsPatterns))
	for i := range b.argsPatterns {
		replacedArgs[i] = strings.ReplaceAll(b.argsPatterns[i], "{+q}", query)
	}

	cmd := exec.CommandContext(ctx, b.program, replacedArgs...)
	cmd.Stdin = strings.NewReader(data)
	return cmd, nil
}
