package util

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestChannelReader(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	b := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	reader := NewChannelReader(ctx, bytes.NewReader(b))

	select {
	case data := <-reader.Out:
		if !bytes.Equal(b, data) {
			t.Errorf("%v != %v", b, data)
		}
	case err := <-reader.Err:
		t.Errorf("reading err: %v", err)
	}
}

func TestChannelReaderMultiPartRead(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	b := []byte("Testing FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting FramerTesting Framer")
	b1 := b[:512]
	b2 := b[512:]
	reader := NewChannelReader(ctx, bytes.NewReader(b))

	counter := 0
outer:
	for {
		select {
		case data := <-reader.Out:
			if counter == 0 {
			if !bytes.Equal(b1, data) {
				t.Errorf("%v != %v", b, data)
			}
			} else {
				if !bytes.Equal(b2[:len(data)], data) {
				t.Errorf("%v != %v", b, data)
			}
			}
			counter++
			if counter == 2 {
				break outer
			}

		case err := <-reader.Err:
			t.Errorf("reading err: %v", err)
			break outer
		}
	}
}
