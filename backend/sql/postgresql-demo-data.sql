BEGIN;

SET LOCAL search_path = public, pg_catalog;

-- 一次性清理相关表并重置自增（包含外键依赖）
TRUNCATE TABLE public.sys_org_units,
               public.sys_positions,
               public.sys_tasks,
               public.sys_login_policies,
               public.sys_dict_types,
               public.sys_dict_entries,
               public.internal_message_categories,
               public.sys_plan_modules,
               public.sys_plan_quotas,
               public.sys_plans,
               public.sys_notification_channels,
               public.sys_scripts,
               public.sys_access_keys,
               public.internal_messages,
               public.internal_message_recipients,
               public.sys_memberships,
               public.sys_membership_roles
RESTART IDENTITY CASCADE;

-- 测试租户
INSERT INTO public.sys_tenants(id, name, code, type, audit_status, status, admin_user_id, created_at)
VALUES (1, '测试租户', 'super', 'PAID', 'APPROVED', 'ON', 2, now())
;
SELECT setval('sys_tenants_id_seq', (SELECT MAX(id) FROM sys_tenants));

-- 插入租户管理员用户（sys_users 不在上方 TRUNCATE 清单——需保留平台管理员 admin，
-- 故以 ON CONFLICT 幂等插入，重复执行不冲突）
INSERT INTO public.sys_users (id, tenant_id, username, nickname, realname, email, gender, created_at)
VALUES
    -- 2. 租户管理员（TENANT_ADMIN）
    (2, 1, 'tenant_admin', '租户管理', '张管理员', 'tenant@company.com', 'MALE', now())
ON CONFLICT (id) DO NOTHING
;
SELECT setval('sys_users_id_seq', (SELECT MAX(id) FROM sys_users));

-- 插入4个用户的凭证（密码统一为admin，哈希值与原admin一致，方便测试）；
-- NOT EXISTS 守卫：user_id=2 已有凭证时整组跳过，保证脚本可重复执行
INSERT INTO public.sys_user_credentials (tenant_id, user_id, identity_type, identifier, credential_type, credential, status,
                                         is_primary, created_at)
SELECT * FROM (VALUES
    -- 租户管理员（对应users表id=2，tenant_id=1）
    (1, 2, 'USERNAME', 'tenant_admin', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, now()),
    (1, 2, 'EMAIL', 'tenant@company.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, now())
) AS seed(tenant_id, user_id, identity_type, identifier, credential_type, credential, status, is_primary, created_at)
WHERE NOT EXISTS (SELECT 1 FROM public.sys_user_credentials WHERE user_id = 2)
;
SELECT setval('sys_user_credentials_id_seq', (SELECT MAX(id) FROM sys_user_credentials));

-- 组织架构单元
INSERT INTO public.sys_org_units (id, tenant_id, parent_id, type, name, code, description, path, sort_order, leader_id, status, created_at)
VALUES
    (1, 1, NULL, 'COMPANY', 'XX集团总部', 'HEADQUARTERS', '集团核心管理机构，统筹全集团战略规划、业务管控及资源调配', '/1', 1, 1, 'ON', now()),
    (2, 1, 1, 'DIVISION', '技术部', 'TECH', '负责集团整体技术架构规划、研发管理、系统运维及技术创新', '/1/2', 2, 5, 'ON', now()),
    (3, 1, 1, 'DIVISION', '财务部', 'FIN', '负责集团财务核算、资金管理、税务筹划、预算编制及财务风控', '/1/3', 3, 8, 'ON', now()),
    (4, 1, 1, 'DIVISION', '人事部', 'HR', '负责人力资源规划、招聘配置、薪酬绩效、员工培训及组织发展', '/1/4', 4, 9, 'ON', now()),
    (5, 1, 2, 'DEPARTMENT', '研发一部', 'DEV-1', '聚焦新能源领域产品研发、技术迭代及核心模块开发', '/1/2/5', 1, 6, 'ON', now()),
    (6, 1, 1, 'REGION', '华北大区', 'NORTH', '负责华北区域市场运营、客户维护、销售管理及本地化服务落地', '/1/6', 3, 12, 'ON', now()),
    (7, 1, 1, 'SUBSIDIARY', '广州分公司', 'GZ', '负责华南区域（广州及周边）业务拓展、客户服务及本地化运营', '/1/7', 5, 2, 'ON', now()),
    (8, 1, 1, 'SUBSIDIARY', '深圳子公司', 'SZ', '负责深圳区域市场开拓、科技创新业务落地及高端客户对接', '/1/8', 6, 4, 'ON', now()),
    (9, 1, 1, 'DIVISION', '销售部', 'SALES', '统筹集团整体销售策略制定、销售团队管理及业绩目标达成', '/1/9', 7, 16, 'ON', now()),
    (10, 1, 9, 'DEPARTMENT', '海外事业部', 'INTL', '负责海外市场拓展、国际客户合作、跨境业务管理及本地化运营', '/1/9/10', 1, 17, 'ON', now()),
    (11, 1, 10, 'TEAM', '海外销售组', 'INTL-SALES-1', '具体执行海外市场销售任务，跟进客户需求及订单落地', '/1/9/10/11', 1, 18, 'ON', now()),
    (12, 1, 5, 'PROJECT', '新能源项目组', 'NEO-PROJ', '专项负责新能源项目的研发、落地、运营及成果转化', '/1/2/5/12', 1, 6, 'ON', now()),
    (13, 1, 1, 'COMMITTEE', '审计委员会', 'AUDIT', '独立开展集团内部审计、风控检查、合规监督及问题整改跟进', '/1/13', 8, 12, 'ON', now()),
    (14, 1, 1, 'DEPARTMENT', '客服部', 'CS', '负责全集团客户咨询、投诉处理、售后服务及客户满意度提升', '/1/14', 9, 11, 'ON', now()),
    (15, 1, 14, 'TEAM', '客服一组', 'CS-1', '承接华南区域客户服务、售后问题处理及客户关系维护', '/1/14/15', 1, 20, 'ON', now())
;
SELECT setval('sys_org_units_id_seq', (SELECT COALESCE(MAX(id), 1) FROM sys_org_units));

-- 岗位数据
INSERT INTO public.sys_positions (id, tenant_id, type, name, code, org_unit_id, reports_to_position_id, description, job_family, job_grade, level, headcount, is_key_position, status, sort_order, created_at)
VALUES
    (1, 1, 'LEADER', '技术总监', 'TECH-DIRECTOR-001', 2, NULL, '负责公司整体技术战略规划、团队管理及核心技术决策', 'TECH', 1, 1, 1, true, 'ON', 1, now()),
    (2, 1, 'MANAGER', '技术部经理', 'TECH-MANAGER-001', 2, 1, '负责技术部日常管理、项目排期及团队协作', 'TECH', 2, 2, 1, true, 'ON', 2, now()),
    (3, 1, 'MANAGER', '前端主管', 'TECH-FE-LEADER-001', 2, 2, '负责前端团队开发管理、技术方案评审及需求落地', 'TECH', 3, 3, 3, false, 'ON', 3, now()),
    (4, 1, 'MANAGER', '后端主管', 'TECH-BE-LEADER-001', 2, 2, '负责后端服务架构设计、数据库优化及接口开发管理', 'TECH', 4, 3, 3, false, 'ON', 4, now()),
    (5, 1, 'REGULAR', '前端开发专员', 'TECH-FE-DEV-001', 2, 3, '负责Web/移动端前端页面开发、交互实现及兼容性优化', 'TECH', 5, 4, 5, false, 'ON', 5, now()),
    (6, 1, 'REGULAR', '后端开发专员', 'TECH-BE-DEV-001', 2, 4, '负责后端接口开发、业务逻辑实现及系统稳定性维护', 'TECH', 6, 4, 5, false, 'ON', 6, now()),
    (7, 1, 'REGULAR', '测试工程师', 'TECH-TEST-001', 2, 2, '负责项目功能测试、性能测试及自动化测试脚本开发', 'TECH', 3, 4, 3, false, 'ON', 7, now()),
    (8, 1, 'LEADER', '人力总监', 'HR-DIRECTOR-001', 2, NULL, '负责人力资源战略规划、组织架构设计及人才梯队建设', 'HR', 1, 1, 1, true, 'ON', 1, now()),
    (9, 1, 'MANAGER', '招聘主管', 'HR-RECRUIT-LEADER-001', 2, 8, '负责公司各部门招聘需求对接、简历筛选及面试安排', 'HR', 2, 2, 1, false, 'ON', 2, now()),
    (10, 1, 'REGULAR', '薪酬绩效专员', 'HR-C&P-001', 2, 8, '负责员工薪酬核算、绩效考核制度落地及社保公积金管理', 'HR', 3, 2, 1, false, 'ON', 3, now()),
    (11, 1, 'REGULAR', 'HRBP', 'HR-BP-001', 2, 8, '对接业务部门，提供人力资源支持（入离职、员工关系等）', 'HR', 4, 2, 1, false, 'ON', 4, now()),
    (12, 1, 'LEADER', '财务总监', 'FIN-DIRECTOR-001', 2, NULL, '负责公司财务战略、预算管理及财务风险控制', 'FIN', 1, 1, 1, true, 'ON', 1, now()),
    (13, 1, 'MANAGER', '会计主管', 'FIN-ACCOUNT-LEADER-001', 2, 12, '负责账务处理、财务报表编制及税务申报管理', 'FIN', 2, 2, 1, false, 'ON', 2, now()),
    (14, 1, 'REGULAR', '出纳专员', 'FIN-CASHIER-001', 2, 13, '负责日常资金收付、银行对账及票据管理', 'FIN', 3, 3, 1, false, 'ON', 3, now()),
    (15, 1, 'REGULAR', '成本会计', 'FIN-COST-001', 2, 13, '负责成本核算、成本分析及成本控制方案制定', 'FIN', 4, 3, 1, false, 'ON', 4, now()),
    (16, 1, 'LEADER', '市场总监', 'MKT-DIRECTOR-001', 4, NULL, '负责市场战略规划、品牌建设及营销活动策划', 'MKT', 1, 1, 1, true, 'ON', 1, now()),
    (17, 1, 'MANAGER', '新媒体运营主管', 'MKT-NEWS-LEADER-001', 4, 16, '负责新媒体平台内容运营及用户增长', 'MKT', 2, 2, 1, false, 'ON', 2, now()),
    (18, 1, 'REGULAR', '活动策划专员', 'MKT-EVENT-001', 4, 16, '负责线下活动策划、执行及效果复盘', 'MKT', 3, 3, 1, false, 'ON', 3, now()),
    (19, 1, 'REGULAR', '市场调研专员', 'MKT-RESEARCH-001', 4, 16, '负责行业动态调研、竞品分析及市场趋势报告撰写', 'MKT', 4, 3, 1, false, 'ON', 4, now()),
    (20, 1, 'REGULAR', '行政助理', 'ADMIN-ASSIST-001', 2, 8, '负责办公用品采购、会议安排等行政工作（已合并至HRBP）', 'ADMIN', 5, 5, 1, false, 'OFF', 5, now())
;
SELECT setval('sys_positions_id_seq', (SELECT COALESCE(MAX(id), 1) FROM sys_positions));

-- 用户-租户关联关系
INSERT INTO public.sys_memberships (id, tenant_id, user_id, org_unit_id, position_id, role_id, is_primary, status)
VALUES
    -- 租户管理员（TENANT_ADMIN）
    (2, 1, 2, null, null, 2, true, 'ACTIVE')
;
SELECT setval('sys_memberships_id_seq', (SELECT MAX(id) FROM sys_memberships));

-- 租户成员-角色关联关系
INSERT INTO sys_membership_roles (id, membership_id, tenant_id, role_id, is_primary, status)
VALUES
    -- 租户管理员（TENANT_ADMIN）
    (2, 2, 1, 2, true, 'ACTIVE')
;
SELECT setval('sys_membership_roles_id_seq', (SELECT MAX(id) FROM sys_membership_roles));

-- 调度任务
INSERT INTO public.sys_tasks(type, type_name, task_payload, cron_spec, enable, created_at)
VALUES
    ('PERIODIC', 'backup', '{ "name": "test"}', '0 * * * *', true, now())
;
SELECT setval('sys_tasks_id_seq', (SELECT MAX(id) FROM sys_tasks));

-- 登录策略
INSERT INTO public.sys_login_policies(id, target_id, type, method, value, reason, created_at)
VALUES
    (1, 1, 'BLACKLIST', 'IP', '127.0.0.1', '无理由', now()),
    (2, 1, 'WHITELIST', 'MAC', '00:1B:44:11:3A:B7 ', '无理由', now())
;
SELECT setval('sys_login_policies_id_seq', (SELECT MAX(id) FROM sys_login_policies));

-- 插入字典类型
INSERT INTO public.sys_dict_types (
    id, type_code, type_name, sort_order, is_enabled, created_at, updated_at
) VALUES
      (1, 'USER_STATUS', '用户状态', 10, true, now(), now()),
      (2, 'DEVICE_TYPE', '设备类型', 20, true, now(), now()),
      (3, 'ORDER_STATUS', '订单状态', 30, true, now(), now()),
      (4, 'GENDER', '性别', 40, true, now(), now()),
      (5, 'PAYMENT_METHOD', '支付方式', 50, true, now(), now())
;
SELECT setval('sys_dict_types_id_seq', (SELECT MAX(id) FROM sys_dict_types));

-- 插入字典条目
INSERT INTO public.sys_dict_entries (
    id, type_id, entry_value, numeric_value, sort_order, is_enabled, created_at, updated_at, tenant_id
) VALUES
      -- 用户状态
      (1, 1, 'NORMAL', 1, 1, true, now(), now(), 0),
      (2, 1, 'FROZEN', 2, 2, true, now(), now(), 0),
      (3, 1, 'CANCELED', 3, 3, true, now(), now(), 0),
      -- 设备类型
      (4, 2, 'TEMP_SENSOR', 101, 1, true, now(), now(), 0),
      (5, 2, 'CURRENT_METER', 102, 2, true, now(), now(), 0),
      (6, 2, 'GAS_DETECTOR', 103, 3, false, now(), now(), 0),
      -- 订单状态
      (7, 3, 'PENDING', 1, 1, true, now(), now(), 0),
      (8, 3, 'PAID', 2, 2, true, now(), now(), 0),
      (9, 3, 'SHIPPED', 3, 3, true, now(), now(), 0),
      (10, 3, 'COMPLETED', 4, 4, true, now(), now(), 0),
      (11, 3, 'CANCELED', 5, 5, true, now(), now(), 0),
      -- 性别
      (12, 4, 'MALE', 1, 1, true, now(), now(), 0),
      (13, 4, 'FEMALE', 2, 2, true, now(), now(), 0),
      (14, 4, 'UNKNOWN', 0, 3, true, now(), now(), 0),
      -- 支付方式
      (15, 5, 'ALIPAY', 1, 1, true, now(), now(), 0),
      (16, 5, 'WECHAT', 2, 2, true, now(), now(), 0),
      (17, 5, 'UNIONPAY', 3, 3, true, now(), now(), 0),
      (18, 5, 'CASH', 4, 4, false, now(), now(), 0)
;
SELECT setval('sys_dict_entries_id_seq', (SELECT MAX(id) FROM sys_dict_entries));

-- 插入字典条目国际化（zh-CN）
INSERT INTO public.sys_dict_entry_i18n (
    entry_id, language_code, entry_label, description, sort_order, tenant_id, created_at, updated_at
) VALUES
      -- 用户状态
      (1, 'zh-CN', '正常', '用户可正常登录和操作', 1, 0, now(), now()),
      (2, 'zh-CN', '冻结', '因违规被临时冻结，需管理员解冻', 2, 0, now(), now()),
      (3, 'zh-CN', '注销', '用户主动注销，数据保留但不可登录', 3, 0, now(), now()),
      -- 设备类型
      (4, 'zh-CN', '温湿度传感器', '支持温度（-20~80℃）和湿度（0~100%RH）采集', 1, 0, now(), now()),
      (5, 'zh-CN', '电流仪表', '交流/直流电流测量，精度0.5级', 2, 0, now(), now()),
      (6, 'zh-CN', '气体探测器', '暂不支持，待硬件适配（2025Q4计划启用）', 3, 0, now(), now()),
      -- 订单状态
      (7, 'zh-CN', '待支付', '下单后未支付，超时自动取消', 1, 0, now(), now()),
      (8, 'zh-CN', '已支付', '支付成功，等待发货', 2, 0, now(), now()),
      (9, 'zh-CN', '已发货', '商品已出库，物流配送中', 3, 0, now(), now()),
      (10, 'zh-CN', '已完成', '用户确认收货，订单结束', 4, 0, now(), now()),
      (11, 'zh-CN', '已取消', '用户或系统取消订单', 5, 0, now(), now()),
      -- 性别
      (12, 'zh-CN', '男', '', 1, 0, now(), now()),
      (13, 'zh-CN', '女', '', 2, 0, now(), now()),
      (14, 'zh-CN', '未知', '用户未填写时默认值', 3, 0, now(), now()),
      -- 支付方式
      (15, 'zh-CN', '支付宝', '支持花呗、余额宝', 1, 0, now(), now()),
      (16, 'zh-CN', '微信支付', '需绑定微信', 2, 0, now(), now()),
      (17, 'zh-CN', '银联支付', '支持信用卡、储蓄卡', 3, 0, now(), now()),
      (18, 'zh-CN', '现金支付', '线下支付，已废弃（2025-01停用）', 4, 0, now(), now()),

      -- User Status
      (1, 'en-US', 'Normal', 'User can log in and operate normally', 1, 0, now(), now()),
      (2, 'en-US', 'Frozen', 'Temporarily frozen due to violation; requires admin to unfreeze', 2, 0, now(), now()),
      (3, 'en-US', 'Canceled', 'User voluntarily canceled; data retained but login disabled', 3, 0, now(), now()),

      -- Device Type
      (4, 'en-US', 'Temperature & Humidity Sensor', 'Supports temperature (-20~80°C) and humidity (0~100% RH) measurement', 1, 0, now(), now()),
      (5, 'en-US', 'Current Meter', 'Measures AC/DC current with 0.5-class accuracy', 2, 0, now(), now()),
      (6, 'en-US', 'Gas Detector', 'Not supported yet; hardware integration planned for Q4 2025', 3, 0, now(), now()),

      -- Order Status
      (7, 'en-US', 'Pending Payment', 'Order placed but not paid; auto-canceled if timeout', 1, 0, now(), now()),
      (8, 'en-US', 'Paid', 'Payment successful; awaiting shipment', 2, 0, now(), now()),
      (9, 'en-US', 'Shipped', 'Item has left warehouse; in transit', 3, 0, now(), now()),
      (10, 'en-US', 'Completed', 'User confirmed receipt; order closed', 4, 0, now(), now()),
      (11, 'en-US', 'Canceled', 'Order canceled by user or system', 5, 0, now(), now()),

      -- Gender
      (12, 'en-US', 'Male', '', 1, 0, now(), now()),
      (13, 'en-US', 'Female', '', 2, 0, now(), now()),
      (14, 'en-US', 'Unknown', 'Default value when user does not specify', 3, 0, now(), now()),

      -- Payment Method
      (15, 'en-US', 'Alipay', 'Supports Huabei and Yu’ebao', 1, 0, now(), now()),
      (16, 'en-US', 'WeChat Pay', 'Requires WeChat account binding', 2, 0, now(), now()),
      (17, 'en-US', 'UnionPay', 'Supports credit and debit cards', 3, 0, now(), now()),
      (18, 'en-US', 'Cash', 'Offline payment; deprecated as of Jan 2025', 4, 0, now(), now())
;
SELECT setval('sys_dict_entry_i18n_id_seq', (SELECT MAX(id) FROM sys_dict_entry_i18n));

-- 站内信分类
INSERT INTO public.internal_message_categories (id, code, name, remark, sort_order, is_enabled, created_at)
VALUES
    -- 订单相关分类（原主分类+子分类平级展示）
    (1, 'order', '订单通知', '包含订单支付、发货、退款等全流程通知', 1, true, NOW()),
    (101, 'order_paid', '支付成功', '订单支付完成时触发的通知', 2, true, NOW()),
    (102, 'order_unpaid', '支付超时', '订单未在规定时间内支付的提醒', 3, true, NOW()),
    (103, 'order_shipped', '已发货', '商家发货后通知用户', 4, true, NOW()),
    (104, 'order_refunded', '已退款', '订单退款流程完成的通知', 5, true, NOW()),

    -- 系统相关分类
    (2, 'system', '系统通知', '系统公告、维护提醒、版本更新等平台级通知', 6, true, NOW()),
    (201, 'system_announcement', '系统公告', '平台规则更新、重要通知等', 7, true, NOW()),
    (202, 'system_maintenance', '维护通知', '系统计划内维护的时间提醒', 8, true, NOW()),
    (203, 'system_upgrade', '版本更新', '客户端或功能升级的提示', 9, true, NOW()),

    -- 活动相关分类
    (3, 'activity', '活动通知', '营销活动报名、开始、结束等提醒', 10, true, NOW()),
    (301, 'activity_signup', '报名成功', '用户报名活动后确认通知', 11, true, NOW()),
    (302, 'activity_start', '活动开始', '活动即将开始的倒计时提醒', 12, true, NOW()),
    (303, 'activity_end', '活动结束', '活动结束及结果公示通知', 13, true, NOW()),

    -- 用户相关分类
    (4, 'user', '用户通知', '账号安全、信息变更、权限调整等个人相关通知', 14, true, NOW()),
    (401, 'user_login_abnormal', '异地登录', '账号在陌生设备登录的安全提醒', 15, true, NOW()),
    (402, 'user_profile_updated', '资料变更', '用户手机号、邮箱等信息修改后通知', 16, true, NOW()),
    (403, 'user_permission_changed', '权限变更', '账号角色或功能权限调整通知', 17, true, NOW())
;
SELECT setval('internal_message_categories_id_seq', (SELECT MAX(id) FROM internal_message_categories));

-- 套餐目录种子（三版本：免费/标准/企业）
INSERT INTO public.sys_plans (id, name, version, expiry_policy, data_retention_days, description, created_at)
VALUES
    (1, '免费版', 'FREE', 'READONLY', 0, '免费版套餐，仅基础功能', now()),
    (2, '标准版', 'STANDARD', 'BLOCK_LOGIN', 30, '标准版套餐，含核心业务模块', now()),
    (3, '企业版', 'ENTERPRISE', 'FREEZE', 90, '企业版套餐，全功能模块', now())
;
SELECT setval('sys_plans_id_seq', (SELECT MAX(id) FROM sys_plans));

-- 套餐配额种子
INSERT INTO public.sys_plan_quotas (plan_id, quota_type, quota_value, created_at)
VALUES
    (1, 'USER_LIMIT', 5, now()),
    (1, 'STORAGE', 1073741824, now()),
    (1, 'API_CALL', 1000, now()),
    (2, 'USER_LIMIT', 50, now()),
    (2, 'STORAGE', 10737418240, now()),
    (2, 'API_CALL', 10000, now()),
    (3, 'USER_LIMIT', 500, now()),
    (3, 'STORAGE', 107374182400, now()),
    (3, 'API_CALL', 100000, now())
;

-- 套餐模块白名单种子
-- 免费版（plan_id=1）：DASHBOARD + TENANT
INSERT INTO public.sys_plan_modules (plan_id, module, created_at)
VALUES
    (1, 'DASHBOARD', now()),
    (1, 'TENANT', now())
;

-- 标准版（plan_id=2）：DASHBOARD + OPM + SYSTEM + DICT + LOG + TENANT
INSERT INTO public.sys_plan_modules (plan_id, module, created_at)
VALUES
    (2, 'DASHBOARD', now()),
    (2, 'OPM', now()),
    (2, 'SYSTEM', now()),
    (2, 'DICT', now()),
    (2, 'LOG', now()),
    (2, 'TENANT', now())
;

-- 企业版（plan_id=3）：全模块
INSERT INTO public.sys_plan_modules (plan_id, module, created_at)
VALUES
    (3, 'DASHBOARD', now()),
    (3, 'OPM', now()),
    (3, 'SYSTEM', now()),
    (3, 'DICT', now()),
    (3, 'TENANT', now()),
    (3, 'PERMISSION', now()),
    (3, 'LOG', now()),
    (3, 'INTERNAL_MESSAGE', now()),
    (3, 'FILE', now()),
    (3, 'TASK', now())
;

-- ============================================================
-- 增补：通知渠道 / 脚本 / 访问密钥 / 站内信（含收件人）
-- （tenant_admin 的登录凭证已由上方 sys_user_credentials 段 provisioning）
-- ============================================================

-- 通知渠道（类型 tag：EMAIL；状态 tag：ON/OFF）
-- smtp_tls 值域只有 NONE / START_TLS / SSL（ent 枚举 + mailer 拨号 switch 同一份表），
-- 写宽了不会被库拦下（列是 varchar），但投递时会被判成"配置不可用"→ 台账 SKIPPED。
INSERT INTO public.sys_notification_channels (status, name, type, smtp_host, smtp_port, smtp_username, smtp_password, smtp_from, smtp_tls, remark) VALUES
    ('ON', '运维告警邮箱', 'EMAIL', 'smtp.example.com', 465, 'ops@example.com', 'demo-pass-1', 'ops@example.com', 'SSL', '生产告警主通道'),
    ('ON', '市场活动通知', 'EMAIL', 'smtp.example.com', 587, 'mkt@example.com', 'demo-pass-2', 'mkt@example.com', 'START_TLS', '市场推广通知'),
    ('OFF', '备用邮箱通道', 'EMAIL', 'smtp.backup.com', 25, 'bak@example.com', 'demo-pass-3', 'bak@example.com', 'NONE', '灾备备用，停用中')
;

-- 脚本（语言 tag：LUA/JAVASCRIPT；启用状态；关键脚本 tag）
INSERT INTO public.sys_scripts (is_enabled, name, language, hook_point, source, priority, description, critical) VALUES
    (true, '租户创建审计钩子', 'LUA', 'entity.after_create', 'function on_after_create(ctx) log("tenant created") end', 10, '租户创建后写审计日志', true),
    (true, '用户敏感字段脱敏', 'JAVASCRIPT', 'entity.after_query', 'function afterQuery(ctx) { mask(ctx.user.mobile); }', 5, '查询返回前脱敏手机号', false),
    (false, '订单校验规则（停用）', 'LUA', 'entity.before_update', 'function before_update(ctx) end', 0, '旧版校验规则，已停用', false)
;

-- 访问密钥（状态 tag：ON/OFF；含过期时间演示）
INSERT INTO public.sys_access_keys (status, tenant_id, name, access_key, secret_hash, expires_at) VALUES
    ('ON', 0, '监控平台接入', 'AK-demo-monitor-001', 'demo-hash-monitor', now() + interval '365 days'),
    ('ON', 1, '测试租户数据同步', 'AK-demo-tenant-sync', 'demo-hash-sync', now() + interval '90 days'),
    ('OFF', 0, '已停用的CI密钥', 'AK-demo-ci-old', 'demo-hash-ci', now() - interval '30 days')
;

-- 站内信（状态 tag：DRAFT/PUBLISHED/SCHEDULED/ARCHIVED；类型 tag：NOTIFICATION）
INSERT INTO public.internal_messages (tenant_id, title, content, sender_id, category_id, status, type) VALUES
    (0, '平台维护公告', '本周六 02:00-04:00 平台例行维护，期间服务短暂不可用。', 1,
        (SELECT id FROM public.internal_message_categories ORDER BY id LIMIT 1), 'PUBLISHED', 'NOTIFICATION'),
    (1, '租户版本更新说明', '新版工作台已上线，详情见帮助中心。', 1,
        (SELECT id FROM public.internal_message_categories ORDER BY id OFFSET 1 LIMIT 1), 'PUBLISHED', 'NOTIFICATION'),
    (0, '促销活动草稿', '双十一活动方案（草稿，待审批）。', 1, NULL, 'DRAFT', 'NOTIFICATION'),
    (0, '下月功能预告', '定时发布：下月 1 日 09:00 自动推送。', 1, NULL, 'SCHEDULED', 'NOTIFICATION'),
    (0, '已归档的旧公告', '2025 年度旧公告，已归档。', 1, NULL, 'ARCHIVED', 'NOTIFICATION')
;

-- 站内信收件人（收件箱列表：状态 tag RECEIVED/READ；仅已发出的消息有收件人）
INSERT INTO public.internal_message_recipients (tenant_id, message_id, recipient_user_id, status, received_at, read_at) VALUES
    (0, (SELECT id FROM public.internal_messages WHERE title = '平台维护公告'), 1, 'READ', now(), now()),
    (0, (SELECT id FROM public.internal_messages WHERE title = '已归档的旧公告'), 1, 'RECEIVED', now(), NULL),
    (1, (SELECT id FROM public.internal_messages WHERE title = '租户版本更新说明'), 2, 'READ', now(), now())
;

COMMIT;
