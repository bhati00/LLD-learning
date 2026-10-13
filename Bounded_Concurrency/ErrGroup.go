package main

import (
	"context"

	"golang.org/x/sync/errgroup"
)

func ErrGroup[T any, R any](ctx context.Context, worker int, items []T, fn func(context.Context, T) (R, error)) ([]R, error) {
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(worker) // bounding the concurrency
	results := make([]R, len(items))

	for i, val := range items {
		// g.Go blocks at the limit and can't see ctx; stop launching once cancelled.
		if gctx.Err() != nil {
			break
		}
		g.Go(func() error {
			result, err := fn(gctx, val)
			if err != nil {
				return err
			}
			results[i] = result
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return results, err
	}
	// gctx is always cancelled after Wait, so report only the parent's state.
	return results, ctx.Err()
}
