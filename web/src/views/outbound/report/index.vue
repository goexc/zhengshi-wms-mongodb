<script setup lang="ts">
import CustomerPageItem from "@/components/Customer/CustomerPageItem.vue";
import {ref} from "vue";
import dayjs from "dayjs";
import { OutboundOrderRecord, OutboundOrderSummaryRequest} from "@/api/outbound/types.ts";
import {reqOutboundOrderSummary} from "@/api/outbound";
import {reqCustomerList} from "@/api/customer";
import {ElMessage} from "element-plus";
import { DateFormat} from "@/utils/time.ts";
import NP from "number-precision";
import ExcelJS from "exceljs";
import type {Worksheet} from "exceljs";
import type {Customer} from "@/api/customer/types.ts";

let form = ref<OutboundOrderSummaryRequest>({customer_id: '', start_date: dayjs().subtract(1, 'month').unix(), end_date: dayjs().unix()})
let customerName = ref<string>('')
let searchedForm = ref<OutboundOrderSummaryRequest>({...form.value})
let searchedCustomerName = ref<string>('')
let customerNameMap = ref<Map<string, string>>(new Map<string, string>())

const disabledDate = (time: Date) => {
  return time.getTime() > Date.now()
}

const shortcuts = [
  {
    text: '今天',
    value: new Date(),
  },
  {
    text: '昨天',
    value: () => {
      const date = new Date()
      date.setTime(date.getTime() - 3600 * 1000 * 24)
      return date
    },
  },
  {
    text: '本周一',
    value: () => {
      const date = new Date()
      date.setTime(date.getTime() - 3600 * 1000 * 24 * 7)
      return dayjs().startOf('week')
    }
  },
  {
    text: '本月初',
    value: () => {
      return dayjs().startOf('month')
    },
  },
]

//出库列表
let list = ref<OutboundOrderRecord[]>([])
//出库物料map数组
let materialMap = ref<Map<string, OutboundOrderRecord[]>>(new Map<string, OutboundOrderRecord[]>())
//出库物料二维数组
let materials = ref<OutboundOrderRecord[][]>([])

type ExportColumn = {
  header: string;
  width: number;
}

const exportColumns: ExportColumn[] = [
  {header: '序号', width: 8},
  {header: '产品型号', width: 24},
  {header: '名称', width: 18},
  {header: '规格', width: 24},
  {header: '签收日期', width: 14},
  {header: '出库单编号', width: 20},
  {header: '单价', width: 12},
  {header: '数量', width: 12},
  {header: '总数量', width: 12},
  {header: '金额', width: 14},
  {header: '总金额', width: 14},
]

const compareRecord = (a: OutboundOrderRecord, b: OutboundOrderRecord) => {
  return (a.receipt_date || Number.MAX_SAFE_INTEGER) - (b.receipt_date || Number.MAX_SAFE_INTEGER)
      || a.order_code.localeCompare(b.order_code)
      || a.index - b.index
}

const amount = (row: OutboundOrderRecord) => {
  return NP.strip(NP.times(row.quantity || 0, row.price || 0))
}

const totalQuantity = (rows: OutboundOrderRecord[]) => {
  return NP.strip(rows.reduce((total, row) => NP.plus(total, row.quantity || 0), 0))
}

const totalAmount = (rows: OutboundOrderRecord[]) => {
  return NP.strip(rows.reduce((total, row) => NP.plus(total, amount(row)), 0))
}

const groupBy = (rows: OutboundOrderRecord[], keyGetter: (row: OutboundOrderRecord) => string) => {
  const groupMap = new Map<string, OutboundOrderRecord[]>()
  rows.forEach(row => {
    const key = keyGetter(row)
    groupMap.set(key, [...(groupMap.get(key) || []), row])
  })
  return Array.from(groupMap.values())
}

const handleCustomerChange = (customer?: Customer) => {
  customerName.value = customer?.name || ''
  if (customer?.id && customer.name) {
    customerNameMap.value.set(customer.id, customer.name)
  }
}

const resolveCustomerName = async (customerId: string) => {
  if (!customerId) {
    return ''
  }
  const cachedName = customerNameMap.value.get(customerId)
  if (cachedName) {
    return cachedName
  }

  const res = await reqCustomerList()
  if (res.code !== 200) {
    ElMessage.error(res.msg)
    return ''
  }

  res.data.list.forEach(customer => {
    customerNameMap.value.set(customer.id, customer.name)
  })
  return customerNameMap.value.get(customerId) || ''
}

const buildReportTitle = () => {
  return `x司（${DateFormat(searchedForm.value.start_date)} 至 ${DateFormat(searchedForm.value.end_date)}）`
}

const buildFileName = (type: string) => {
  return `出库报表-${type}-${DateFormat(searchedForm.value.start_date)}-${DateFormat(searchedForm.value.end_date)}.xlsx`
}

const createWorksheet = (sheetName: string) => {
  const workbook = new ExcelJS.Workbook()
  const worksheet = workbook.addWorksheet(sheetName)

  workbook.creator = 'zhengshi-wms'
  worksheet.properties.defaultRowHeight = 22
  exportColumns.forEach((column, index) => {
    worksheet.getColumn(index + 1).width = column.width
  })

  worksheet.mergeCells(1, 1, 1, exportColumns.length)
  worksheet.mergeCells(2, 1, 2, exportColumns.length)
  worksheet.getCell(1, 1).value = buildReportTitle()
  worksheet.getCell(2, 1).value = `客户：${searchedCustomerName.value || '全部客户'}`
  worksheet.getCell(1, 1).font = {bold: true, size: 16}
  worksheet.getCell(1, 1).alignment = {horizontal: 'center', vertical: 'middle'}
  worksheet.getCell(2, 1).font = {bold: true, size: 12}
  worksheet.getCell(2, 1).alignment = {horizontal: 'left', vertical: 'middle'}
  worksheet.getRow(1).height = 28
  worksheet.getRow(2).height = 24

  worksheet.addRow(exportColumns.map(column => column.header))
  worksheet.getRow(3).eachCell(cell => {
    cell.font = {bold: true}
    cell.alignment = {horizontal: 'center', vertical: 'middle'}
    cell.fill = {type: 'pattern', pattern: 'solid', fgColor: {argb: 'FFD9EAF7'}}
    cell.border = {
      top: {style: 'thin'},
      left: {style: 'thin'},
      bottom: {style: 'thin'},
      right: {style: 'thin'},
    }
  })
  worksheet.views = [{state: 'frozen', ySplit: 3}]

  return {workbook, worksheet}
}

const mergeGroupCells = (worksheet: Worksheet, startRow: number, endRow: number, columns: number[]) => {
  if (endRow <= startRow) {
    return
  }
  columns.forEach(column => {
    worksheet.mergeCells(startRow, column, endRow, column)
  })
}

const styleDataRows = (worksheet: Worksheet) => {
  for (let rowIndex = 4; rowIndex <= worksheet.rowCount; rowIndex++) {
    const row = worksheet.getRow(rowIndex)
    row.eachCell(cell => {
      cell.alignment = {horizontal: 'center', vertical: 'middle', wrapText: true}
      cell.border = {
        top: {style: 'thin'},
        left: {style: 'thin'},
        bottom: {style: 'thin'},
        right: {style: 'thin'},
      }
    })
    row.getCell(7).numFmt = '#,##0.0000'
    row.getCell(8).numFmt = '#,##0.####'
    row.getCell(9).numFmt = '#,##0.####'
    row.getCell(10).numFmt = '#,##0.0000'
    row.getCell(11).numFmt = '#,##0.0000'
  }
}

const saveWorkbook = async (workbook: ExcelJS.Workbook, fileName: string) => {
  const buffer = await workbook.xlsx.writeBuffer()
  const blob = new Blob([buffer as BlobPart], {type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'})
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = fileName
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.setTimeout(() => URL.revokeObjectURL(url), 0)
}

const ensureExportable = () => {
  if (list.value.length === 0) {
    ElMessage.warning('暂无可导出数据')
    return false
  }
  return true
}

const exportByOrder = async () => {
  if (!ensureExportable()) {
    return
  }

  const {workbook, worksheet} = createWorksheet('按单据导出')
  const sortedRows = [...list.value].sort(compareRecord)
  const orderGroups = groupBy(sortedRows, row => row.order_code)

  orderGroups.forEach((group, groupIndex) => {
    const sortedGroup = [...group].sort(compareRecord)
    const startRow = worksheet.rowCount + 1
    const groupTotalQuantity = totalQuantity(sortedGroup)
    const groupTotalAmount = totalAmount(sortedGroup)

    sortedGroup.forEach(row => {
      worksheet.addRow([
        groupIndex + 1,
        row.model,
        row.name,
        row.specification,
        DateFormat(row.receipt_date),
        row.order_code,
        row.price,
        row.quantity,
        groupTotalQuantity,
        amount(row),
        groupTotalAmount,
      ])
    })
    mergeGroupCells(worksheet, startRow, worksheet.rowCount, [1, 5, 6, 9, 11])
  })

  styleDataRows(worksheet)
  await saveWorkbook(workbook, buildFileName('按单据'))
}

const exportByMaterial = async () => {
  if (!ensureExportable()) {
    return
  }

  const {workbook, worksheet} = createWorksheet('按物料导出')
  const materialGroups = groupBy([...list.value], row => row.material_id)
      .sort((a, b) => {
        return a[0].model.localeCompare(b[0].model)
            || a[0].name.localeCompare(b[0].name)
            || a[0].specification.localeCompare(b[0].specification)
      })

  materialGroups.forEach((group, groupIndex) => {
    const sortedGroup = [...group].sort(compareRecord)
    const startRow = worksheet.rowCount + 1
    const groupTotalQuantity = totalQuantity(sortedGroup)
    const groupTotalAmount = totalAmount(sortedGroup)

    sortedGroup.forEach(row => {
      worksheet.addRow([
        groupIndex + 1,
        row.model,
        row.name,
        row.specification,
        DateFormat(row.receipt_date),
        row.order_code,
        row.price,
        row.quantity,
        groupTotalQuantity,
        amount(row),
        groupTotalAmount,
      ])
    })
    mergeGroupCells(worksheet, startRow, worksheet.rowCount, [1, 2, 3, 4, 9, 11])
  })

  styleDataRows(worksheet)
  await saveWorkbook(workbook, buildFileName('按物料'))
}


//查询
let handleSearch = async () => {
  const queryForm = {...form.value}
  list.value = []
  materialMap.value = new Map<string, OutboundOrderRecord[]>()
  materials.value = []

  let res = await reqOutboundOrderSummary(queryForm)
  if (res.code === 200) {
    searchedForm.value = queryForm
    searchedCustomerName.value = await resolveCustomerName(queryForm.customer_id)
    list.value = res.data.sort((a: OutboundOrderRecord, b: OutboundOrderRecord) => (a.model > b.model) ? -1 : 1)
    list.value.forEach((item: OutboundOrderRecord) => {
      if (!materialMap.value.has(item.material_id)) {
        materialMap.value.set(item.material_id, [item])
      }else{
        let material = [...materialMap.value.get(item.material_id)! as OutboundOrderRecord[], item].sort((x: OutboundOrderRecord, y: OutboundOrderRecord) => x.receipt_date - y.receipt_date)
        materialMap.value.set(item.material_id, material)
      }
    })

    materials.value = Array.from(materialMap.value.values())
    console.log('materials:', materials.value)
  } else {
    list.value = []
    ElMessage.error(res.msg)
  }
}

</script>

<template>
  <div>
    <el-card>
      <el-form
          :model="form"
          inline
          label-width="60px"
          size="default"
          style="display: flex; flex-wrap: wrap;"
      >
        <CustomerPageItem
            :form="form"
            @change="handleCustomerChange"
        />
        <el-form-item
            label="起始日期"
            prop="start_date"
        >
          <el-date-picker
              v-model.number="form.start_date"
              type="date"
              placeholder="请选择起始日期"
              size="default"
              value-format="X"
              :disabled-date="disabledDate"
              :shortcuts="shortcuts"
              start-placeholder="起始日期"
              end-placeholder="截止日期"
          />

        </el-form-item>
        <el-form-item
            label="截止日期"
            prop="end_date"
        >
          <el-date-picker
              v-model.number="form.end_date"
              type="date"
              placeholder="请选择截止日期"
              size="default"
              value-format="X"
              :disabled-date="disabledDate"
              :shortcuts="shortcuts"
              start-placeholder="起始日期"
              end-placeholder="截止日期"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" plain size="default" icon="search" @click="handleSearch">查询</el-button>
          <el-button type="success" plain size="default" icon="download" @click="exportByOrder">按单据导出</el-button>
          <el-button type="success" plain size="default" icon="download" @click="exportByMaterial">按物料导出</el-button>
        </el-form-item>
      </el-form>
    </el-card>
<!--    <el-card class="m-y-2">-->
      <div class="m-y-2 content-between">
        <div><span class="content-between-data">{{ Array.from(new Set(list.map(item => item.code)))?.length }}</span> 单
        </div>
        <div><span class="content-between-data">{{
            list.map(one => one.quantity).reduce((total, value) => total + value, 0)
          }}</span> 件
        </div>
        <div><span class="content-between-data">{{NP.strip(list.reduce((total, item) => NP.plus(total + NP.times(item.quantity, item.price)), 0))}}</span> 元
        </div>
      </div>

    <div v-for="(material,index) in materials" :key="index" class="receipt">
      <table style="align-content: center">
        <thead>
        <tr v-if="index===0">
          <th style="width:50px">序号</th>
          <th style="width:200px">产品</th>
          <th style="width:100px;min-width:100px">名称</th>
          <th style="width:200px;min-width:200px">尺寸</th>
          <th style="min-width:120px">出库日期</th>
          <th style="min-width:120px">签收日期</th>
          <th style="width:140px;min-width:140px">出库单编号</th>
          <th style="min-width:90px">单价</th>
          <th style="min-width:100px">数量</th>
          <th style="min-width:100px">总数量</th>
          <th style="min-width:140px">金额</th>
          <th style="min-width:140px">总金额</th>
        </tr>
        </thead>
        <tbody>
        <tr v-for="(row,idx) in material" :key="idx">
          <td :rowspan="material.length" v-if="idx===0" style="width:50px;text-align: center">{{ index+1}}</td>
          <!--            <td :rowspan="material.length" v-if="idx===0" style="width:600px">{{ row.material_model }} / {{row.material_name}} / {{row.material_specs}}</td>-->
          <td :rowspan="material.length" v-if="idx===0" style="width:200px">{{ row.model}}</td>
          <td :rowspan="material.length" v-if="idx===0" style="width:100px;min-width:100px">{{ row.name }}</td>
          <td :rowspan="material.length" v-if="idx===0" style="width:200px;min-width:200px">{{ row.specification}}</td>
          <td style="min-width:120px">{{ DateFormat(row.departure_date) }}</td>
          <td style="min-width:120px">{{ DateFormat(row.receipt_date) }}</td>
          <td style="width:140px;min-width:140px;">{{ row.order_code }}</td>
          <td style="min-width:90px">{{ row.price }}</td>
          <td style="min-width:100px">{{ row.quantity }}</td>
          <td style="min-width:100px" :rowspan="material.length" v-if="idx===0">{{ material.map(one=>one.quantity).reduce((total, value)=>total+value, 0) }}</td>
          <td style="min-width:140px">{{ (row.quantity * row.price).toFixed(4) }}</td>
          <td style="min-width:140px" :rowspan="material.length" v-if="idx===0">￥{{
              material.map(one => one.price * one.quantity).reduce((total, value) => total + value, 0).toFixed(4)
            }}
          </td>
        </tr>
        </tbody>
      </table>
    </div>


<!--    </el-card>-->
  </div>
</template>

<style scoped lang="scss">
.content-between {
  display: flex;
  justify-content: space-between;
  background-color: #79bbff;
  padding: 40px 20px;
  color: #ffffff;

  &-data {
    font-size: 42px;
  }
}

.receipt {
  padding: 10px 20px 10px 20px;
  //margin-bottom: 40px;
  //border: 2px dashed #1e80ff;
  background-color: rgba(215, 209, 35, 0.4);
}

table {
  width: 100%;
  border-collapse: collapse;
  border-width: 0 !important;
  border: solid #1e80ff;
}

table th, table td {
  border: 1px solid #1e80ff;
  padding: 4px;
}
</style>
