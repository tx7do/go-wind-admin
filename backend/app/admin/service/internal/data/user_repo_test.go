package data

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/protobuf/proto"

	"github.com/tx7do/go-utils/crypto"
	"github.com/tx7do/go-utils/fieldmaskutil"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"

	"go-wind-admin/app/admin/service/internal/data/ent"
)

var reSpaces = regexp.MustCompile(`\s+`)

func TestUserFieldMask(t *testing.T) {
	u := &identityV1.User{
		Username: trans.String("UserName"),
		Realname: trans.String("RealName"),
		//Avatar:   trans.String("Avatar"),
		Address: trans.String("Address"),
	}

	updateUserReq := &identityV1.UpdateUserRequest{
		Data: &identityV1.User{
			Username: trans.String("UserName1"),
			Realname: trans.String("RealName1"),
			//Avatar:   trans.String("Avatar1"),
			Address: trans.String("Address1"),
		},
		// paths 必须与 proto 字段名一致：username/realname 是一词字段，
		// Normalize 会把 userName/realName 转成 user_name/real_name 导致校验失败
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"username", "realname", "avatar", "role_id"},
		},
	}
	updateUserReq.UpdateMask.Normalize()
	if !updateUserReq.UpdateMask.IsValid(u) {
		t.Fatalf("invalid field mask: %v", updateUserReq.UpdateMask.GetPaths())
	}

	fieldmaskutil.Filter(updateUserReq.GetData(), updateUserReq.UpdateMask.GetPaths())
	proto.Merge(u, updateUserReq.GetData())

	// mask 内字段被更新，mask 外字段保持原值
	assert.Equal(t, "UserName1", u.GetUsername())
	assert.Equal(t, "RealName1", u.GetRealname())
	assert.Equal(t, "Address", u.GetAddress(), "address not in mask, should keep origin")

	fmt.Println(reSpaces.ReplaceAllString(u.String(), " "))
}

func TestFilterReuseMask(t *testing.T) {
	users := []*identityV1.User{
		{
			Id:       trans.Ptr(int64(1)),
			Username: trans.String("name 1"),
		},
		{
			Id:       trans.Ptr(int64(2)),
			Username: trans.String("name 2"),
		},
	}
	// Create a mask only once and reuse it.
	mask := fieldmaskutil.NestedMaskFromPaths([]string{"userName", "realName", "positionId"})
	for _, u := range users {
		mask.Filter(u)
	}
	fmt.Println(users)
	assert.Equal(t, len(users), 2)
	// Output: [userName:"name 1" userName:"name 2"]
}

func TestNilValuePaths(t *testing.T) {
	u := &identityV1.User{
		Id:       trans.Ptr(int64(2)),
		Username: trans.String("name 2"),
		//RealName: trans.String(""),
	}
	paths := []string{"userName", "realName", "positionId"}
	nilPaths := fieldmaskutil.NilValuePaths(u, paths)
	fmt.Println(nilPaths)
	fmt.Println(u.PositionId)
}

func TestMessageNil(t *testing.T) {
	u := &identityV1.User{
		Id:       trans.Ptr(int64(2)),
		Username: trans.String("name 2"),
	}

	pr := u.ProtoReflect()
	md := pr.Descriptor()
	// proto 字段名是 username（一词），ByName("userName") 恒为 nil
	fd := md.Fields().ByName("username")
	if fd == nil {
		t.Fatalf("field descriptor not found for username")
	}

	v := pr.Get(fd)
	fmt.Println(v)
	assert.Equal(t, "name 2", v.Interface().(string))
}

func TestAuthEnum(t *testing.T) {
	fmt.Println(authenticationV1.GrantType_password.String())
	fmt.Println(authenticationV1.GrantType_client_credentials.String())
	fmt.Println(authenticationV1.GrantType_refresh_token.String())

	fmt.Println(authenticationV1.TokenType_bearer.String())
	fmt.Println(authenticationV1.TokenType_mac.String())
}

func TestDecryptAES(t *testing.T) {
	//key的长度必须是16、24或者32字节，分别用于选择AES-128, AES-192, or AES-256
	aesKey := crypto.DefaultAESKey

	plainText := []byte("admin")
	encryptText, err := crypto.AesEncrypt(plainText, aesKey, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	pass64 := base64.StdEncoding.EncodeToString(encryptText)
	fmt.Printf("加密后:%v\n", pass64)

	bytesPass, err := base64.StdEncoding.DecodeString(pass64)
	if err != nil {
		fmt.Println(err)
		return
	}

	decryptText, err := crypto.AesDecrypt(bytesPass, aesKey, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("解密后:%s\n", decryptText)
	assert.Equal(t, plainText, decryptText)
}

func TestCopier(t *testing.T) {
	{
		var entMsg ent.User
		var protoMsg identityV1.User

		entMsg.ID = 1
		entMsg.Username = trans.Ptr("Username")
		entMsg.Nickname = trans.Ptr("NickName")
		entMsg.Realname = trans.Ptr("RealName")
		entMsg.Email = trans.Ptr("test@gmail.com")
		entMsg.TenantID = trans.Ptr(int64(2))

		_ = copier.Copy(&protoMsg, entMsg)
		assert.Equal(t, protoMsg.GetUsername(), *entMsg.Username)
		assert.Equal(t, protoMsg.GetNickname(), *entMsg.Nickname)
		assert.Equal(t, protoMsg.GetRealname(), *entMsg.Realname)
		assert.Equal(t, protoMsg.GetEmail(), *entMsg.Email)
		// GetTenantId() 返回值类型，与指针比较恒不等，需解引用
		assert.Equal(t, protoMsg.GetTenantId(), *entMsg.TenantID)
		assert.Equal(t, protoMsg.GetId(), entMsg.ID)
	}

	{
		var entMsg ent.User
		var protoMsg identityV1.User

		_ = copier.Copy(&entMsg, &protoMsg)
	}

	{
		var in struct {
			IntArray []int32
		}
		var out struct {
			IntArray []int
		}

		in.IntArray = []int32{1}
		_ = copier.Copy(&out, &in)
		fmt.Println("IntArray: ", out.IntArray)

		out.IntArray = []int{3}
		_ = copier.Copy(&in, &out)
		fmt.Println("IntArray: ", in.IntArray)
	}

	{
		var in struct {
			Int *int32
		}
		var out struct {
			Int int32
		}

		in.Int = trans.Ptr(int32(1))
		_ = copier.Copy(&out, &in)
		fmt.Println("Int32: ", out.Int)

		out.Int = 3
		_ = copier.Copy(&in, &out)
		fmt.Println("Int32: ", *in.Int)
	}

	{
		var entMsg ent.User
		var protoMsg identityV1.User

		entMsg.ID = 1
		entMsg.Username = trans.Ptr("Username")
		entMsg.Nickname = trans.Ptr("NickName")
		entMsg.Realname = trans.Ptr("RealName")
		entMsg.Email = trans.Ptr("test@gmail.com")
		entMsg.CreatedAt = trans.Ptr(time.Now())

		converter := copier.TypeConverter{
			SrcType: &time.Time{},  // 源类型
			DstType: trans.Ptr(""), // 目标类型
			Fn: func(src interface{}) (interface{}, error) {
				return timeutil.TimeToTimeString(src.(*time.Time)), nil
			},
		}

		option := copier.Option{
			Converters: []copier.TypeConverter{converter},
		}

		err := copier.CopyWithOption(&protoMsg, &entMsg, option)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		fmt.Println(protoMsg.GetUsername(), protoMsg.GetCreatedAt())
	}
}
