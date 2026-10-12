USE kpl_bp;

-- ============================================================================
-- 问题1修复：战队队标路径规范化
-- ----------------------------------------------------------------------------
-- 背景：
--   team.logo_url 存的是"相对路径"（如 'images/teams/x.png'，无前导斜杠），
--   而 hero.icon_path 历来是"绝对路径"（'/images/heroes/x.png'）。
--   前端在 /room/:id 页面渲染队标时，相对路径会以当前页面为基准解析成
--   /room/images/teams/x.png → 必然 404（且被 onError 静默隐藏，问题难被发现）。
--
-- 方案：
--   统一补上前导斜杠，与后端 r.Static("/images", "./static/images") 对齐。
--   同时后端 normalizeLogoURL() 会兜住未来新写入的相对路径（双保险）。
--
-- 幂等性：可重复执行。
-- ============================================================================

UPDATE team
   SET logo_url = CONCAT('/', logo_url)
 WHERE logo_url IS NOT NULL
   AND logo_url <> ''
   AND logo_url NOT LIKE '/%'
   AND logo_url NOT LIKE 'http://%'
   AND logo_url NOT LIKE 'https://%';

-- 自检：不应再有相对路径（期望 0 行）
SELECT COUNT(*) AS relative_logo_count
  FROM team
 WHERE logo_url IS NOT NULL
   AND logo_url <> ''
   AND logo_url NOT LIKE '/%'
   AND logo_url NOT LIKE 'http://%'
   AND logo_url NOT LIKE 'https://%';

SELECT id, name, logo_url FROM team ORDER BY id;