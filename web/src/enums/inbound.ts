/*入库单*/

//入库单状态
export const InboundReceiptStatus= [
  // '全部' = '',
  { label:'待审核', value: 10},//入库单已提交但还未通过审核时，状态为待审核。需要相关审核人员对入库单进行审核。
  { label:'审核不通过', value: 20},//入库单未通过审核时的状态，通常需要重新修改或撤销入库单。
  { label:'审核通过', value: 30},//入库单经过审核并获得批准后，状态变为审核通过。准备进入执行阶段。
  { label:'未发货', value: 40},//
  // { label:'在途', value: 50},//
  { label:'部分入库', value: 60},//当入库单中的部分物料已入库，但尚有未入库的物料时，状态为部分入库。
  { label:'作废', value: 70},//当入库单发生错误或不再需要时，可以将其状态设置为作废，表示该入库单无效。
  { label:'入库完成', value: 80},//当入库单中的所有物料都已经成功入库并完成相关操作时，状态为入库完成。
]

//入库单类型
export const InboundReceiptTypes = [
  '采购入库',
  '外协入库',
  '生产入库',
  '退货入库',
]


export const InboundReceiptMaterialStatus = [
 ' ',
 '未发货',
 // '在途',
 '部分入库',
  '作废',
  '入库完成',
]

export const InboundReceiptMaterialStatusText = {
  0:'',
  40:'未发货',
  50:'在途',
  60:'部分入库',
  70:'作废',
  80:'入库完成',
}