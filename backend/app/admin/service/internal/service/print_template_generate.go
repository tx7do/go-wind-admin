package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"github.com/tx7do/go-utils/trans"

	"go-wind-admin/app/admin/service/internal/data/ent"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	printV1 "go-wind-admin/api/gen/go/print/service/v1"
	"go-wind-admin/pkg/middleware/auth"
)

// —— AI 生成模板（M4-1）的护栏：LLM 产出的 TemplateData 必须过本文件校验
// 才允许返回给前端（前端预览确认后再走既有 Create 保存）。
// 原则与问数护栏一致：fail-closed，发现不可接受的形态直接拒绝而不是静默改写；
// 但几何/长度类"钳制后语义不变"的项按上限收敛（与文档 §7 的"schema 校验 +
// 字段白名单校验后方可保存"对应）。

const (
	// printGenMaxPayload AI 生成 payload 的字节数上限。生成面远小于渲染面
	// （printRenderMaxHTML 8MiB）——一句话描述产不出几十万字段的合法模板，
	// 超限即模型失控或提示注入，直接拒绝。
	printGenMaxPayload = 512 << 10

	// printGenMaxElements 单个模板元素数上限（真实单据几十个元素封顶）。
	printGenMaxElements = 200

	// printGenMaxFieldRefs 单模板 {field} 绑定引用数上限。
	printGenMaxFieldRefs = 300

	// printGenMaxJSONDepth JSON 嵌套深度上限（TemplateData 合法形态最深 4 层：
	// 根 → elements[i] → options → margin 数值）。
	printGenMaxJSONDepth = 8

	// printGenMaxText AI 生成文本类值的字符上限（formatter/名称/说明）。
	printGenMaxText = 2000

	// printGenMaxMM 几何数值的毫米上限（A3 横向约 420mm，放大余量防溢出画布）。
	printGenMaxMM = 2000

	// printGenMaxMargin 页边距钳制上限（毫米）。
	printGenMaxMargin = 50
)

// printTemplateAllowedTypes AI 生成元素类型白名单（hiprint 内核家族的常规类型，
// 从严取用）：未知类型一律拒绝（fail-closed）——设计器里用户仍可手动加，
// 生成面不提供任意元素类型的攻击面。html 类不在列：任意 HTML 走
// sanitizePrintHTML 那条链更合适，AI 生成不开这个口子。
var printTemplateAllowedTypes = map[string]bool{
	"text":     true,
	"longtext": true,
	"image":    true,
	"barcode":  true,
	"qrcode":   true,
	"table":    true,
	"rect":     true,
	"oval":     true,
	"line":     true,
	"customize": true,
}

var printTemplateAllowedPaper = map[string]bool{
	"A3": true, "A4": true, "A5": true, "B4": true, "B5": true,
}

// printTemplateAllowedTopKeys TemplateData 顶层键白名单（对齐种子模板与
// 内核 generateHtml 消费的形态）。生成中出现其他键 → 拒绝（不静默丢弃，
// 避免模型塞入内核未来才消费的键时行为不可预期）。
var printTemplateAllowedTopKeys = map[string]bool{
	"unit": true, "paperSize": true, "orientation": true, "schemaVersion": true,
	"margins": true, "header": true, "footer": true, "watermark": true,
	"guides": true, "name": true, "elements": true,
	// 面板容器：种子模板固定 header/footer 两个，但内核允许 panelCount>1 时
	// 的 panels 数组——白名单一并放行（存在与否由内核按版本处理）。
	"panels": true,
}

// printTemplateAllowedElemKeys 元素级键白名单（种子样例：id/type/options/printElementType）。
var printTemplateAllowedElemKeys = map[string]bool{
	"id": true, "type": true, "options": true, "printElementType": true,
	"tid": true, // hiprint 面板 tid 分类键，内核样例携带
}

// printFieldRefPattern {field} 绑定占位符的字段名格式：标识符（字母/下划线开头，
// 可含数字），不支持路径/表达式——表达式是设计器里的能力，AI 生成面不放行。
var printFieldRefPattern = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// printIdentifierPattern 字段名本体（同上，用于拒绝全数字等退化形态——正则已保证）。

// sanitizeGeneratedTemplate 校验 AI 生成的 TemplateData：
// 返回原样 JSON（仅做几何/文本钳制的重新序列化）与引用的 {field} 集合。
// 任何结构性越界（未知顶层键/未知元素类型/超深嵌套/超限）直接报错。
func sanitizeGeneratedTemplate(raw string) (string, []string, error) {
	if len(raw) > printGenMaxPayload {
		return "", nil, adminV1.ErrorBadRequest("generated template too large")
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil, adminV1.ErrorBadRequest("generated template is empty")
	}

	var root map[string]interface{}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		return "", nil, adminV1.ErrorBadRequest("generated template is not valid json")
	}
	if dec.More() {
		return "", nil, adminV1.ErrorBadRequest("generated template has trailing data")
	}

	// 顶层键白名单
	for k := range root {
		if !printTemplateAllowedTopKeys[k] {
			return "", nil, adminV1.ErrorBadRequest("generated template: unexpected top-level key %q", k)
		}
	}
	if err := checkJSONDepth(root, 1); err != nil {
		return "", nil, err
	}

	// 必填语义钳制：unit 固定毫米（内核按 mm 布局，其他单位直接拒）
	if u, _ := root["unit"].(string); u != "mm" {
		return "", nil, adminV1.ErrorBadRequest("generated template: unit must be \"mm\"")
	}
	// 纸张/方向：缺失时补默认（提示词要求过，但 LLM 漏字段是常态，补齐无害）
	paper := strOr(root["paperSize"], "A4")
	if !printTemplateAllowedPaper[paper] {
		return "", nil, adminV1.ErrorBadRequest("generated template: unsupported paperSize %q", paper)
	}
	root["paperSize"] = paper
	orient := strOr(root["orientation"], "portrait")
	if orient != "portrait" && orient != "landscape" {
		return "", nil, adminV1.ErrorBadRequest("generated template: orientation must be portrait/landscape")
	}
	root["orientation"] = orient
	root["schemaVersion"] = json.Number("1")

	if err := clampMargins(root); err != nil {
		return "", nil, err
	}
	for _, zone := range []string{"header", "footer"} {
		if z, ok := root[zone].(map[string]interface{}); ok {
			z["height"] = clampMM(z["height"], 0, printGenMaxMargin)
		}
	}
	if name, ok := root["name"].(string); ok && len([]rune(name)) > 200 {
		root["name"] = string([]rune(name)[:200])
	}

	// 元素：数量上限 + 逐项白名单 + 几何钳制 + 占位符收集
	fields := make(map[string]bool)
	elems, _ := root["elements"].([]interface{})
	if len(elems) > printGenMaxElements {
		return "", nil, adminV1.ErrorBadRequest("generated template: too many elements (%d > %d)", len(elems), printGenMaxElements)
	}
	for i, e := range elems {
		em, ok := e.(map[string]interface{})
		if !ok {
			return "", nil, adminV1.ErrorBadRequest("generated template: element %d is not an object", i)
		}
		for k := range em {
			if !printTemplateAllowedElemKeys[k] {
				return "", nil, adminV1.ErrorBadRequest("generated template: element %d has unexpected key %q", i, k)
			}
		}
		typ := strOr(em["type"], "")
		if !printTemplateAllowedTypes[typ] {
			return "", nil, adminV1.ErrorBadRequest("generated template: element %d has disallowed type %q", i, typ)
		}
		// id：字符串且非空（内核以 id 做元素寻址），上限收紧防滥用
		id, _ := em["id"].(string)
		if strings.TrimSpace(id) == "" {
			return "", nil, adminV1.ErrorBadRequest("generated template: element %d missing id", i)
		}
		if len(id) > 100 {
			em["id"] = id[:100]
		}
		opts, ok := em["options"].(map[string]interface{})
		if !ok {
			return "", nil, adminV1.ErrorBadRequest("generated template: element %d missing options", i)
		}
		for _, g := range []string{"left", "top", "width", "height"} {
			opts[g] = clampMM(opts[g], 0, printGenMaxMM)
		}
		if f, ok := opts["formatter"].(string); ok {
			if len([]rune(f)) > printGenMaxText {
				return "", nil, adminV1.ErrorBadRequest("generated template: element %d formatter too long", i)
			}
			for _, m := range printFieldRefPattern.FindAllStringSubmatch(f, -1) {
				fields[m[1]] = true
			}
		}
	}
	if len(fields) > printGenMaxFieldRefs {
		return "", nil, adminV1.ErrorBadRequest("generated template: too many field refs (%d > %d)", len(fields), printGenMaxFieldRefs)
	}

	clean, err := json.Marshal(root)
	if err != nil {
		return "", nil, adminV1.ErrorInternalServerError("re-serialize generated template failed")
	}

	fieldList := make([]string, 0, len(fields))
	for f := range fields {
		fieldList = append(fieldList, f)
	}
	// 稳定输出顺序，便于测试与前端展示
	for i := 0; i < len(fieldList); i++ {
		for j := i + 1; j < len(fieldList); j++ {
			if fieldList[j] < fieldList[i] {
				fieldList[i], fieldList[j] = fieldList[j], fieldList[i]
			}
		}
	}
	return string(clean), fieldList, nil
}

// checkJSONDepth 递归检查嵌套深度（map/slice 各算一层）。
func checkJSONDepth(v interface{}, depth int) error {
	if depth > printGenMaxJSONDepth {
		return adminV1.ErrorBadRequest("generated template: nesting too deep (> %d)", printGenMaxJSONDepth)
	}
	switch t := v.(type) {
	case map[string]interface{}:
		for _, vv := range t {
			if err := checkJSONDepth(vv, depth+1); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, vv := range t {
			if err := checkJSONDepth(vv, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

// clampMargins 校验并钳制 margins（缺省补 10；非数值/负值/超 50mm 拒绝或钳制）。
func clampMargins(root map[string]interface{}) error {
	m, ok := root["margins"].(map[string]interface{})
	if !ok {
		root["margins"] = map[string]interface{}{
			"top": json.Number("10"), "right": json.Number("10"),
			"bottom": json.Number("10"), "left": json.Number("10"),
		}
		return nil
	}
	for _, side := range []string{"top", "right", "bottom", "left"} {
		if _, present := m[side]; !present {
			m[side] = json.Number("10")
			continue
		}
		v, ok := m[side].(json.Number)
		if !ok {
			return adminV1.ErrorBadRequest("generated template: margins.%s is not a number", side)
		}
		f, err := v.Float64()
		if err != nil {
			return adminV1.ErrorBadRequest("generated template: margins.%s is not a valid number", side)
		}
		if f < 0 {
			m[side] = json.Number("0") // 负边距无物理意义，与元素几何同款钳 0
			continue
		}
		if f > printGenMaxMargin {
			m[side] = json.Number(fmt.Sprintf("%.2f", float64(printGenMaxMargin)))
		}
	}
	return nil
}

// clampMM 数值钳到 [min,max]；非数值返回 0（几何缺失由内核按默认处理，不因
// 模型漏字段而整体拒绝）。
func clampMM(v interface{}, min, max float64) json.Number {
	n, ok := v.(json.Number)
	if !ok {
		return json.Number(fmt.Sprintf("%.1f", min))
	}
	f, err := n.Float64()
	if err != nil || f < min {
		return json.Number(fmt.Sprintf("%.1f", min))
	}
	if f > max {
		return json.Number(fmt.Sprintf("%.1f", max))
	}
	return n
}

func strOr(v interface{}, def string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return def
}

// extractTemplatePayload 从模型回复中剥出 JSON 主体：容忍 ```json 围栏
// 与前后闲话（取第一个 { 到最后一个 } 之间的内容——TemplateData 是单对象，
// 不存在"正文里合法包含裸大括号对"的歧义）。
func extractTemplatePayload(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	start := strings.IndexByte(raw, '{')
	end := strings.LastIndexByte(raw, '}')
	if start < 0 || end <= start {
		return raw
	}
	return raw[start : end+1]
}

// printGenCallTimeout 生成调用的 LLM 往返超时：结构化 JSON 产出比问答慢，
// 与问数同量级取 90s。
const printGenCallTimeout = 90 * time.Second

// GenerateTemplate AI 一句话生成模板（M4-1，docs/print_template.md §7 第 1 条）：
// 描述单据要素 → LLM 产出 TemplateData → sanitizeGeneratedTemplate 护栏校验 →
// 返回（未入库，前端预览确认后走 Create 保存）。
// 复用问数链路的四件套：AI_TOKENS 月度配额 / provider 默认解析与客户端构造 /
// 非流式调用 / 用量落库。
func (s *PrintTemplateService) GenerateTemplate(ctx context.Context, req *printV1.GeneratePrintTemplateRequest) (*printV1.GeneratePrintTemplateResponse, error) {
	if req == nil || strings.TrimSpace(req.GetDescription()) == "" {
		return nil, adminV1.ErrorBadRequest("description is required")
	}
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err = checkAiTokenQuota(ctx, s.aiUsageRepo, s.log, operator); err != nil {
		return nil, err
	}

	provider, err := s.aiProviderRepo.GetEnabledDefault(ctx)
	if err != nil {
		return nil, err
	}
	client, err := newOpenAIClientForProvider(ctx, provider)
	if err != nil {
		return nil, err
	}

	raw, tokens, err := s.generateTemplatePayload(ctx, client, provider, req)
	if err != nil {
		return nil, err
	}

	clean, fields, err := sanitizeGeneratedTemplate(extractTemplatePayload(raw))
	if err != nil {
		// 模型产物被护栏拒绝：记原始输出（不吞错，便于调提示词），对调用方回 400
		s.log.Warnf(ctx, "print template generation rejected: %v, raw=%s", err, truncateForLog(raw))
		return nil, err
	}

	// 用量落库（同问数口径：失败不阻断返回，Errorf 留痕）
	if s.aiUsageRepo != nil && tokens > 0 {
		modelName := ptrStrOr(provider.ModelName, "gpt-4o-mini")
		totalTokens := uint32(tokens)
		if uerr := s.aiUsageRepo.Create(ctx, &aiV1.AiUsageLog{
			UserId:      trans.Ptr(operator.GetUserId()),
			TenantId:    trans.Ptr(operator.GetTenantId()),
			ModelName:   &modelName,
			TotalTokens: &totalTokens,
		}); uerr != nil {
			s.log.Errorf(ctx, "record print template generation usage failed: %v", uerr)
		}
	}

	return &printV1.GeneratePrintTemplateResponse{
		Payload:     clean,
		Fields:      fields,
		TotalTokens: uint32(tokens),
	}, nil
}

// generateTemplatePayload 提示词 + 单次非流式调用。提示词把 TemplateData 的
// 骨架与约束写死（单位毫米 / 元素类型白名单 / 几何以毫米计 / 文本绑定用
// {field} 占位符），模型越界由 sanitizeGeneratedTemplate 统一兜底拒绝。
func (s *PrintTemplateService) generateTemplatePayload(ctx context.Context, client *openai.Client, provider *ent.AiProvider, req *printV1.GeneratePrintTemplateRequest) (string, int, error) {
	var sb strings.Builder
	sb.WriteString("你是单据打印模板生成引擎，输出 hiprint 风格的 TemplateData JSON（毫米布局）。\n")
	sb.WriteString("输出规则：\n")
	sb.WriteString("1. 只输出一个 JSON 对象，不要解释、不要 Markdown 代码块；\n")
	sb.WriteString("2. 顶层键只用：unit(固定\"mm\")、paperSize、orientation、schemaVersion(固定 1)、margins、header、footer、watermark、guides、name、elements；\n")
	sb.WriteString("3. elements 元素只用这些 type：text、longtext、image、barcode、qrcode、table、rect、oval、line；\n")
	sb.WriteString("4. 每个元素必须有 id(唯一)、type、options；options 里 left/top/width/height 用毫米数值，文本内容放 options.formatter；\n")
	sb.WriteString("5. 数据绑定用 {fieldName} 占位符（如 出库单号：{orderNo}），不要输出表达式、函数或 HTML；\n")
	sb.WriteString("6. 表格元素用 type:\"table\"，列定义放 options.columns（每列 {title,field,width}）；\n")
	sb.WriteString("7. 布局要符合单据排版惯例：标题居上居中、字段行靠左、明细表在中部、合计/签收在底部。\n\n")
	sb.WriteString("用户对单据的描述：\n")
	sb.WriteString(req.GetDescription())
	sb.WriteString("\n\n")

	paper := req.GetPaperSize()
	if paper == "" {
		paper = "A4"
	}
	orient := req.GetOrientation()
	if orient == "" {
		orient = "portrait"
	}
	if rk := req.GetResourceKey(); rk != "" {
		sb.WriteString("（该模板将绑定业务资源 " + rk + "，占位符字段名尽量贴合该资源的常见字段语义。）\n\n")
	}

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: sb.String()},
		{Role: openai.ChatMessageRoleUser, Content: fmt.Sprintf("生成一张 %s %s 模板：%s", paper, orient, req.GetDescription())},
	}

	gCtx, cancel := context.WithTimeout(ctx, printGenCallTimeout)
	defer cancel()
	resp, err := client.CreateChatCompletion(gCtx, openai.ChatCompletionRequest{
		Model:       ptrStrOr(provider.ModelName, "gpt-4o-mini"),
		Messages:    messages,
		Temperature: 0.2,
	})
	if err != nil {
		return "", 0, adminV1.ErrorInternalServerError("ai generate template failed: %v", err)
	}
	if len(resp.Choices) == 0 {
		return "", int(resp.Usage.TotalTokens), adminV1.ErrorInternalServerError("ai generate template empty")
	}
	return resp.Choices[0].Message.Content, resp.Usage.TotalTokens, nil
}

// truncateForLog 日志用截断（原始模型输出可能很长）。
func truncateForLog(s string) string {
	const max = 2000
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}
