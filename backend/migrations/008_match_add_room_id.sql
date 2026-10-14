USE kpl_bp;

-- ============================================================================
-- 新增功能：未结束的比赛可从对局记录中"继续模拟 BP"
-- ----------------------------------------------------------------------------
-- 背景：
--   房间（Room）此前只存在于后端内存中，roomId 由随机 hex 生成但**从未落库**。
--   因此后端一重启，DB 里那些 status='ongoing' 的比赛就再也回不到房间，
--   用户无法继续把它们打完。
--
-- 方案：
--   给 match_record 增加 room_id 列并落库，使"比赛记录 → 房间"可反查。
--   恢复时若内存中已无该房间，则依据 game / ban_pick_action 回放重建。
--
-- 兼容性：
--   本列允许 NULL，且对旧数据全部回填。旧记录即使没有 room_id，
--   仍可通过 matchId 重建房间（新建一个 roomId 并回写），不会成为孤儿。
-- 幂等性：可重复执行。
-- ============================================================================

-- 1) 加列（MySQL 8.0 不支持 ADD COLUMN IF NOT EXISTS，用存储过程保证幂等）
DROP PROCEDURE IF EXISTS add_room_id_column;
DELIMITER //
CREATE PROCEDURE add_room_id_column()
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.COLUMNS
         WHERE TABLE_SCHEMA = DATABASE()
           AND TABLE_NAME   = 'match_record'
           AND COLUMN_NAME  = 'room_id'
    ) THEN
        ALTER TABLE match_record
          ADD COLUMN room_id VARCHAR(32) DEFAULT NULL
          COMMENT '房间号（用于从对局记录恢复房间）；旧数据可为 NULL';
    END IF;
END //
DELIMITER ;
CALL add_room_id_column();
DROP PROCEDURE add_room_id_column;

-- 2) 幂等加唯一索引（room_id 不应重复；NULL 可重复，不影响旧数据）
DROP PROCEDURE IF EXISTS add_room_id_index;
DELIMITER //
CREATE PROCEDURE add_room_id_index()
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.STATISTICS
         WHERE TABLE_SCHEMA = DATABASE()
           AND TABLE_NAME   = 'match_record'
           AND INDEX_NAME   = 'uk_match_room'
    ) THEN
        ALTER TABLE match_record
          ADD UNIQUE KEY uk_match_room (room_id);
    END IF;
END //
DELIMITER ;
CALL add_room_id_index();
DROP PROCEDURE add_room_id_index;

-- 3) 自检
SELECT COUNT(*) AS total_matches,
       SUM(room_id IS NOT NULL) AS with_room_id
  FROM match_record;

SHOW COLUMNS FROM match_record LIKE 'room_id';