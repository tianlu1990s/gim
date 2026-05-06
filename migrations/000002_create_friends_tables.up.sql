CREATE TABLE friends (
    id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    owner_id   VARCHAR(64)  NOT NULL,
    friend_id  VARCHAR(64)  NOT NULL,
    remark     VARCHAR(64)  NOT NULL DEFAULT '',
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_owner_friend (owner_id, friend_id),
    INDEX idx_owner (owner_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE friend_requests (
    id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    from_user_id VARCHAR(64)  NOT NULL,
    to_user_id   VARCHAR(64)  NOT NULL,
    message      VARCHAR(256) NOT NULL DEFAULT '',
    status       TINYINT      NOT NULL DEFAULT 0 COMMENT '0=待处理 1=已同意 2=已拒绝',
    created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_to_user (to_user_id),
    INDEX idx_from_to (from_user_id, to_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
