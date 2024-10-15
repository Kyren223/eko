package util

import (
	"context"
	"io"
)

const bufferSize = 512

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
		data := make([]byte, bufferSize)
	outer:
		for {
			select {
			case <-ctx.Done():
				inErr <- ctx.Err()
				break outer
			default:
				n, err := reader.Read(data)
				if err != nil && err != io.EOF {
					inErr <- err
					break outer
				}
				in <- data[:n]
			}
		}
	}(outCh, errCh)

	return ChannelReader{
		Out: outCh,
		Err: errCh,
	}
}
