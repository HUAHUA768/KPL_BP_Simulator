CREATE DATABASE IF NOT EXISTS kpl_bp CHARACTER SET utf8mb4;
CREATE TABLE team (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(50) NOT NULL UNIQUE,
    logo_url VARCHAR(255) DEFAULT NULL,      -- 战队图标（可选）
    is_preset TINYINT(1) DEFAULT 0,          -- 是否为预设战队（后续优化）
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE match_record (
    id INT PRIMARY KEY AUTO_INCREMENT,
    team_a_id INT NOT NULL,          -- 参赛队伍A
    team_b_id INT NOT NULL,          -- 参赛队伍B
    team_a_score INT DEFAULT 0,      -- A队总得分
    team_b_score INT DEFAULT 0,      -- B队总得分
    current_game_num INT DEFAULT 1,  -- 当前进行到第几局
    side_picker_team_id INT DEFAULT NULL,  -- 当前拥有选边权的队伍ID（可为空，初始由外部决定）
    status ENUM('pending','ongoing','finished') DEFAULT 'pending',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (team_a_id) REFERENCES team(id),
    FOREIGN KEY (team_b_id) REFERENCES team(id),
    FOREIGN KEY (side_picker_team_id) REFERENCES team(id)
);
CREATE TABLE game (
    id INT PRIMARY KEY AUTO_INCREMENT,
    match_id INT NOT NULL,
    game_number INT NOT NULL,                 -- 第几局（1,2,3...）
    blue_team_id INT NOT NULL,
    red_team_id INT NOT NULL,
    winner ENUM('blue','red') DEFAULT NULL,
    bp_completed TINYINT(1) DEFAULT 0,        -- BP是否完成
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (match_id) REFERENCES match_record(id) ON DELETE CASCADE,
    FOREIGN KEY (blue_team_id) REFERENCES team(id),
    FOREIGN KEY (red_team_id) REFERENCES team(id),
    CONSTRAINT chk_bp_completed CHECK(bp_completed IN (0,1))    -- 显式命名约束
);
CREATE TABLE hero (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(20) NOT NULL UNIQUE,
    icon_path VARCHAR(255) NOT NULL,          -- 例如 '/images/heroes/guan_yu.png'
    lanes JSON DEFAULT NULL,                  -- 推荐分路，如 ["对抗路","打野"] （JSON数组）
    is_active TINYINT(1) DEFAULT 1
);
CREATE TABLE ban_pick_action (
    id INT PRIMARY KEY AUTO_INCREMENT,
    game_id INT NOT NULL,
    step_order INT NOT NULL,                  -- 操作序号（1~10 ban + 10 pick = 20步？实际KPL是10ban10pick共20步）
    action_type ENUM('ban','pick') NOT NULL,
    side ENUM('blue','red') NOT NULL,
    hero_id INT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES game(id) ON DELETE CASCADE,
    FOREIGN KEY (hero_id) REFERENCES hero(id)
);