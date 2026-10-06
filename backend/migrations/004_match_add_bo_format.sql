USE kpl_bp;
-- 系列赛制：5=BO5(先3胜), 7=BO7(先4胜), 0=无限制(不自动结束)
ALTER TABLE match_record
  ADD COLUMN bo_format TINYINT NOT NULL DEFAULT 5
  COMMENT '赛制: 5=BO5, 7=BO7, 0=无限制';