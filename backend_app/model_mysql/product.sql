CREATE TABLE `product`
(
    `id`         int(10) unsigned NOT NULL AUTO_INCREMENT,
    `client_id`  int(10) unsigned NOT NULL COMMENT '客户id',
    `model`      varchar(64)     NOT NULL COMMENT '产品型号',
    `name`       varchar(64)     NOT NULL COMMENT '产品品名',
    `specs`      varchar(64)     NOT NULL COMMENT '产品规格',
    `created_at` timestamp        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` timestamp        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_model` (`model`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;