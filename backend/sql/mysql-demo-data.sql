USE `gwa`;

SET FOREIGN_KEY_CHECKS = 0;

-- 租户
TRUNCATE TABLE `sys_tenants`;
INSERT INTO `sys_tenants`(id, name, code, type, audit_status, status, admin_user_id, created_at)
VALUES
    (1, '超级租户', 'super', 'PAID', 'APPROVED', 'ON', 1, NOW()),
    (2, '测试租户', 'test', 'PAID', 'APPROVED', 'ON', NULL, NOW()),
    (3, '测试租户2', 'test2', 'PAID', 'APPROVED', 'ON', NULL, NOW());

-- 组织
TRUNCATE TABLE `sys_organizations`;
INSERT INTO `sys_organizations`(id, parent_id, sort_order, manager_id, name, organization_type, is_legal_entity, business_scope, status, created_at)
VALUES
    (1, NULL, 1, 1, '虾米集团', 'GROUP', 1, '综合型集团企业，涵盖多领域', 'ON', NOW()),
    (2, 1, 2, 1, '北京分公司', 'SUBSIDIARY', 0, '负责华北区域业务运营', 'ON', NOW()),
    (3, 1, 3, 1, '上海子公司', 'FILIALE', 1, '负责华东区域研发与生产', 'ON', NOW()),
    (4, 1, 4, 1, '新能源事业部', 'DIVISION', 0, '新能源汽车技术研发与市场拓展', 'ON', NOW());

-- 部门
TRUNCATE TABLE `sys_departments`;
INSERT INTO `sys_departments`(id, parent_id, sort_order, organization_id, manager_id, name, description, status, created_at)
VALUES
    (1, NULL, 1, 2, 1, '技术部', '负责北京分公司系统开发', 'ON', NOW()),
    (2, NULL, 2, 2, 1, '财务部', '负责北京分公司财务核算', 'ON', NOW()),
    (3, NULL, 3, 2, 1, '人力资源部', '负责北京分公司人员招聘', 'ON', NOW()),
    (4, NULL, 4, 3, 1, '研发一部', '上海子公司核心技术研发', 'ON', NOW()),
    (5, NULL, 5, 4, 1, '市场部', '新能源事业部市场推广', 'ON', NOW()),
    (6, 1, 1, 2, 1, '前端组', '技术部下属前端开发团队', 'ON', NOW());

-- 职位
TRUNCATE TABLE `sys_positions`;
INSERT INTO `sys_positions` (id, name, code, parent_id, department_id, organization_id, quota, description, status, sort_order, created_at)
VALUES
    (1, '技术总监', 'TECH-DIRECTOR-001', NULL, 1, 2, 1, '负责公司整体技术战略规划、团队管理及核心技术决策', 'ON', 1, NOW()),
    (2, '技术部经理', 'TECH-MANAGER-001', 1, 1, 2, 1, '负责技术部日常管理、项目排期及团队协作', 'ON', 2, NOW()),
    (3, '前端主管', 'TECH-FE-LEADER-001', 2, 1, 2, 1, '负责前端团队开发管理、技术方案评审及需求落地', 'ON', 3, NOW()),
    (4, '后端主管', 'TECH-BE-LEADER-001', 2, 1, 2, 1, '负责后端服务架构设计、数据库优化及接口开发管理', 'ON', 4, NOW()),
    (5, '前端开发专员', 'TECH-FE-DEV-001', 3, 1, 2, 5, '负责Web/移动端前端页面开发、交互实现及兼容性优化', 'ON', 5, NOW()),
    (6, '后端开发专员', 'TECH-BE-DEV-001', 4, 1, 2, 5, '负责后端接口开发、业务逻辑实现及系统稳定性维护', 'ON', 6, NOW()),
    (7, '测试工程师', 'TECH-TEST-001', 2, 1, 2, 3, '负责项目功能测试、性能测试及自动化测试脚本开发', 'ON', 7, NOW()),
    (8, '人力总监', 'HR-DIRECTOR-001', NULL, 3, 2, 1, '负责人力资源战略规划、组织架构设计及人才梯队建设', 'ON', 1, NOW()),
    (9, '招聘主管', 'HR-RECRUIT-LEADER-001', 8, 3, 2, 2, '负责公司各部门招聘需求对接、简历筛选及面试安排', 'ON', 2, NOW()),
    (10, '薪酬绩效专员', 'HR-C&P-001', 8, 3, 2, 2, '负责员工薪酬核算、绩效考核制度落地及社保公积金管理', 'ON', 3, NOW()),
    (11, 'HRBP', 'HR-BP-001', 8, 3, 2, 3, '对接业务部门，提供人力资源支持（入离职、员工关系等）', 'ON', 4, NOW()),
    (12, '财务总监', 'FIN-DIRECTOR-001', NULL, 2, 2, 1, '负责公司财务战略、预算管理及财务风险控制', 'ON', 1, NOW()),
    (13, '会计主管', 'FIN-ACCOUNT-LEADER-001', 12, 2, 2, 1, '负责账务处理、财务报表编制及税务申报管理', 'ON', 2, NOW()),
    (14, '出纳专员', 'FIN-CASHIER-001', 13, 2, 2, 2, '负责日常资金收付、银行对账及票据管理', 'ON', 3, NOW()),
    (15, '成本会计', 'FIN-COST-001', 13, 2, 2, 1, '负责成本核算、成本分析及成本控制方案制定', 'ON', 4, NOW()),
    (16, '市场总监', 'MKT-DIRECTOR-001', NULL, 5, 4, 1, '负责市场战略规划、品牌建设及营销活动策划', 'ON', 1, NOW()),
    (17, '新媒体运营主管', 'MKT-NEWS-LEADER-001', 16, 5, 4, 1, '负责新媒体平台（微信、抖音等）内容运营及用户增长', 'ON', 2, NOW()),
    (18, '活动策划专员', 'MKT-EVENT-001', 16, 5, 4, 3, '负责线下活动策划、执行及效果复盘', 'ON', 3, NOW()),
    (19, '市场调研专员', 'MKT-RESEARCH-001', 16, 5, 4, 2, '负责行业动态调研、竞品分析及市场趋势报告撰写', 'ON', 4, NOW()),
    (20, '行政助理', 'ADMIN-ASSIST-001', 8, 3, 2, 0, '负责办公用品采购、会议安排等行政工作（已合并至HRBP）', 'OFF', 5, NOW());

-- 调度任务
TRUNCATE TABLE `sys_tasks`;
INSERT INTO `sys_tasks`(type, type_name, task_payload, cron_spec, enable, created_at)
VALUES
    ('PERIODIC', 'backup', '{ "name": "test"}', '0 * * * *', 1, NOW());

-- 登录策略
TRUNCATE TABLE `sys_login_policies`;
INSERT INTO `sys_login_policies`(id, target_id, type, method, value, reason, created_at)
VALUES
    (1, 1, 'BLACKLIST', 'IP', '127.0.0.1', '无理由', NOW()),
    (2, 1, 'WHITELIST', 'MAC', '00:1B:44:11:3A:B7', '无理由', NOW());

-- 字典类型
TRUNCATE TABLE `sys_dict_types`;
INSERT INTO `sys_dict_types`(id, type_code, type_name, sort_order, description, is_enabled, created_at)
VALUES
    (1, 'USER_STATUS', '用户状态', 10, '系统用户的状态管理，包括正常、冻结、注销', 1, NOW()),
    (2, 'DEVICE_TYPE', '设备类型', 20, 'IoT平台接入的设备品类，新增需同步至设备接入模块', 1, NOW()),
    (3, 'ORDER_STATUS', '订单状态', 30, '电商订单的全生命周期状态', 1, NOW()),
    (4, 'GENDER', '性别', 40, '用户性别枚举，默认未知', 1, NOW()),
    (5, 'PAYMENT_METHOD', '支付方式', 50, '支持的支付渠道，含第三方支付和自有渠道', 1, NOW());

-- 字典条目
TRUNCATE TABLE `sys_dict_entries`;
INSERT INTO `sys_dict_entries`(id, type_id, entry_value, entry_label, numeric_value, sort_order, description, is_enabled, created_at)
VALUES
    (1, 1, 'NORMAL', '正常', 1, 1, '用户可正常登录和操作', 1, NOW()),
    (2, 1, 'FROZEN', '冻结', 2, 2, '因违规被临时冻结，需管理员解冻', 1, NOW()),
    (3, 1, 'CANCELED', '注销', 3, 3, '用户主动注销，数据保留但不可登录', 1, NOW()),
    (4, 2, 'TEMP_SENSOR', '温湿度传感器', 101, 1, '支持温度（-20~80℃）和湿度（0~100%RH）采集', 1, NOW()),
    (5, 2, 'CURRENT_METER', '电流仪表', 102, 2, '交流/直流电流测量，精度0.5级', 1, NOW()),
    (6, 2, 'GAS_DETECTOR', '气体探测器', 103, 3, '暂不支持，待硬件适配（2025Q4计划启用）', 0, NOW()),
    (7, 3, 'PENDING', '待支付', 1, 1, '下单后未支付，超时自动取消', 1, NOW()),
    (8, 3, 'PAID', '已支付', 2, 2, '支付成功，等待发货', 1, NOW()),
    (9, 3, 'SHIPPED', '已发货', 3, 3, '商品已出库，物流配送中', 1, NOW()),
    (10, 3, 'COMPLETED', '已完成', 4, 4, '用户确认收货，订单结束', 1, NOW()),
    (11, 3, 'CANCELED', '已取消', 5, 5, '用户或系统取消订单', 1, NOW()),
    (12, 4, 'MALE', '男', 1, 1, '', 1, NOW()),
    (13, 4, 'FEMALE', '女', 2, 2, '', 1, NOW()),
    (14, 4, 'UNKNOWN', '未知', 0, 3, '用户未填写时默认值', 1, NOW()),
    (15, 5, 'ALIPAY', '支付宝', 1, 1, '支持花呗、余额宝', 1, NOW()),
    (16, 5, 'WECHAT', '微信支付', 2, 2, '需绑定微信', 1, NOW()),
    (17, 5, 'UNIONPAY', '银联支付', 3, 3, '支持信用卡、储蓄卡', 1, NOW()),
    (18, 5, 'CASH', '现金支付', 4, 4, '线下支付，已废弃（2025-01停用）', 0, NOW());

-- 站内信分类 - 主分类 & 子分类
TRUNCATE TABLE `internal_message_categories`;
INSERT INTO `internal_message_categories` (id, parent_id, code, name, remark, sort_order, is_enabled, created_at)
VALUES
    (1, NULL, 'order', '订单通知', '包含订单支付、发货、退款等全流程通知', 1, 1, NOW()),
    (2, NULL, 'system', '系统通知', '系统公告、维护提醒、版本更新等平台级通知', 2, 1, NOW()),
    (3, NULL, 'activity', '活动通知', '营销活动报名、开始、结束等提醒', 3, 1, NOW()),
    (4, NULL, 'user', '用户通知', '账号安全、信息变更、权限调整等个人相关通知', 4, 1, NOW()),
    (101, 1, 'order_paid', '支付成功', '订单支付完成时触发的通知', 1, 1, NOW()),
    (102, 1, 'order_unpaid', '支付超时', '订单未在规定时间内支付的提醒', 2, 1, NOW()),
    (103, 1, 'order_shipped', '已发货', '商家发货后通知用户', 3, 1, NOW()),
    (104, 1, 'order_refunded', '已退款', '订单退款流程完成的通知', 4, 1, NOW()),
    (201, 2, 'system_announcement', '系统公告', '平台规则更新、重要通知等', 1, 1, NOW()),
    (202, 2, 'system_maintenance', '维护通知', '系统计划内维护的时间提醒', 2, 1, NOW()),
    (203, 2, 'system_upgrade', '版本更新', '客户端或功能升级的提示', 3, 1, NOW()),
    (301, 3, 'activity_signup', '报名成功', '用户报名活动后确认通知', 1, 1, NOW()),
    (302, 3, 'activity_start', '活动开始', '活动即将开始的倒计时提醒', 2, 1, NOW()),
    (303, 3, 'activity_end', '活动结束', '活动结束及结果公示通知', 3, 1, NOW()),
    (401, 4, 'user_login_abnormal', '异地登录', '账号在陌生设备登录的安全提醒', 1, 1, NOW()),
    (402, 4, 'user_profile_updated', '资料变更', '用户手机号、邮箱等信息修改后通知', 2, 1, NOW()),
    (403, 4, 'user_permission_changed', '权限变更', '账号角色或功能权限调整通知', 3, 1, NOW());

SET FOREIGN_KEY_CHECKS = 1;
