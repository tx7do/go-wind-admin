package metadata

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/stretchr/testify/assert"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

func ptrUint32(v uint32) *uint32 { return &v }
func ptrBool(b bool) *bool       { return &b }

func TestFromOperatorMetadata_JSON(t *testing.T) {
	ctx := context.Background()

	ds := permissionV1.DataScope(1)
	src := OperatorInfo{
		UserID:    ptrUint32(123),
		TenantID:  ptrUint32(456),
		OrgUnitID: ptrUint32(789),
		DataScope: &ds,
	}
	b, err := json.Marshal(src)
	assert.NoError(t, err)

	ctx = metadata.NewServerContext(ctx, metadata.Metadata{
		mdOperator: []string{string(b)},
	})

	got := FromOperatorMetadata(ctx)
	if assert.NotNil(t, got) {
		assert.NotNil(t, got.UserID)
		assert.Equal(t, uint32(123), *got.UserID)

		assert.NotNil(t, got.TenantID)
		assert.Equal(t, uint32(456), *got.TenantID)

		assert.NotNil(t, got.OrgUnitID)
		assert.Equal(t, uint32(789), *got.OrgUnitID)

		assert.NotNil(t, got.DataScope)
		assert.Equal(t, ds, *got.DataScope)
	}
}

func TestFromOperatorMetadata_NoHeaderOrInvalid(t *testing.T) {
	// no header -> nil
	ctx := context.Background()
	assert.Nil(t, FromOperatorMetadata(ctx))

	// invalid json -> nil
	ctx = metadata.NewServerContext(context.Background(), metadata.Metadata{
		mdOperator: []string{"not-a-json"},
	})
	assert.Nil(t, FromOperatorMetadata(ctx))
}

func TestNewOperatorMetadataContext_WriteAndRead(t *testing.T) {
	ctx := context.Background()

	ds := permissionV1.DataScope(2)
	ctx = NewOperatorMetadataContext(ctx, 321, 654, 987, ds)

	md, ok := metadata.FromClientContext(ctx)
	assert.True(t, ok)

	op := md.Get(mdOperator)
	assert.NotEmpty(t, op)

	var got OperatorInfo
	assert.NoError(t, json.Unmarshal([]byte(op), &got))

	if assert.NotNil(t, got.UserID) {
		assert.Equal(t, uint32(321), *got.UserID)
	}
	if assert.NotNil(t, got.TenantID) {
		assert.Equal(t, uint32(654), *got.TenantID)
	}
	if assert.NotNil(t, got.OrgUnitID) {
		assert.Equal(t, uint32(987), *got.OrgUnitID)
	}
	if assert.NotNil(t, got.DataScope) {
		assert.Equal(t, ds, *got.DataScope)
	}
}
