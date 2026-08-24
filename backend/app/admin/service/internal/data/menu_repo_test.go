package data

import (
	"strings"
	"testing"

	"google.golang.org/genproto/protobuf/field_mask"

	entgoUpdate "github.com/tx7do/go-crud/entgo/update"

	"github.com/tx7do/go-utils/fieldmaskutil"
	"github.com/tx7do/go-utils/trans"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

func TestMenuMetaFieldMask(t *testing.T) {
	updateMenuReq := &permissionV1.UpdateMenuRequest{
		Data: &permissionV1.Menu{
			Meta: &permissionV1.MenuMeta{
				Title: trans.Ptr("标题1"),
				Order: trans.Ptr(int32(1)),
			},
		},
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"id", "meta", "meta.order", "meta.title"},
		},
	}
	var metaPaths []string
	for _, v := range updateMenuReq.UpdateMask.GetPaths() {
		if strings.HasPrefix(v, "meta.") {
			metaPaths = append(metaPaths, strings.SplitAfter(v, "meta.")[1])
		}
	}
	updateMenuReq.UpdateMask.Normalize()
	if !updateMenuReq.UpdateMask.IsValid(updateMenuReq.Data) {
		// Return an error.
		panic("invalid field mask")
	}
	fieldmaskutil.Filter(updateMenuReq.GetData(), updateMenuReq.UpdateMask.GetPaths())

	fieldmaskutil.Filter(updateMenuReq.GetData().Meta, metaPaths)

	nilPaths := fieldmaskutil.NilValuePaths(updateMenuReq.GetData().Meta, metaPaths)
	keyValues := entgoUpdate.ExtractJsonFieldKeyValues(updateMenuReq.GetData().Meta, metaPaths, false)

	t.Logf("UPDATE: [%v] [%v] [%v] [%v]", updateMenuReq.Data, updateMenuReq.Data.Meta, nilPaths, keyValues)
}
