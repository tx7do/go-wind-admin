BEGIN;

SET LOCAL search_path = public, pg_catalog;

-- 一次性清理相关表并重置自增（包含外键依赖）
TRUNCATE TABLE public.sys_org_units,
               public.sys_positions,
               public.sys_tasks,
               public.sys_admin_login_restrictions,
               public.sys_dict_types,
               public.sys_dict_entries,
               public.internal_message_categories
RESTART IDENTITY CASCADE;

-- 租户
INSERT INTO public.sys_tenants(id, name, code, type, audit_status, status, admin_user_id, created_at)
VALUES (1, '测试租户', 'super', 'PAID', 'APPROVED', 'ON', 2, now())
;
SELECT setval('sys_tenants_id_seq', (SELECT MAX(id) FROM sys_tenants));

-- 默认的角色
INSERT INTO public.sys_roles(id, tenant_id, sort_order, name, code, status, description, created_at)
VALUES 
       (2, 0, 2, '安全审计员', 'security_auditor', 'ON', '仅可查看系统操作日志和数据记录，无修改权限', now()),
       (3, 0, 3, '租户管理员', 'tenant_admin', 'ON', '管理当前租户下的用户、角色及资源，无跨租户操作权限', now())
;
SELECT setval('sys_roles_id_seq', (SELECT MAX(id) FROM sys_roles));

-- 插入4个权限的用户
INSERT INTO public.sys_users (id, tenant_id, username, nickname, realname, email, gender, created_at)
VALUES
    -- 2. 租户管理员（TENANT_ADMIN）
    (2, 1, 'tenant_admin', '租户管理', '张管理员', 'tenant@company.com', 'MALE', now()),
    -- 3. 普通用户（USER）
    (3, 0, 'normal_user', '普通用户', '李用户', 'user@company.com', 'FEMALE', now()),
    -- 4. 访客（GUEST）
    (4, 0, 'guest_user', '临时访客', '王访客', 'guest@company.com', 'SECRET', now())
;
SELECT setval('sys_users_id_seq', (SELECT MAX(id) FROM sys_users));

-- 插入4个用户的凭证（密码统一为admin，哈希值与原admin一致，方便测试）
INSERT INTO public.sys_user_credentials (user_id, identity_type, identifier, credential_type, credential, status,
                                         is_primary, created_at)
VALUES
       -- 租户管理员（对应users表id=2）
       (2, 'USERNAME', 'tenant_admin', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a',
        'ENABLED', true, now()),
       (2, 'EMAIL', 'tenant@company.com', 'PASSWORD_HASH',
        '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, now()),

       -- 普通用户（对应users表id=3）
       (3, 'USERNAME', 'normal_user', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a',
        'ENABLED', true, now()),
       (3, 'EMAIL', 'user@company.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a',
        'ENABLED', false, now()),

       -- 访客（对应users表id=4）
       (4, 'USERNAME', 'guest_user', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a',
        'ENABLED', true, now()),
       (4, 'EMAIL', 'guest@company.com', 'PASSWORD_HASH',
        '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, now())
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

-- 岗位数据
INSERT INTO public.sys_positions (id, tenant_id, parent_id, type, name, code, org_unit_id, reports_to_position_id, description, job_family, job_grade, level, headcount, is_key_position, status, sort_order, created_at)
VALUES
    (1, 1, NULL, 'LEADER', '技术总监', 'TECH-DIRECTOR-001', 2, NULL, '负责公司整体技术战略规划、团队管理及核心技术决策', 'TECH', 1, 1, 1, true, 'ON', 1, now()),
    (2, 1, 1, 'MANAGER', '技术部经理', 'TECH-MANAGER-001', 2, 1, '负责技术部日常管理、项目排期及团队协作', 'TECH', 2, 2, 1, true, 'ON', 2, now()),
    (3, 1, 2, 'MANAGER', '前端主管', 'TECH-FE-LEADER-001', 2, 2, '负责前端团队开发管理、技术方案评审及需求落地', 'TECH', 3, 3, 3, false, 'ON', 3, now()),
    (4, 1, 2, 'MANAGER', '后端主管', 'TECH-BE-LEADER-001', 2, 2, '负责后端服务架构设计、数据库优化及接口开发管理', 'TECH', 4, 3, 3, false, 'ON', 4, now()),
    (5, 1, 3, 'REGULAR', '前端开发专员', 'TECH-FE-DEV-001', 2, 3, '负责Web/移动端前端页面开发、交互实现及兼容性优化', 'TECH', 5, 4, 5, false, 'ON', 5, now()),
    (6, 1, 4, 'REGULAR', '后端开发专员', 'TECH-BE-DEV-001', 2, 4, '负责后端接口开发、业务逻辑实现及系统稳定性维护', 'TECH', 6, 4, 5, false, 'ON', 6, now()),
    (7, 1, 2, 'REGULAR', '测试工程师', 'TECH-TEST-001', 2, 2, '负责项目功能测试、性能测试及自动化测试脚本开发', 'TECH', 3, 4, 3, false, 'ON', 7, now()),
    (8, 1, NULL, 'LEADER', '人力总监', 'HR-DIRECTOR-001', 2, NULL, '负责人力资源战略规划、组织架构设计及人才梯队建设', 'HR', 1, 1, 1, true, 'ON', 1, now()),
    (9, 1, 8, 'MANAGER', '招聘主管', 'HR-RECRUIT-LEADER-001', 2, 8, '负责公司各部门招聘需求对接、简历筛选及面试安排', 'HR', 2, 2, 1, false, 'ON', 2, now()),
    (10, 1, 8, 'REGULAR', '薪酬绩效专员', 'HR-C&P-001', 2, 8, '负责员工薪酬核算、绩效考核制度落地及社保公积金管理', 'HR', 3, 2, 1, false, 'ON', 3, now()),
    (11, 1, 8, 'REGULAR', 'HRBP', 'HR-BP-001', 2, 8, '对接业务部门，提供人力资源支持（入离职、员工关系等）', 'HR', 4, 2, 1, false, 'ON', 4, now()),
    (12, 1, NULL, 'LEADER', '财务总监', 'FIN-DIRECTOR-001', 2, NULL, '负责公司财务战略、预算管理及财务风险控制', 'FIN', 1, 1, 1, true, 'ON', 1, now()),
    (13, 1, 12, 'MANAGER', '会计主管', 'FIN-ACCOUNT-LEADER-001', 2, 12, '负责账务处理、财务报表编制及税务申报管理', 'FIN', 2, 2, 1, false, 'ON', 2, now()),
    (14, 1, 13, 'REGULAR', '出纳专员', 'FIN-CASHIER-001', 2, 13, '负责日常资金收付、银行对账及票据管理', 'FIN', 3, 3, 1, false, 'ON', 3, now()),
    (15, 1, 13, 'REGULAR', '成本会计', 'FIN-COST-001', 2, 13, '负责成本核算、成本分析及成本控制方案制定', 'FIN', 4, 3, 1, false, 'ON', 4, now()),
    (16, 1, NULL, 'LEADER', '市场总监', 'MKT-DIRECTOR-001', 4, NULL, '负责市场战略规划、品牌建设及营销活动策划', 'MKT', 1, 1, 1, true, 'ON', 1, now()),
    (17, 1, 16, 'MANAGER', '新媒体运营主管', 'MKT-NEWS-LEADER-001', 4, 16, '负责新媒体平台内容运营及用户增长', 'MKT', 2, 2, 1, false, 'ON', 2, now()),
    (18, 1, 16, 'REGULAR', '活动策划专员', 'MKT-EVENT-001', 4, 16, '负责线下活动策划、执行及效果复盘', 'MKT', 3, 3, 1, false, 'ON', 3, now()),
    (19, 1, 16, 'REGULAR', '市场调研专员', 'MKT-RESEARCH-001', 4, 16, '负责行业动态调研、竞品分析及市场趋势报告撰写', 'MKT', 4, 3, 1, false, 'ON', 4, now()),
    (20, 1, 8, 'REGULAR', '行政助理', 'ADMIN-ASSIST-001', 2, 8, '负责办公用品采购、会议安排等行政工作（已合并至HRBP）', 'ADMIN', 5, 5, 1, false, 'OFF', 5, now())
;
SELECT setval('sys_positions_id_seq', (SELECT COALESCE(MAX(id), 1) FROM sys_positions));

-- 用户-租户关联关系
INSERT INTO public.sys_memberships (id, tenant_id, user_id, org_unit_id, position_id, role_id, is_primary, status)
VALUES
    -- 租户管理员（TENANT_ADMIN）
    (2, 1, 2, null, null, 2, true, 'ACTIVE'),
    -- 普通用户（USER）
    (3, 0, 3, null, null, 3, true, 'ACTIVE'),
    -- 访客（GUEST）
    (4, 0, 4, null, null, 4, true, 'ACTIVE')
;
SELECT setval('sys_memberships_id_seq', (SELECT MAX(id) FROM sys_memberships));

-- 租户成员-角色关联关系
INSERT INTO sys_membership_roles (id, membership_id, tenant_id, role_id, is_primary, status)
VALUES
    -- 租户管理员（TENANT_ADMIN）
    (2, 2, 1, 2, true, 'ACTIVE'),
    -- 普通用户（USER）
    (3, 3, 0, 3, true, 'ACTIVE'),
    -- 访客（GUEST）
    (4, 4, 0, 4, true, 'ACTIVE')
SELECT setval('sys_membership_roles_id_seq', (SELECT MAX(id) FROM sys_membership_roles));

-- 调度任务
INSERT INTO public.sys_tasks(type, type_name, task_payload, cron_spec, enable, created_at)
VALUES
    ('PERIODIC', 'backup', '{ "name": "test"}', '0 * * * *', true, now())
;
SELECT setval('sys_tasks_id_seq', (SELECT MAX(id) FROM sys_tasks));

-- 后台登录限制
INSERT INTO public.sys_admin_login_restrictions(id, target_id, type, method, value, reason, created_at)
VALUES
(1, 1, 'BLACKLIST', 'IP', '127.0.0.1', '无理由', now()),
(2, 1, 'WHITELIST', 'MAC', '00:1B:44:11:3A:B7 ', '无理由', now())
;
SELECT setval('sys_admin_login_restrictions_id_seq', (SELECT MAX(id) FROM sys_admin_login_restrictions));

-- 字典类型
ALTER SEQUENCE public.sys_dict_types_id_seq RESTART WITH 1;
INSERT INTO public.sys_dict_types(id, type_code, type_name, sort_order, description, is_enabled, created_at)
VALUES
    (1, 'USER_STATUS', '用户状态', 10, '系统用户的状态管理，包括正常、冻结、注销', true, now()),
    (2, 'DEVICE_TYPE', '设备类型', 20, 'IoT平台接入的设备品类，新增需同步至设备接入模块', true, now()),
    (3, 'ORDER_STATUS', '订单状态', 30, '电商订单的全生命周期状态', true, now()),
    (4, 'GENDER', '性别', 40, '用户性别枚举，默认未知', true, now()),
    (5, 'PAYMENT_METHOD', '支付方式', 50, '支持的支付渠道，含第三方支付和自有渠道', true, now())
;
SELECT setval('sys_dict_types_id_seq', (SELECT MAX(id) FROM sys_dict_types));

-- 字典条目
INSERT INTO public.sys_dict_entries(id, type_id, entry_value, entry_label, numeric_value, sort_order, description, is_enabled, created_at)
VALUES
    -- 用户状态
    (1, 1, 'NORMAL', '正常', 1, 1, '用户可正常登录和操作', true, now()),
    (2, 1, 'FROZEN', '冻结', 2, 2, '因违规被临时冻结，需管理员解冻', true, now()),
    (3, 1, 'CANCELED', '注销', 3, 3, '用户主动注销，数据保留但不可登录', true, now()),
    -- 设备类型
    (4, 2, 'TEMP_SENSOR', '温湿度传感器', 101, 1, '支持温度（-20~80℃）和湿度（0~100%RH）采集', true, now()),
    (5, 2, 'CURRENT_METER', '电流仪表', 102, 2, '交流/直流电流测量，精度0.5级', true, now()),
    (6, 2, 'GAS_DETECTOR', '气体探测器', 103, 3, '暂不支持，待硬件适配（2025Q4计划启用）', false, now()),
    -- 订单状态
    (7, 3, 'PENDING', '待支付', 1, 1, '下单后未支付，超时自动取消', true, now()),
    (8, 3, 'PAID', '已支付', 2, 2, '支付成功，等待发货', true, now()),
    (9, 3, 'SHIPPED', '已发货', 3, 3, '商品已出库，物流配送中', true, now()),
    (10, 3, 'COMPLETED', '已完成', 4, 4, '用户确认收货，订单结束', true, now()),
    (11, 3, 'CANCELED', '已取消', 5, 5, '用户或系统取消订单', true, now()),
    -- 性别
    (12, 4, 'MALE', '男', 1, 1, '', true, now()),
    (13, 4, 'FEMALE', '女', 2, 2, '', true, now()),
    (14, 4, 'UNKNOWN', '未知', 0, 3, '用户未填写时默认值', true, now()),
    -- 支付方式
    (15, 5, 'ALIPAY', '支付宝', 1, 1, '支持花呗、余额宝', true, now()),
    (16, 5, 'WECHAT', '微信支付', 2, 2, '需绑定微信', true, now()),
    (17, 5, 'UNIONPAY', '银联支付', 3, 3, '支持信用卡、储蓄卡', true, now()),
    (18, 5, 'CASH', '现金支付', 4, 4, '线下支付，已废弃（2025-01停用）', false, now())
;
SELECT setval('sys_dict_entries_id_seq', (SELECT MAX(id) FROM sys_dict_entries));

-- 站内信分类 - 主分类
INSERT INTO public.internal_message_categories (id, parent_id, code, name, remark, sort_order, is_enabled, created_at)
VALUES
    -- 订单相关主分类
    (1, null, 'order', '订单通知', '包含订单支付、发货、退款等全流程通知', 1, true, NOW()),
    -- 系统相关主分类
    (2, null, 'system', '系统通知', '系统公告、维护提醒、版本更新等平台级通知', 2, true, NOW()),
    -- 活动相关主分类
    (3, null, 'activity', '活动通知', '营销活动报名、开始、结束等提醒', 3, true, NOW()),
    -- 用户相关主分类
    (4, null, 'user', '用户通知', '账号安全、信息变更、权限调整等个人相关通知', 4, true, NOW())
;
-- 站内信分类 - 子分类
INSERT INTO public.internal_message_categories (id, parent_id, code, name, remark, sort_order, is_enabled, created_at)
VALUES
    -- 订单主分类（id=1）的子分类
    (101, 1, 'order_paid', '支付成功', '订单支付完成时触发的通知', 1, true, NOW()),
    (102, 1, 'order_unpaid', '支付超时', '订单未在规定时间内支付的提醒', 2, true, NOW()),
    (103, 1, 'order_shipped', '已发货', '商家发货后通知用户', 3, true, NOW()),
    (104, 1, 'order_refunded', '已退款', '订单退款流程完成的通知', 4, true, NOW()),
    -- 系统主分类（id=2）的子分类
    (201, 2, 'system_announcement', '系统公告', '平台规则更新、重要通知等', 1, true, NOW()),
    (202, 2, 'system_maintenance', '维护通知', '系统计划内维护的时间提醒', 2, true, NOW()),
    (203, 2, 'system_upgrade', '版本更新', '客户端或功能升级的提示', 3, true, NOW()),
    -- 活动主分类（id=3）的子分类
    (301, 3, 'activity_signup', '报名成功', '用户报名活动后确认通知', 1, true, NOW()),
    (302, 3, 'activity_start', '活动开始', '活动即将开始的倒计时提醒', 2, true, NOW()),
    (303, 3, 'activity_end', '活动结束', '活动结束及结果公示通知', 3, true, NOW()),
    -- 用户主分类（id=4）的子分类
    (401, 4, 'user_login_abnormal', '异地登录', '账号在陌生设备登录的安全提醒', 1, true, NOW()),
    (402, 4, 'user_profile_updated', '资料变更', '用户手机号、邮箱等信息修改后通知', 2, true, NOW()),
    (403, 4, 'user_permission_changed', '权限变更', '账号角色或功能权限调整通知', 3, true, NOW())
;
SELECT setval('internal_message_categories_id_seq', (SELECT MAX(id) FROM internal_message_categories));

COMMIT;
