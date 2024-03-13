<script setup lang="ts">
import CustomerPageItem from "@/components/Customer/CustomerPageItem.vue";
import {ref} from "vue";
import dayjs from "dayjs";
import { OutboundOrderRecord, OutboundOrderSummaryRequest} from "@/api/outbound/types.ts";
import {reqOutboundOrderSummary} from "@/api/outbound";
import {ElMessage} from "element-plus";
import { DateFormat} from "@/utils/time.ts";
import NP from "number-precision";

let form = ref<OutboundOrderSummaryRequest>({customer_id: '', start_date: dayjs().subtract(1, 'month').unix(), end_date: dayjs().unix()})

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


//查询
let handleSearch = async () => {
  list.value = []
  materialMap.value = new Map<string, OutboundOrderRecord[]>()
  materials.value = []

  let res = await reqOutboundOrderSummary(form.value)
  if (res.code === 200) {
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