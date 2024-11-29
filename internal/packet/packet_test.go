package packet

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/pkg/snowflake"
)

func TestPacketEncodingDecoding(t *testing.T) {
	testPacketEncodingDecoding(t, &Error{PktType: 0, Error: ""})

	node := snowflake.NewNode(1)
	id := node.Generate()
	testPacketEncodingDecoding(t, &MessagesInfo{Messages: []data.Message{
		{
			ID:          node.Generate(),
			SenderID:    node.Generate(),
			Content:     "MyMessage",
			FrequencyID: &id,
			ReceiverID:  nil,
		},
		{
			ID:          node.Generate(),
			SenderID:    node.Generate(),
			Content:     "Another Message\nWith a bunch of stuff",
			FrequencyID: nil,
			ReceiverID:  &id,
		},
	}})
}

func testPacketEncodingDecoding(t *testing.T, payload Payload) {
	t.Helper()

	jsonEncoder := NewJsonEncoder(payload)
	jsonPacket := NewPacket(jsonEncoder)

	msgPackEncoder := NewJsonEncoder(payload)
	msgPackPacket := NewPacket(msgPackEncoder)

	jsonPayload, err := jsonPacket.DecodedPayload()
	require.NoError(t, err, "json payload decoding should not fail")
	require.True(t, reflect.DeepEqual(payload, jsonPayload), "json payload mismatch", "got", jsonPayload, "want", payload)

	msgPackPayload, err := msgPackPacket.DecodedPayload()
	require.NoError(t, err, "msgpack payload decoding should not fail")
	require.True(t, reflect.DeepEqual(payload, msgPackPayload), "msgpack payload mismatch", "got", msgPackPayload, "want", payload)
}

func TestPacketFramer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*20)
	defer cancel()
	framer := NewFramer()

	pkt := NewPacket(NewJsonEncoder(&Error{PktType: 0, Error: ""}))
	length := len(pkt.data)
	count := 5
	data := make([]byte, length*count)
	for i := 0; i < count; i++ {
		copy(data[i*length:], pkt.data)
	}

	first := 2
	require.True(t, first < HEADER_SIZE, "TEST ERROR size needs to be less than the header")
	err := framer.Push(ctx, data[:first])
	data = data[first:]
	require.NoError(t, err, "expecting framer to return nil and wait for more data for header")
	require.False(t, doesChannelHaveValue(framer.Out), "expecting channel to block with no value")

	err = framer.Push(ctx, data[:HEADER_SIZE])
	data = data[HEADER_SIZE:]
	require.NoError(t, err, "expecting framer to return nil and wait for more data for payload")
	require.False(t, doesChannelHaveValue(framer.Out), "expecting channel to block with no value")

	err = framer.Push(ctx, data[:length])
	data = data[length:]
	require.NoError(t, err, "expecting framer to return nil and process exactly 1 packet")
	select {
	case p, ok := <-framer.Out:
		require.True(t, ok, "expecting a value from the framer")
		require.True(t, slices.Equal(p.data, pkt.data), "expecting packet to be equal", "got", p.data, "want", pkt.data)
	default:
		require.Fail(t, "expected packet but channel was blocking")
	}
	require.False(t, doesChannelHaveValue(framer.Out), "expecting channel to block with no value because it was consumed already")

	err = framer.Push(ctx, data[:])
	require.NoError(t, err, "expecting framer to return nil and process the rest of the packets")
	for i := 0; i < count-1; i++ {
		select {
		case p, ok := <-framer.Out:
			require.True(t, ok, "expecting a value from the framer")
			require.True(t, slices.Equal(p.data, pkt.data), "expecting packet to be equal", "got", p.data, "want", pkt.data)
		default:
			require.Failf(t, "channel blocked", "%v expected packet", i)
		}
	}
	require.False(t, doesChannelHaveValue(framer.Out), "expecting channel to block with no value because it was consumed already")
}

func doesChannelHaveValue[T any](c <-chan T) bool {
	select {
	case _, ok := <-c:
		return ok
	default:
		return false
	}
}
