package api

import (
	"context"
	"strings"

	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/snowflake"
)

const hex = "0123456789abcdefABCDEF"

func isValidHexColor(color string) (bool, string) {
	if len(color) != 7 {
		return false, "color must be hex with length of 7"
	}

	if color[0] != '#' {
		return false, "color must start with '#'"
	}

	for _, c := range color[1:] {
		if !strings.ContainsRune(hex, c) {
			return false, "color must start with '#' and contain exactly 6 digits 0-9, a-f, A-F"
		}
	}

	return true, ""
}

func IsNetworkAdmin(ctx context.Context, queries *data.Queries, userId, networkId snowflake.ID) (bool, error) {
	userNetwork, err := queries.GetUserNetwork(ctx, data.GetUserNetworkParams{
		UserID:    userId,
		NetworkID: networkId,
	})
	if err != nil {
		return false, err
	}

	isAdmin := userNetwork.IsAdmin && userNetwork.IsMember && !userNetwork.IsBanned
	return isAdmin, nil
}

func GetSingleNetworkInfo(ctx context.Context, queries *data.Queries, network data.Network) (packet.Payload, error) {
	frequencies, err := queries.GetNetworkFrequencies(ctx, network.ID)
	if err != nil {
		return nil, err
	}

	members, err := queries.GetNetworkMembers(ctx, network.ID)
	if err != nil {
		return nil, err
	}

	fullNetwork := packet.FullNetwork{
		Network:     network,
		Frequencies: frequencies,
		Members:     members,
		Position:    -1,
	}

	return &packet.NetworksInfo{
		Networks:       []packet.FullNetwork{fullNetwork},
		RemoveNetworks: nil,
		Set:            false,
	}, nil
}
