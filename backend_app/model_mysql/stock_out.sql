CREATE TABLE `stock_out`
(
    `id`          int(10) unsigned       NOT NULL AUTO_INCREMENT,
    `client_id`   int(10) unsigned       NOT NULL COMMENT '客户id',
    `client_name` varchar(255)           NOT NULL COMMENT '客户名称',
    `numbering`   varchar(64)            NOT NULL COMMENT '编号',
    `has_tax`     bool                   NOT NULL COMMENT '是否含税',
    `tax`         tinyint(4) unsigned    NOT NULL COMMENT '税率(%)',
    `total`       decimal(9, 4) unsigned NOT NULL COMMENT '总金额(元)',
    `date`        int(10) unsigned       NOT NULL DEFAULT 0 COMMENT '出库日期',
    `created_at`  timestamp              NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`  timestamp              NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    INDEX `idx_date_client` (`date`,`client_id`),
    INDEX `idx_number` (`numbering`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;