// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: networks.sql

package data

import (
	"context"

	"github.com/kyren223/eko/pkg/snowflake"
)

const createNetwork = `-- name: CreateNetwork :one
INSERT INTO networks (
  id, owner_id, name, is_public,
  icon, bg_hex_color, fg_hex_color
) VALUES (
  ?, ?, ?, ?,
  ?, ?, ?
)
RETURNING id, owner_id, name, icon, bg_hex_color, fg_hex_color, is_public
`

type CreateNetworkParams struct {
	ID         snowflake.ID
	OwnerID    snowflake.ID
	Name       string
	IsPublic   bool
	Icon       string
	BgHexColor string
	FgHexColor string
}

func (q *Queries) CreateNetwork(ctx context.Context, arg CreateNetworkParams) (Network, error) {
	row := q.db.QueryRowContext(ctx, createNetwork,
		arg.ID,
		arg.OwnerID,
		arg.Name,
		arg.IsPublic,
		arg.Icon,
		arg.BgHexColor,
		arg.FgHexColor,
	)
	var i Network
	err := row.Scan(
		&i.ID,
		&i.OwnerID,
		&i.Name,
		&i.Icon,
		&i.BgHexColor,
		&i.FgHexColor,
		&i.IsPublic,
	)
	return i, err
}

const deleteNetwork = `-- name: DeleteNetwork :exec
DELETE FROM networks WHERE id = ?
`

func (q *Queries) DeleteNetwork(ctx context.Context, id snowflake.ID) error {
	_, err := q.db.ExecContext(ctx, deleteNetwork, id)
	return err
}

const getNetworkById = `-- name: GetNetworkById :one
SELECT id, owner_id, name, icon, bg_hex_color, fg_hex_color, is_public FROM networks
WHERE id = ?
`

func (q *Queries) GetNetworkById(ctx context.Context, id snowflake.ID) (Network, error) {
	row := q.db.QueryRowContext(ctx, getNetworkById, id)
	var i Network
	err := row.Scan(
		&i.ID,
		&i.OwnerID,
		&i.Name,
		&i.Icon,
		&i.BgHexColor,
		&i.FgHexColor,
		&i.IsPublic,
	)
	return i, err
}

const transferNetwork = `-- name: TransferNetwork :one
UPDATE networks SET
  owner_id = ?
WHERE id = ?
RETURNING id, owner_id, name, icon, bg_hex_color, fg_hex_color, is_public
`

type TransferNetworkParams struct {
	OwnerID snowflake.ID
	ID      snowflake.ID
}

func (q *Queries) TransferNetwork(ctx context.Context, arg TransferNetworkParams) (Network, error) {
	row := q.db.QueryRowContext(ctx, transferNetwork, arg.OwnerID, arg.ID)
	var i Network
	err := row.Scan(
		&i.ID,
		&i.OwnerID,
		&i.Name,
		&i.Icon,
		&i.BgHexColor,
		&i.FgHexColor,
		&i.IsPublic,
	)
	return i, err
}

const updateNetwork = `-- name: UpdateNetwork :one
UPDATE networks SET
  name = ?, icon = ?,
  bg_hex_color = ?, fg_hex_color = ?,
  is_public = ?
WHERE id = ?
RETURNING id, owner_id, name, icon, bg_hex_color, fg_hex_color, is_public
`

type UpdateNetworkParams struct {
	Name       string
	Icon       string
	BgHexColor string
	FgHexColor string
	IsPublic   bool
	ID         snowflake.ID
}

func (q *Queries) UpdateNetwork(ctx context.Context, arg UpdateNetworkParams) (Network, error) {
	row := q.db.QueryRowContext(ctx, updateNetwork,
		arg.Name,
		arg.Icon,
		arg.BgHexColor,
		arg.FgHexColor,
		arg.IsPublic,
		arg.ID,
	)
	var i Network
	err := row.Scan(
		&i.ID,
		&i.OwnerID,
		&i.Name,
		&i.Icon,
		&i.BgHexColor,
		&i.FgHexColor,
		&i.IsPublic,
	)
	return i, err
}
