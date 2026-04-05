<template>
  <el-card>
    <template #header>
      <div class="card-header">
        <span>极速出库单录入</span>
      </div>
    </template>
    
    <el-form :model="formData" label-width="120px" ref="formRef" >
      <el-row :gutter="20">
        <el-col :span="8">
          <el-form-item label="出库单号" prop="code" :rules="[{ required: true, message: '请输入单号', trigger: 'blur' }]">
            <el-input v-model="formData.code" placeholder="输入出库单号或自动生成" />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="出库类型" prop="type" :rules="[{ required: true, message: '请选择类型', trigger: 'change' }]">
            <el-select v-model="formData.type" placeholder="出库类型" style="width: 100%">
              <el-option label="销售出库" value="销售出库" />
              <el-option label="样品出库" value="样品出库" />
              <el-option label="退货出库" value="退货出库" />
              <el-option label="生产用料出库" value="生产用料出库" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="客户ID" prop="customer_id" :rules="[{ required: true, message: '请输入客户ID', trigger: 'blur' }]">
            <el-input v-model="formData.customer_id" placeholder="客户ObjectID" />
          </el-form-item>
        </el-col>
      </el-row>
      
      <el-divider>时间节点配置 (默认使用当前时间)</el-divider>
      
      <el-row :gutter="20">
        <el-col :span="8">
          <el-form-item label="发货时间" prop="departure_time" :rules="[{ required: true, message: '发货时间必填' }]">
            <el-date-picker v-model="formData.departure_time" type="datetime" value-format="X" placeholder="选择发货时间" style="width: 100%" />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="拣货时间" prop="picking_time">
            <el-date-picker v-model="formData.picking_time" type="datetime" value-format="X" placeholder="选择拣货时间" style="width: 100%" />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="打包时间" prop="packing_time">
            <el-date-picker v-model="formData.packing_time" type="datetime" value-format="X" placeholder="选择打包时间" style="width: 100%" />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="称重时间" prop="weighing_time">
            <el-date-picker v-model="formData.weighing_time" type="datetime" value-format="X" placeholder="选择称重时间" style="width: 100%" />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="签收时间" prop="receipt_time">
            <el-date-picker v-model="formData.receipt_time" type="datetime" value-format="X" placeholder="选择签收时间" style="width: 100%" />
          </el-form-item>
        </el-col>
      </el-row>

      <el-divider>物料明细</el-divider>
      <div style="margin-bottom: 10px;">
        <el-button type="primary" @click="handleAddMaterial">添加物料</el-button>
      </div>
      <el-table :data="formData.materials" border style="width: 100%">
        <el-table-column label="物料ID">
          <template #default="{ row }">
            <el-input v-model="row.material_id" placeholder="物料ObjectID" />
          </template>
        </el-table-column>
        <el-table-column label="出库数量" width="200">
          <template #default="{ row }">
            <el-input-number v-model="row.quantity" :min="1" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" align="center">
          <template #default="{ $index }">
            <el-button type="danger" @click="handleRemoveMaterial($index)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      
      <div style="margin-top: 30px; text-align: center;">
        <el-button type="success" size="large" @click="submitForm">极速出库提交</el-button>
      </div>
    </el-form>
  </el-card>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { reqFastDepartureOutboundOrder } from '@/api/outbound/index'
import type { FastOutboundRequest, FastOutboundMaterial } from '@/api/outbound/types'

const formRef = ref()

const nowSeconds = Math.floor(Date.now() / 1000)

const formData = reactive<FastOutboundRequest>({
  code: `FAST_${Date.now()}`,
  type: '生产用料出库',
  customer_id: '',
  departure_time: nowSeconds,
  picking_time: nowSeconds,
  packing_time: nowSeconds,
  weighing_time: nowSeconds,
  receipt_time: nowSeconds,
  materials: [] as FastOutboundMaterial[]
})

const handleAddMaterial = () => {
  formData.materials.push({
    material_id: '',
    quantity: 1
  })
}

const handleRemoveMaterial = (index: number) => {
  formData.materials.splice(index, 1)
}

const submitForm = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid: boolean) => {
    if (valid) {
      if (formData.materials.length === 0) {
        ElMessage.warning('请添加至少一项物料明细')
        return
      }
      for (let i = 0; i < formData.materials.length; i++) {
        if (!formData.materials[i].material_id) {
          ElMessage.warning(`第 ${i + 1} 行物料ID为空`)
          return
        }
      }
      try {
        const res = await reqFastDepartureOutboundOrder(formData)
        if (res.code === 200) {
          ElMessage.success('极速出库成功！若库存不足已自动补录产线入库。')
          formData.code = `FAST_${Date.now()}`
          formData.materials = []
        } else {
          ElMessage.error(res.msg || '出库失败')
        }
      } catch (e: any) {
        ElMessage.error(e.message || '网络错误')
      }
    }
  })
}
</script>

<style scoped>
.card-header {
  font-weight: bold;
  font-size: 18px;
}
</style>
