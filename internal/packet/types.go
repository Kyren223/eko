package packet

import (
	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/pkg/snowflake"
)

type Error struct {
	Error string
}

func (m *Error) Type() PacketType {
	return PacketError
}

type CreateNetwork struct {
	Name       string
	Icon       string
	BgHexColor string
	FgHexColor string
	IsPublic   bool
}

func (m *CreateNetwork) Type() PacketType {
	return PacketCreateNetwork
}

type UpdateNetwork struct {
	Network snowflake.ID
	CreateNetwork
}

func (m *UpdateNetwork) Type() PacketType {
	return PacketUpdateNetwork
}

type TransferNetwork struct {
	Network snowflake.ID
	User    snowflake.ID
}

func (m *TransferNetwork) Type() PacketType {
	return PacketTransferNetwork
}

type DeleteNetwork struct {
	Network snowflake.ID
}

func (m *DeleteNetwork) Type() PacketType {
	return PacketDeleteNetwork
}

type SetNetworkUser struct {
	Network   snowflake.ID
	User      snowflake.ID
	Member    *bool
	Admin     *bool
	Muted     *bool
	Banned    *bool
	BanReason *string
}

func (m *SetNetworkUser) Type() PacketType {
	return PacketSetNetworkUser
}

type NetworksInfo struct {
	Networks []struct {
		Network data.Network
		Frequencies []data.Frequency
		Members []data.User
	}
}

func (m *NetworksInfo) Type() PacketType {
	return PacketNetworksInfo
}

type CreateFrequency struct {
	Network  snowflake.ID
	Name     string
	HexColor string
	Perms    int
}

func (m *CreateFrequency) Type() PacketType {
	return PacketCreateFrequency
}

type UpdateFrequency struct {
	Frequency snowflake.ID
	Name      string
	HexColor  string
	Perms     byte
}

func (m *UpdateFrequency) Type() PacketType {
	return PacketUpdateFrequency
}

type DeleteFrequency struct {
	Frequency snowflake.ID
}

func (m *DeleteFrequency) Type() PacketType {
	return PacketDeleteFrequency
}

type SwapFrequencies struct {
	Network snowflake.ID
	Pos1    int
	Pos2    int
}

func (m *SwapFrequencies) Type() PacketType {
	return PacketSwapFrequencies
}

type SendMessage struct {
	ReceiverID  *snowflake.ID
	FrequencyID *snowflake.ID
	Content     string
}

func (m *SendMessage) Type() PacketType {
	return PacketSendMessage
}

type EditMessage struct {
	Message snowflake.ID
	Content string
}

func (m *EditMessage) Type() PacketType {
	return PacketEditMessage
}

type DeleteMessage struct {
	Message snowflake.ID
}

func (m *DeleteMessage) Type() PacketType {
	return PacketDeleteMessage
}

type RequestMessages struct {
	ReceiverID  *snowflake.ID
	FrequencyID *snowflake.ID
}

func (m *RequestMessages) Type() PacketType {
	return PacketRequestMessages
}

type MessagesInfo struct {
	Messages []data.Message
}

func (m *MessagesInfo) Type() PacketType {
	return PacketMessagesInfo
}
