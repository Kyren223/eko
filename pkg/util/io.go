package util

import (
	"context"
	"io"
)

type ChannelReader struct {
	Out <-chan []byte
	Err <-chan error
}

func NewChannelReader(ctx context.Context, reader io.Reader) ChannelReader {
	outCh := make(chan []byte)
	errCh := make(chan error)

	go func(in chan<- []byte, inErr chan<- error) {
		defer close(in)
		defer close(inErr)
	outer:
		for {
			select {
			case <-ctx.Done():
				inErr <- ctx.Err()
				break outer
			default:
				data, err := io.ReadAll(reader)
				if err != nil {
					inErr <- err
					break outer
				}
				in <- data
			}
		}
	}(outCh, errCh)

	return ChannelReader{
		Out: outCh,
		Err: errCh,
	}
}
