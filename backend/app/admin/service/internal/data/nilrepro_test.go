package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"

	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// 复现 Update 带 role_ids（mask 含 role_ids）时的空指针 panic。
func TestReproNilPanicOnRoleIdsUpdate(t *testing.T) {
	r := newUserRepoSqlite(t)
	ctx := enttest.NewSystemContext(context.Background())

	created, err := r.Create(ctx, &identityV1.CreateUserRequest{
		Data: &identityV1.User{
			Username: trans.Ptr("nilpanic_u"),
			Status:   identityV1.User_NORMAL.Enum(),
		},
	})
	require.NoError(t, err)

	req := &identityV1.UpdateUserRequest{
		Id:         *created.Id,
		Data:       &identityV1.User{RoleIds: []uint64{1}},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"role_ids"}},
	}
	require.NotPanics(t, func() {
		err = r.Update(ctx, req)
	})
	require.NoError(t, err)
}
