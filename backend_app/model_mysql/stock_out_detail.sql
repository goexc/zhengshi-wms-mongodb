CREATE TABLE `stock_out_detail`
(
    `id`            int(10) unsigned NOT NULL AUTO_INCREMENT,
    `stock_id`      int(10) unsigned NOT NULL COMMENT '出库单id',
    `product_id`    int(10) unsigned NOT NULL COMMENT '产品id',
    `product_model` varchar(64)      NOT NULL COMMENT '产品型号',
    `product_name`  varchar(64)      NOT NULL COMMENT '产品品名',
    `product_specs` varchar(64)      NOT NULL COMMENT '产品规格',
    `count`         int(10) unsigned NOT NULL COMMENT '产品数量（个、件）',
    `price`         decimal(9, 4) unsigned NOT NULL COMMENT '产品单价(厘/个)',
    `created_at`    timestamp        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`    timestamp        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    INDEX `idx_stock_id` (`stock_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;