package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/notificationchannel"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"
	"go-wind-admin/pkg/crypto"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	notificationChannelV1 "go-wind-admin/api/gen/go/notification_channel/service/v1"
)

type NotificationChannelRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

	mapper        *mapper.CopierMapper[notificationChannelV1.NotificationChannel, ent.NotificationChannel]
	typeConverter *mapper.EnumTypeConverter[notificationChannelV1.NotificationChannel_Type, notificationchannel.Type]
	tlsConverter  *mapper.EnumTypeConverter[notificationChannelV1.NotificationChannel_TlsMode, notificationchannel.SMTPTLS]
	// styleConverter 按枚举名字字符串配对，所以 proto 的成员名必须与 ent 列值逐字相同
	// （CUSTOM / NONE / DINGTALK / FEISHU / WECOM）。给任何一侧加前缀都不会报错，只会让
	// 这一列在读路径上静默变成 nil——proto 侧的注释里钉着同一条约束。
	styleConverter *mapper.EnumTypeConverter[notificationChannelV1.SignStyle, notificationchannel.WebhookSignStyle]

	repository *entCrud.Repository[
		ent.NotificationChannelQuery, ent.NotificationChannelSelect,
		ent.NotificationChannelCreate, ent.NotificationChannelCreateBulk,
		ent.NotificationChannelUpdate, ent.NotificationChannelUpdateOne,
		ent.NotificationChannelDelete,
		predicate.NotificationChannel,
		notificationChannelV1.NotificationChannel, ent.NotificationChannel,
	]
}

func NewNotificationChannelRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *NotificationChannelRepo {
	return newNotificationChannelRepo(ctx.NewLoggerHelper("notification-channel/repo/admin-service"), entClient)
}

// newNotificationChannelRepo 装配一条渠道 repo：converter 清单只此一份。
//
// 为什么单独拆出来：测试侧要换 NopLogger 与 SQLite client，此前各自逐字段复刻了一份，
// 于是"加一个 converter"要在三处同步——漏一处的表现不是编译失败，而是第一次读路径
// 在 copier 的 nil 接收者上 panic。
func newNotificationChannelRepo(log *bLogger.Helper, entClient *entCrud.EntClient[*ent.Client]) *NotificationChannelRepo {
	repo := &NotificationChannelRepo{
		log:       log,
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[notificationChannelV1.NotificationChannel, ent.NotificationChannel](),
		typeConverter: mapper.NewEnumTypeConverter[notificationChannelV1.NotificationChannel_Type, notificationchannel.Type](
			notificationChannelV1.NotificationChannel_Type_name, notificationChannelV1.NotificationChannel_Type_value,
		),
		tlsConverter: mapper.NewEnumTypeConverter[notificationChannelV1.NotificationChannel_TlsMode, notificationchannel.SMTPTLS](
			notificationChannelV1.NotificationChannel_TlsMode_name, notificationChannelV1.NotificationChannel_TlsMode_value,
		),
		styleConverter: mapper.NewEnumTypeConverter[notificationChannelV1.SignStyle, notificationchannel.WebhookSignStyle](
			notificationChannelV1.SignStyle_name, notificationChannelV1.SignStyle_value,
		),
	}

	repo.init()

	return repo
}

func (r *NotificationChannelRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.NotificationChannelQuery, ent.NotificationChannelSelect,
		ent.NotificationChannelCreate, ent.NotificationChannelCreateBulk,
		ent.NotificationChannelUpdate, ent.NotificationChannelUpdateOne,
		ent.NotificationChannelDelete,
		predicate.NotificationChannel,
		notificationChannelV1.NotificationChannel, ent.NotificationChannel,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.tlsConverter.NewConverterPair())
	r.mapper.AppendConverters(r.styleConverter.NewConverterPair())
}

// List 分页查询通知渠道（密码字段不出现在 DTO，靠 HasPassword 标识）。
func (r *NotificationChannelRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*notificationChannelV1.ListNotificationChannelResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().NotificationChannel.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &notificationChannelV1.ListNotificationChannelResponse{Total: 0, Items: nil}, nil
	}

	// 填充 HasPassword / HasWebhookSecret 标识：一次批量查询取两列的有无
	r.querySecretFlagsByIDs(ctx, ret.Items)
	// Type 回填（同 Get：mapper 无法赋入指针字段，见 Get 处注释）
	r.queryTypeByIDs(ctx, ret.Items)
	// Enabled 回填（实体侧为 status 枚举列，copier 字段名失配，见 Get 处注释）
	r.queryEnabledByIDs(ctx, ret.Items)

	return &notificationChannelV1.ListNotificationChannelResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

// querySecretFlagsByIDs 为 DTO 列表填充两列"是否已配置凭据"标识。
//
// 为什么单独查一遍而不是顺手从 mapper 拿：smtp_password / webhook_secret 都是
// Sensitive 列且 DTO 根本没有对应字段（只有 has* 布尔），读路径不该把密文带出去。
func (r *NotificationChannelRepo) querySecretFlagsByIDs(ctx context.Context, items []*notificationChannelV1.NotificationChannel) {
	if len(items) == 0 {
		return
	}
	entities, err := r.entClient.Client().NotificationChannel.Query().
		Select(notificationchannel.FieldID, notificationchannel.FieldSMTPPassword, notificationchannel.FieldWebhookSecret).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query credential flags failed: %s", err.Error())
		return
	}
	hasSmtpPwd := make(map[uint32]bool, len(entities))
	hasWebhookSecret := make(map[uint32]bool, len(entities))
	for _, e := range entities {
		hasSmtpPwd[e.ID] = e.SMTPPassword != nil
		hasWebhookSecret[e.ID] = e.WebhookSecret != nil
	}
	for _, it := range items {
		it.HasPassword = trans.Ptr(hasSmtpPwd[it.GetId()])
		it.HasWebhookSecret = trans.Ptr(hasWebhookSecret[it.GetId()])
	}
}

// queryTypeByIDs 为 DTO 列表回填渠道类型（WEBHOOK/EMAIL）。
//
// 枚举转换机制注记：mapper 经 EnumTypeConverter.NewConverterPair 注册的
// 是**指针↔指针对**（*ent枚举 → *proto枚举）——实体侧可空指针枚举列（如
// 本仓 SMTPTLS）在 copier 的转换查表里精确命中、读路径本就如实流通。
// 被丢弃的是**混合形态**：实体侧为值型枚举列（带 Default 且无 Nillable，
// 即本仓 Type）而 DTO 侧为可选指针字段——值型源与指针对键失配，copier
// 转而给 DTO 指针分配零值（WEBHOOK 渠道读视图曾恒呈缺省 EMAIL，下游
// SendTestEmail 的"仅 EMAIL"守卫因此失效）。故只对 Type 做批量回填。
func (r *NotificationChannelRepo) queryTypeByIDs(ctx context.Context, items []*notificationChannelV1.NotificationChannel) {
	if len(items) == 0 {
		return
	}
	entities, err := r.entClient.Client().NotificationChannel.Query().
		Select(notificationchannel.FieldID, notificationchannel.FieldType).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query channel types failed: %s", err.Error())
		return
	}
	types := make(map[uint32]notificationchannel.Type, len(entities))
	for _, e := range entities {
		if e.Type != "" {
			types[e.ID] = e.Type
		}
	}
	for _, it := range items {
		if t, ok := types[it.GetId()]; ok {
			tv := t
			if p := r.typeConverter.ToDTO(&tv); p != nil {
				it.Type = p
			}
		}
	}
}

// queryEnabledByIDs 为 DTO 列表回填 Enabled 标识（status 枚举列 → 布尔，
// copier 字段名失配同 Type/HasPassword，见 Get 处注释）。
func (r *NotificationChannelRepo) queryEnabledByIDs(ctx context.Context, items []*notificationChannelV1.NotificationChannel) {
	if len(items) == 0 {
		return
	}
	entities, err := r.entClient.Client().NotificationChannel.Query().
		Select(notificationchannel.FieldID, notificationchannel.FieldStatus).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query channel statuses failed: %s", err.Error())
		return
	}
	statuses := make(map[uint32]notificationchannel.Status, len(entities))
	for _, e := range entities {
		if e.Status != nil {
			statuses[e.ID] = *e.Status
		}
	}
	for _, it := range items {
		s, ok := statuses[it.GetId()]
		it.Enabled = trans.Ptr(ok && s == notificationchannel.StatusOn)
	}
}

func (r *NotificationChannelRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().NotificationChannel.Query().
		Where(notificationchannel.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query notification channel exist failed: %s", err.Error())
		return false, adminV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

// Get 按 ID 查询渠道（脱敏：DTO 不含密码，仅 HasPassword 标识）。
func (r *NotificationChannelRepo) Get(ctx context.Context, id uint32) (*notificationChannelV1.NotificationChannel, error) {
	entity, err := r.entClient.Client().NotificationChannel.Get(ctx, id)
	if err != nil {
		r.log.Errorf(ctx, "get notification channel [%d] failed: %s", id, err.Error())
		return nil, adminV1.ErrorNotFound("notification channel not found")
	}
	dto := r.mapper.ToDTO(entity)
	dto.HasPassword = trans.Ptr(entity.SMTPPassword != nil)
	dto.HasWebhookSecret = trans.Ptr(entity.WebhookSecret != nil)
	// Enabled 回填：写路径把 enabled 布尔落为 status 枚举列（statusFromEnabled），
	// copier 按字段名映射 Status≠Enabled，读视图不回填则恒呈停用态。
	dto.Enabled = trans.Ptr(entity.Status != nil && *entity.Status == notificationchannel.StatusOn)
	// Type 回填：本仓 Type 为值型枚举列（带 Default、无 Nillable），DTO 侧
	// 为可选指针字段——mapper 注册的是指针↔指针转换对，值型源与其失配，
	// copier 给 DTO 指针分配零值（此前 WEBHOOK 渠道读视图恒呈缺省 EMAIL，
	// 下游 SendTestEmail 的"仅 EMAIL"守卫因此失效）。经仓内既有的
	// typeConverter（实体枚举名 → proto 枚举值）回填；SMTPTLS 为可空指针
	// 枚举列、指针对精确命中，读路径无需回填。
	if entity.Type != "" {
		t := entity.Type
		dto.Type = r.typeConverter.ToDTO(&t)
	}
	return dto, nil
}

// Create 创建渠道；password 明文经 EncryptIfNeeded 加密后落库。
func (r *NotificationChannelRepo) Create(ctx context.Context, req *notificationChannelV1.CreateNotificationChannelRequest, operatorID uint32) (uint32, error) {
	if req == nil || req.Data == nil {
		return 0, adminV1.ErrorBadRequest("invalid request")
	}
	if req.Data.GetName() == "" {
		return 0, adminV1.ErrorBadRequest("channel name is required")
	}

	encrypted, err := crypto.EncryptIfNeeded(req.GetPassword())
	if err != nil {
		r.log.Errorf(ctx, "encrypt smtp password failed: %s", err.Error())
		return 0, adminV1.ErrorInternalServerError("encrypt password failed")
	}

	// 签名密钥与 SMTP 密码同一条加密路径（EncryptIfNeeded：已是密文则原样存，明文则加密）。
	secret, err := crypto.EncryptIfNeeded(req.GetWebhookSecret())
	if err != nil {
		r.log.Errorf(ctx, "encrypt webhook secret failed: %s", err.Error())
		return 0, adminV1.ErrorInternalServerError("encrypt webhook secret failed")
	}

	builder := r.entClient.Client().NotificationChannel.Create().
		SetName(req.Data.GetName()).
		SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
		SetNillableSMTPHost(req.Data.SmtpHost).
		SetNillableSMTPPort(req.Data.SmtpPort).
		SetNillableSMTPUsername(req.Data.SmtpUsername).
		SetNillableSMTPFrom(req.Data.SmtpFrom).
		SetNillableSMTPTLS(r.tlsConverter.ToEntity(req.Data.SmtpTls)).
		SetNillableWebhookURL(req.Data.WebhookUrl).
		SetNillableWebhookSignStyle(r.styleConverter.ToEntity(req.Data.WebhookSignStyle)).
		SetNillableWebhookPayloadTemplate(req.Data.WebhookPayloadTemplate).
		SetStatus(statusFromEnabled(req.Data.GetEnabled())).
		SetNillableRemark(req.Data.Remark).
		SetCreatedBy(operatorID).
		SetCreatedAt(time.Now())

	if req.GetPassword() != "" {
		builder.SetSMTPPassword(encrypted)
	}
	if req.GetWebhookSecret() != "" {
		builder.SetWebhookSecret(secret)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf(ctx, "insert notification channel failed: %s", err.Error())
		return 0, adminV1.ErrorInternalServerError("insert notification channel failed")
	}

	return created.ID, nil
}

// Update 更新渠道；password 留空表示不修改已存密码。
func (r *NotificationChannelRepo) Update(ctx context.Context, req *notificationChannelV1.UpdateNotificationChannelRequest, operatorID uint32) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid request")
	}
	if req.GetId() == 0 {
		return adminV1.ErrorBadRequest("id is required")
	}

	builder := r.entClient.Client().NotificationChannel.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *notificationChannelV1.NotificationChannel) {
			builder.
				SetNillableName(req.Data.Name).
				SetNillableSMTPHost(req.Data.SmtpHost).
				SetNillableSMTPPort(req.Data.SmtpPort).
				SetNillableSMTPUsername(req.Data.SmtpUsername).
				SetNillableSMTPFrom(req.Data.SmtpFrom).
				SetNillableSMTPTLS(r.tlsConverter.ToEntity(req.Data.SmtpTls)).
				SetNillableWebhookURL(req.Data.WebhookUrl).
				SetNillableWebhookSignStyle(r.styleConverter.ToEntity(req.Data.WebhookSignStyle)).
				SetNillableWebhookPayloadTemplate(req.Data.WebhookPayloadTemplate).
				SetNillableStatus(r.statusFromProto(req.Data.Enabled)).
				SetNillableRemark(req.Data.Remark).
				SetNillableUpdatedBy(trans.Ptr(operatorID)).
				SetUpdatedAt(time.Now())
			if req.GetPassword() != "" {
				encrypted, encErr := crypto.EncryptIfNeeded(req.GetPassword())
				if encErr != nil {
					r.log.Errorf(ctx, "encrypt smtp password failed: %s", encErr.Error())
					return
				}
				builder.SetSMTPPassword(encrypted)
			}
			if req.GetWebhookSecret() != "" {
				secret, encErr := crypto.EncryptIfNeeded(req.GetWebhookSecret())
				if encErr != nil {
					r.log.Errorf(ctx, "encrypt webhook secret failed: %s", encErr.Error())
					return
				}
				builder.SetWebhookSecret(secret)
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(notificationchannel.FieldID, req.GetId()))
		},
	)

	return err
}

func (r *NotificationChannelRepo) Delete(ctx context.Context, id uint32) error {
	if id == 0 {
		return adminV1.ErrorBadRequest("id is required")
	}

	if err := r.entClient.Client().NotificationChannel.DeleteOneID(id).Exec(ctx); err != nil {
		r.log.Errorf(ctx, "delete notification channel [%d] failed: %s", id, err.Error())
		return adminV1.ErrorInternalServerError("delete notification channel failed")
	}

	return nil
}

// SmtpAccount 发送邮件所需的解密后 SMTP 配置（仅服务层内部使用，禁止外传）。
//
// ID 是这条配置在 sys_notification_channels 里的主键：渠道自选后还要靠它把
// "实际用了哪个账号"写回投递台账。
type SmtpAccount struct {
	ID       uint32
	Host     string
	Port     uint32
	Username string
	Password string
	From     string
	TlsMode  string
	Enabled  bool
}

// GetFirstEnabledEmailChannel 取第一个启用的 EMAIL 渠道（找回密码/验证码发送用）。
func (r *NotificationChannelRepo) GetFirstEnabledEmailChannel(ctx context.Context) (*SmtpAccount, error) {
	entities, err := r.entClient.Client().NotificationChannel.Query().
		Where(
			notificationchannel.TypeEQ(notificationchannel.TypeEmail),
			notificationchannel.StatusEQ(notificationchannel.StatusOn),
		).
		Order(ent.Asc(notificationchannel.FieldID)).
		Limit(1).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query enabled email channel failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("query email channel failed")
	}
	if len(entities) == 0 {
		return nil, adminV1.ErrorNotFound("no enabled email channel")
	}
	e := entities[0]

	password := ""
	if e.SMTPPassword != nil {
		decrypted, decErr := crypto.DecryptIfNeeded(*e.SMTPPassword)
		if decErr != nil {
			r.log.Errorf(ctx, "decrypt smtp password failed for channel [%d]: %s", e.ID, decErr.Error())
			return nil, adminV1.ErrorInternalServerError("decrypt password failed")
		}
		password = decrypted
	}

	return &SmtpAccount{
		ID:       e.ID,
		Host:     derefStr(e.SMTPHost),
		Port:     derefUint32(e.SMTPPort),
		Username: derefStr(e.SMTPUsername),
		Password: password,
		From:     derefStr(e.SMTPFrom),
		// SMTPTLS 为可空列：此前 string(*e.SMTPTLS) 裸解引用，NULL 行会 panic，
		// 对齐同函数族其余字段的 nil 安全取值（derefStrP）。
		TlsMode: derefStrP(e.SMTPTLS),
		Enabled: true,
	}, nil
}

// GetDecryptedSmtpAccount 取渠道的解密 SMTP 配置。
func (r *NotificationChannelRepo) GetDecryptedSmtpAccount(ctx context.Context, id uint32) (*SmtpAccount, error) {
	entity, err := r.entClient.Client().NotificationChannel.Query().
		Where(notificationchannel.IDEQ(id)).
		Only(ctx)
	if err != nil {
		r.log.Errorf(ctx, "get notification channel [%d] failed: %s", id, err.Error())
		return nil, adminV1.ErrorNotFound("notification channel not found")
	}

	password := ""
	if entity.SMTPPassword != nil {
		decrypted, decErr := crypto.DecryptIfNeeded(*entity.SMTPPassword)
		if decErr != nil {
			r.log.Errorf(ctx, "decrypt smtp password failed for channel [%d]: %s", id, decErr.Error())
			return nil, adminV1.ErrorInternalServerError("decrypt password failed")
		}
		password = decrypted
	}

	return &SmtpAccount{
		ID:       entity.ID,
		Host:     derefStr(entity.SMTPHost),
		Port:     derefUint32(entity.SMTPPort),
		Username: derefStr(entity.SMTPUsername),
		Password: password,
		From:     derefStr(entity.SMTPFrom),
		TlsMode:  derefStrP(entity.SMTPTLS),
		Enabled:  entity.Status != nil && *entity.Status == notificationchannel.StatusOn,
	}, nil
}

// WebhookAccount 投递一条 WEBHOOK 通知所需的解密后配置（仅服务层内部使用，禁止外传）。
//
// 与 SmtpAccount 同形：ID 是"实际用了哪条渠道配置"这条事实的来源，
// URL 是回调地址，Secret 为空表示不签名（签名头不下发）。
type WebhookAccount struct {
	ID     uint32
	URL    string
	Secret string

	// SignStyle 风格列的原始值，不在 repo 里归一化：认不出的值由 sender 报错，
	// 比悄悄换成 CUSTOM 更可排障——台账里看得出自家库里此刻写着什么。
	SignStyle string

	// PayloadTemplate 载荷模板，空串表示"用该风格的内置默认形状"。
	PayloadTemplate string
}

// GetFirstEnabledWebhookChannel 取第一个启用的 WEBHOOK 渠道（事件路由未显式指定渠道时用）。
func (r *NotificationChannelRepo) GetFirstEnabledWebhookChannel(ctx context.Context) (*WebhookAccount, error) {
	entities, err := r.entClient.Client().NotificationChannel.Query().
		Where(
			notificationchannel.TypeEQ(notificationchannel.TypeWebhook),
			notificationchannel.StatusEQ(notificationchannel.StatusOn),
		).
		Order(ent.Asc(notificationchannel.FieldID)).
		Limit(1).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query enabled webhook channel failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("query webhook channel failed")
	}
	if len(entities) == 0 {
		return nil, adminV1.ErrorNotFound("no enabled webhook channel")
	}

	return r.webhookAccountOf(ctx, entities[0])
}

// GetDecryptedWebhookAccount 取指定渠道的解密 WEBHOOK 配置；类型不符即报错，
// 免得一条 SMTP 行被当 webhook 目标使（反过来由 EmailSender.resolveAccount 挡）。
func (r *NotificationChannelRepo) GetDecryptedWebhookAccount(ctx context.Context, id uint32) (*WebhookAccount, error) {
	entity, err := r.entClient.Client().NotificationChannel.Query().
		Where(notificationchannel.IDEQ(id)).
		Only(ctx)
	if err != nil {
		r.log.Errorf(ctx, "get notification channel [%d] failed: %s", id, err.Error())
		return nil, adminV1.ErrorNotFound("notification channel not found")
	}
	if entity.Type != notificationchannel.TypeWebhook {
		return nil, adminV1.ErrorBadRequest("channel [%d] is not a WEBHOOK channel", id)
	}

	return r.webhookAccountOf(ctx, entity)
}

// webhookAccountOf 实体 → WebhookAccount，密钥按 DecryptIfNeeded 解出明文。
//
// 解密失败上抛而不是降级成"不签名发出去"：对端配置了密钥却要收不到签名，
// 是让它把"验签失败"当成对端的故障，比这条投递直接失败更难排。
func (r *NotificationChannelRepo) webhookAccountOf(ctx context.Context, entity *ent.NotificationChannel) (*WebhookAccount, error) {
	secret := ""
	if entity.WebhookSecret != nil {
		decrypted, decErr := crypto.DecryptIfNeeded(*entity.WebhookSecret)
		if decErr != nil {
			r.log.Errorf(ctx, "decrypt webhook secret failed for channel [%d]: %s", entity.ID, decErr.Error())
			return nil, adminV1.ErrorInternalServerError("decrypt webhook secret failed")
		}
		secret = decrypted
	}

	return &WebhookAccount{
		ID:     entity.ID,
		URL:    derefStr(entity.WebhookURL),
		Secret: secret,

		// 两列都是 Optional().Nillable()：ent 迁移只补 DEFAULT 给新建行，存量行是 NULL，
		// 所以这里必须能容忍空值（风格空 → sender 按 CUSTOM 走，模板空 → 内置默认形状）。
		SignStyle:       derefStrP(entity.WebhookSignStyle),
		PayloadTemplate: derefStrP(entity.WebhookPayloadTemplate),
	}, nil
}

// statusFromEnabled DTO enabled 布尔 → ent status 枚举
func statusFromEnabled(enabled bool) notificationchannel.Status {
	if enabled {
		return notificationchannel.StatusOn
	}
	return notificationchannel.StatusOff
}

// statusFromProto DTO optional enabled → ent status 枚举指针（nil 表示不修改）
func (r *NotificationChannelRepo) statusFromProto(enabled *bool) *notificationchannel.Status {
	if enabled == nil {
		return nil
	}
	s := statusFromEnabled(*enabled)
	return &s
}
