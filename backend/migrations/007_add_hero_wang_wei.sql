USE kpl_bp;

-- ============================================================================
-- 新增英雄：王维（中路）
-- ----------------------------------------------------------------------------
-- 图标来源：backend/static/images/heroes/wang_wei.png
--           （已同步一份到 frontend/public/images/heroes/wang_wei.png）
-- 分路定位：中路
--
-- 幂等性：使用 INSERT ... WHERE NOT EXISTS，可重复执行不会产生重复行。
-- ============================================================================

INSERT INTO hero (name, icon_path, lanes, is_active)
SELECT '王维', '/images/heroes/wang_wei.png', '["中路"]', 1
  FROM DUAL
 WHERE NOT EXISTS (SELECT 1 FROM hero WHERE name = '王维');

-- 自检：应能查到王维，且 icon_path 指向已存在的图标文件
SELECT id, name, icon_path, lanes, is_active FROM hero WHERE name = '王维';

-- 自检：全库英雄总数与图标路径唯一性
SELECT COUNT(*) AS hero_total FROM hero;
SELECT icon_path, COUNT(*) AS c, GROUP_CONCAT(name) AS names
  FROM hero GROUP BY icon_path HAVING c > 1;