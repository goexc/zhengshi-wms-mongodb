CREATE TABLE `product_price`
(
    `id`            int(10) unsigned NOT NULL AUTO_INCREMENT,
    `product_id`    int(10) unsigned NOT NULL COMMENT '产品id',
    `price`         decimal(9, 4) unsigned NOT NULL COMMENT '产品单价(元)',
    `created_at`    timestamp        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`    timestamp        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    INDEX `idx_product_id` (`product_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;