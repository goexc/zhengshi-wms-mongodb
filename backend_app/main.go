package main

import (
	"backend/config"
	"backend/handler"
	"backend/svc"
	"flag"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

var configFile = flag.String("f", "etc/main.yaml", "the config file")

func main() {
	flag.Parse()
	logx.DisableStat()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	ctx := svc.NewServiceContext(c)

	//客户数据同步
	//handler.CustomerSync(ctx)

	//旧系统物料有重复数据，暂不使用
	//handler.MaterialSync(ctx)

	//物料单价数据同步
	//handler.PriceSync(ctx)

	//出库单数据同步
	//handler.OutboundSync(ctx)

	//入库单物料数据同步
	//handler.OutboundMaterialSync(ctx)

	//同步客户交易流水
	handler.CustomerTransactionSync(ctx)

	select {}
}
