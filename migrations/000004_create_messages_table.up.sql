CREATE TABLE messages (
    id               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    conversation_id  VARCHAR(128) NOT NULL,
    seq              BIGINT       NOT NULL,
    sender_id        VARCHAR(64)  NOT NULL,
    msg_type         INT          NOT NULL DEFAULT 1 COMMENT '1=文本 2=图片 3=文件 4=系统消息',
    content          TEXT         NOT NULL,
    client_msg_id    VARCHAR(64)  NOT NULL,
    server_msg_id    VARCHAR(64)  NOT NULL,
    is_revoked       TINYINT(1)   NOT NULL DEFAULT 0,
    created_at       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_client_msg (client_msg_id),
    INDEX idx_conv_seq (conversation_id, seq),
    INDEX idx_server_msg (server_msg_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
