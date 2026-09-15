-- 访客通行模块（GORM AutoMigrate 在启动时自动建表；此文件记录 schema 归属）。
-- visitor_passes        访客通行凭证：pass_no 唯一、状态机 pending/approved/checked_in/completed/cancelled/rejected/expired。
-- visitor_events        取消、审核、进入、离开、逾期的全生命周期留痕（与状态变更同事务提交）。
-- building_capacities   楼栋当日访客在场上限（在场数由 visitor_passes.status='checked_in' 实时统计，离开/逾期即恢复）。
-- 时间约定：所有时间列以 UTC 绝对时刻存储（GORM BeforeSave 归一化），接口统一按社区时区（默认 Asia/Shanghai，APP_TIMEZONE 可配）渲染为 YYYY-MM-DD HH:mm 钟面串。
-- 关键业务约束（应用层 + 事务保证）：
--   1) 同一 visitor_phone + building 在有效状态(pending/approved/checked_in)下时段不得重叠；
--   2) 审核与进入时校验在场数 < building_capacities.daily_limit，满则暂停（HTTP 409）；
--   3) 门岗仅对 approved 且当前时间落在 [start_time, end_time] 内的凭证放行。
CREATE TABLE IF NOT EXISTS visitor_passes (
  id               BIGINT PRIMARY KEY AUTO_INCREMENT,
  pass_no          VARCHAR(32) NOT NULL UNIQUE,
  resident_id      BIGINT NOT NULL,
  visitor_name     VARCHAR(50) NOT NULL,
  visitor_phone    VARCHAR(30) NOT NULL,
  building         VARCHAR(50) NOT NULL,
  reason           TEXT NOT NULL,
  start_time       DATETIME(3) NOT NULL,
  end_time         DATETIME(3) NOT NULL,
  status           VARCHAR(20) NOT NULL,
  reviewer_id      BIGINT NULL,
  review_remark    TEXT,
  reviewed_at      DATETIME(3) NULL,
  check_in_at      DATETIME(3) NULL,
  check_out_at     DATETIME(3) NULL,
  checkpoint       VARCHAR(50),
  expire_marked_at DATETIME(3) NULL,
  created_at       DATETIME(3),
  updated_at       DATETIME(3),
  INDEX idx_pass_resident (resident_id),
  INDEX idx_pass_phone (visitor_phone),
  INDEX idx_pass_building (building),
  INDEX idx_pass_status (status),
  INDEX idx_pass_window (start_time, end_time)
);
CREATE TABLE IF NOT EXISTS visitor_events (
  id          BIGINT PRIMARY KEY AUTO_INCREMENT,
  pass_id     BIGINT NOT NULL,
  action      VARCHAR(40) NOT NULL,
  actor_id    BIGINT NULL,
  from_status VARCHAR(20),
  to_status   VARCHAR(20),
  detail      TEXT,
  created_at  DATETIME(3),
  INDEX idx_event_pass (pass_id),
  INDEX idx_event_action (action)
);
CREATE TABLE IF NOT EXISTS building_capacities (
  id          BIGINT PRIMARY KEY AUTO_INCREMENT,
  building    VARCHAR(50) NOT NULL UNIQUE,
  daily_limit INT NOT NULL,
  updated_by  BIGINT NOT NULL,
  created_at  DATETIME(3),
  updated_at  DATETIME(3)
);
