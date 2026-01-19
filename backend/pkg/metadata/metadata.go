package metadata

import (
	"context"
	"encoding/json"

	"github.com/go-kratos/kratos/v2/metadata"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

func FromOperatorMetadata(ctx context.Context) *OperatorInfo {
	md, ok := metadata.FromServerContext(ctx)
	if !ok {
		return nil
	}

	if op := md.Get(mdOperator); op != "" {
		var p OperatorInfo
		if err := json.Unmarshal([]byte(op), &p); err == nil {
			return &p
		}
	}

	return nil
}

func NewOperatorMetadataContext(
	ctx context.Context,
	uid uint64,
	tid uint64,
	ouid uint64,
	dataScope permissionV1.DataScope,
) context.Context {
	payload := OperatorInfo{
		UserID:    &uid,
		TenantID:  &tid,
		OrgUnitID: &ouid,
		DataScope: &dataScope,
	}
	if b, err := json.Marshal(payload); err == nil {
		ctx = metadata.AppendToClientContext(ctx, mdOperator, string(b))
	}

	return ctx
}
