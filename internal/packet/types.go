package packet

import (
	"crypto/ed25519"

	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/pkg/snowflake"
)

type Error struct {
	Error   string
	PktType PacketType
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
	CreateNetwork
	Network snowflake.ID
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

type SetMember struct {
	Member    *bool
	Admin     *bool
	Muted     *bool
	Banned    *bool
	BanReason *string
	Network   snowflake.ID
	User      snowflake.ID
}

func (m *SetMember) Type() PacketType {
	return PacketSetMember
}

type FullNetwork struct {
	data.Network
	Frequencies []data.Frequency
	Members     []data.Member
	Users       []data.User
}

type NetworksInfo struct {
	Networks        []FullNetwork
	RemovedNetworks []snowflake.ID
	Partial         bool
}

func (m *NetworksInfo) Type() PacketType {
	return PacketNetworksInfo
}

type CreateFrequency struct {
	Name     string
	HexColor string
	Network  snowflake.ID
	Perms    int
}

func (m *CreateFrequency) Type() PacketType {
	return PacketCreateFrequency
}

type UpdateFrequency struct {
	Name      string
	HexColor  string
	Frequency snowflake.ID
	Perms     int
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

type FrequenciesInfo struct {
	RemovedFrequencies []snowflake.ID
	Frequencies        []data.Frequency
	Network            snowflake.ID
}

func (m *FrequenciesInfo) Type() PacketType {
	return PacketFrequenciesInfo
}

type SendMessage struct {
	ReceiverID  *snowflake.ID
	FrequencyID *snowflake.ID
	Content     string
	Ping        *snowflake.ID
}

func (m *SendMessage) Type() PacketType {
	return PacketSendMessage
}

type EditMessage struct {
	Content string
	Message snowflake.ID
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
	Messages        []data.Message
	RemovedMessages []snowflake.ID
}

func (m *MessagesInfo) Type() PacketType {
	return PacketMessagesInfo
}

type MembersInfo struct {
	RemovedMembers []snowflake.ID
	Members        []data.Member
	Users          []data.User
	Network        snowflake.ID
}

func (m *MembersInfo) Type() PacketType {
	return PacketMembersInfo
}

type SetUserData struct {
	Data *string
	User *data.User
}

func (m *SetUserData) Type() PacketType {
	return PacketSetUserData
}

type GetUserData struct{}

func (m *GetUserData) Type() PacketType {
	return PacketGetUserData
}

type TrustUser struct {
	User  snowflake.ID
	Trust bool
}

func (m *TrustUser) Type() PacketType {
	return PacketTrustUser
}

type TrustInfo struct {
	Trusteds          []snowflake.ID
	TrustedPublicKeys []ed25519.PublicKey
	RemovedTrusteds   []snowflake.ID
}

func (m *TrustInfo) Type() PacketType {
	return PacketTrustInfo
}

type GetBannedMembers struct {
	Network snowflake.ID
}

func (m *GetBannedMembers) Type() PacketType {
	return PacketGetBannedMembers
}

type SetLastReadMessages struct {
	Source   []snowflake.ID
	LastRead []int64
}

func (m *SetLastReadMessages) Type() PacketType {
	return PacketSetLastReadMessages
}

type NotificationsInfo struct {
	Source []snowflake.ID
	Pings  []*int64
}

func (m *NotificationsInfo) Type() PacketType {
	return PacketNotificationsInfo
}
