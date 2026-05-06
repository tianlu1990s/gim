CREATE TABLE conversations (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    owner_id        VARCHAR(64)  NOT NULL,
    conversation_id VARCHAR(128) NOT NULL,
    conv_type       INT          NOT NULL DEFAULT 1 COMMENT '1=单聊 2=群聊',
    target_id       VARCHAR(64)  NOT NULL DEFAULT '',
    max_seq         BIGINT       NOT NULL DEFAULT 0,
    is_pinned       TINYINT(1)   NOT NULL DEFAULT 0,
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_owner_conv (owner_id, conversation_id),
    INDEX idx_owner_conv (owner_id, conversation_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
