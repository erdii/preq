package querycmd

import "context"

func Execute(ctx context.Context, builder Builder, query, data string) (string, error) {
	cmd, err := builder.Cmd(ctx, query, data)
	if err != nil {
		return "", err
	}

	bytes, err := cmd.CombinedOutput()
	return string(bytes), err
}
