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
			t.Errorf("TestChannelReader() %v != %v", b, data)
		}
	case err := <-reader.Err:
		t.Errorf("TestChannelReader() err = %v", err)
	}
}
