// Eko: A terminal-native social media platform
// Copyright (C) 2025 Kyren223
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package packet

import (
	"encoding/json"

	"github.com/vmihailenco/msgpack/v5"

	"github.com/kyren223/eko/pkg/assert"
)

type Payload interface {
	Type() PacketType
}

type defaultPacketEncoder struct {
	data       []byte
	encoding   Encoding
	packetType PacketType
}

func (e defaultPacketEncoder) Encoding() Encoding {
	return e.encoding
}

func (e defaultPacketEncoder) Type() PacketType {
	return e.packetType
}

func (e defaultPacketEncoder) Payload() []byte {
	return e.data
}

func NewJsonEncoder(payload Payload) PacketEncoder {
	data, err := json.Marshal(payload)
	assert.NoError(err, "encoding a message with JSON should never fail")

	return defaultPacketEncoder{
		data:       data,
		encoding:   EncodingJson,
		packetType: payload.Type(),
	}
}

func NewMsgPackEncoder(payload Payload) PacketEncoder {
	data, err := msgpack.Marshal(payload)
	assert.NoError(err, "encoding a message with msg pack should never fail")

	return defaultPacketEncoder{
		data:       data,
		encoding:   EncodingMsgPack,
		packetType: payload.Type(),
	}
}
