USE kpl_bp;

-- ============================================================================
-- 问题3修复：曜 / 瑶 同音英雄的图标路径区分
-- ----------------------------------------------------------------------------
-- 背景：
--   「曜」与「瑶」读音完全相同（yáo），旧数据里两者仅靠 .jpg / .png 扩展名区分，
--   极易写错。线上库实测结果为两行 icon_path 完全相同：
--       id=31  曜  icon_path='/images/heroes/yao.png'   ← 错误，指向了瑶的图标
--       id=127 瑶  icon_path='/images/heroes/yao.png'   ← 正确
--   导致分路筛选「打野 / 对抗路」时，曜显示成瑶的图标。
--
-- 方案（双管齐下，治本）：
--   ① 图标文件重命名成语义化文件名，消灭"靠扩展名区分同音英雄"的脆弱约定：
--        yao.jpg（内容是 PNG）→ yao_hero.png      （曜）
--        yao.png              → yao_support.png   （瑶）
--   ② 本迁移把 hero 表两行的 name / icon_path / lanes 一并校正。
--
-- 前置条件：必须先完成图标文件重命名（见施工方案 §三 问题3 步骤1），
--          否则修复后会出现图片 404。
-- 幂等性：本迁移可重复执行，结果一致。
-- ============================================================================

-- 1) 曜（英雄，战士/刺客，打野 + 对抗路）
UPDATE hero
   SET name      = '曜',
       icon_path = '/images/heroes/yao_hero.png',
       lanes     = '["打野","对抗路"]'
 WHERE name = '曜';

-- 2) 瑶（辅助，游走）
UPDATE hero
   SET name      = '瑶',
       icon_path = '/images/heroes/yao_support.png',
       lanes     = '["游走"]'
 WHERE name = '瑶';

-- 3) 兜底：若历史上曾把曜的 name 误存成瑶（编码事故），
--    则按"旧路径指向 yao 且分路含对抗路"的语义特征精确定位并修正。
UPDATE hero
   SET name      = '曜',
       icon_path = '/images/heroes/yao_hero.png',
       lanes     = '["打野","对抗路"]'
 WHERE icon_path IN ('/images/heroes/yao.jpg', '/images/heroes/yao.png')
   AND lanes LIKE '%对抗路%'
   AND name <> '曜';

-- 4) 兜底：其余仍指向旧 yao.png 的（即瑶）统一到新路径
UPDATE hero
   SET icon_path = '/images/heroes/yao_support.png'
 WHERE icon_path IN ('/images/heroes/yao.png', '/images/heroes/yao.jpg')
   AND name = '瑶';

-- 5) 自检：应恰好 2 行，且 icon_path 互不相同
SELECT id, name, HEX(name) AS name_hex, icon_path, lanes
  FROM hero WHERE name IN ('曜','瑶');

-- 6) 自检：全库不应再有任何重复 icon_path
SELECT icon_path, COUNT(*) AS c, GROUP_CONCAT(name) AS names
  FROM hero GROUP BY icon_path HAVING c > 1;