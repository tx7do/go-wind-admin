package data

import (
	"context"
	"encoding/json"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/usercredential"
	passwordPolicy "go-wind-admin/pkg/password"
)

func usercredentialTypePasswordHash() usercredential.CredentialType {
	return usercredential.CredentialTypePasswordHash
}

// passwordHistoryKey 是 extra_info JSON 里历史口令哈希数组的键。
const passwordHistoryKey = "password_history"

// checkPasswordHistory 等保口令策略：新明文口令不得与历史哈希列表中任一条
// 相同（bcrypt 比对）。历史列表为空/解析失败时跳过（不阻塞主流程）。
func (r *UserCredentialRepo) checkPasswordHistory(_ context.Context, entity *ent.UserCredential, newPlain string) error {
	limit := passwordPolicy.HistoryCount()
	if limit <= 0 || entity == nil || entity.CredentialType == nil {
		return nil
	}
	if *entity.CredentialType != usercredentialTypePasswordHash() {
		return nil
	}
	var extra map[string]any
	if entity.ExtraInfo != nil && *entity.ExtraInfo != "" {
		if err := json.Unmarshal([]byte(*entity.ExtraInfo), &extra); err != nil {
			// extra_info 非历史格式（可能存了别的业务数据），跳过历史检查
			return nil
		}
	}
	arr, _ := extra[passwordHistoryKey].([]any)
	for _, item := range arr {
		hash, ok := item.(string)
		if !ok || hash == "" {
			continue
		}
		if ok, err := r.passwordCrypto.Verify(newPlain, hash); err == nil && ok {
			return authenticationV1.ErrorBadRequest("new password must differ from recent passwords")
		}
	}
	return nil
}

// appendPasswordHistory 把旧口令哈希追加进 extra_info 的历史列表，
// 只保留最近 limit 条；extra_info 里已有的其他键原样保留。
func appendPasswordHistory(extraInfo *string, oldHash string, limit int) *string {
	if limit <= 0 {
		return extraInfo
	}
	var extra map[string]any
	if extraInfo != nil && *extraInfo != "" {
		if err := json.Unmarshal([]byte(*extraInfo), &extra); err != nil || extra == nil {
			// 解析失败：历史让位，保住其他语义已不可恢复时以新结构重建
			extra = make(map[string]any)
		}
	} else {
		extra = make(map[string]any)
	}

	arr, _ := extra[passwordHistoryKey].([]any)
	history := make([]string, 0, limit)
	for _, item := range arr {
		if s, ok := item.(string); ok && s != "" {
			history = append(history, s)
		}
	}
	if oldHash != "" {
		history = append(history, oldHash)
	}
	if len(history) > limit {
		history = history[len(history)-limit:]
	}

	strs := make([]any, 0, len(history))
	for _, h := range history {
		strs = append(strs, h)
	}
	extra[passwordHistoryKey] = strs

	b, err := json.Marshal(extra)
	if err != nil {
		return extraInfo
	}
	s := string(b)
	return &s
}
