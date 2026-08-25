package codec

import "context"

func abortReconContext() error {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}
