<template>
  <el-card>
    <template #header>
      <div class="card-header">
        <span>极速出库</span>
        <el-button type="primary" @click="openDialog">新增极速出库单</el-button>
      </div>
    </template>

    <el-empty description="点击新增极速出库单，在弹窗中完成录入" />
  </el-card>

  <el-dialog
    v-model="dialogVisible"
    title="极速出库单录入"
    width="980px"
    destroy-on-close
    @closed="resetForm"
  >
    <el-form ref="formRef" :model="formData" label-width="120px">
      <el-row :gutter="20">
        <el-col :span="8">
          <el-form-item
            label="出库单号"
            prop="code"
            :rules="[{ required: true, message: '请输入单号', trigger: 'blur' }]"
          >
            <el-input v-model="formData.code" placeholder="输入出库单号或自动生成" />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item
            label="出库类型"
            prop="type"
            :rules="[{ required: true, message: '请选择类型', trigger: 'change' }]"
          >
            <el-select v-model="formData.type" placeholder="出库类型" style="width: 100%">
              <el-option label="销售出库" value="销售出库" />
              <el-option label="样品出库" value="样品出库" />
              <el-option label="赠品出库" value="赠品出库" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item
            label="客户ID"
            prop="customer_id"
            :rules="[{ required: true, message: '请输入客户ID', trigger: 'blur' }]"
          >
            <el-input v-model="formData.customer_id" placeholder="客户 ObjectID" />
          </el-form-item>
        </el-col>
      </el-row>

      <el-divider>时间节点配置</el-divider>

      <el-row :gutter="20">
        <el-col :span="8">
          <el-form-item
            label="出库时间"
            prop="departure_time"
            :rules="[{ required: true, message: '出库时间必填', trigger: 'change' }]"
          >
            <el-date-picker
              v-model="formData.departure_time"
              type="datetime"
              value-format="X"
              placeholder="选择出库时间"
              style="width: 100%"
            />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="拣货时间" prop="picking_time">
            <el-date-picker
              v-model="formData.picking_time"
              type="datetime"
              value-format="X"
              placeholder="默认当前时间"
              style="width: 100%"
            />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="打包时间" prop="packing_time">
            <el-date-picker
              v-model="formData.packing_time"
              type="datetime"
              value-format="X"
              placeholder="可选"
              style="width: 100%"
            />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="称重时间" prop="weighing_time">
            <el-date-picker
              v-model="formData.weighing_time"
              type="datetime"
              value-format="X"
              placeholder="可选"
              style="width: 100%"
            />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="签收时间" prop="receipt_time">
            <el-date-picker
              v-model="formData.receipt_time"
              type="datetime"
              value-format="X"
              placeholder="默认当前时间"
              style="width: 100%"
            />
          </el-form-item>
        </el-col>
      </el-row>

      <el-divider>物料明细</el-divider>

      <div class="table-actions">
        <el-button type="primary" @click="handleAddMaterial">添加物料</el-button>
      </div>

      <el-table :data="formData.materials" border style="width: 100%">
        <el-table-column label="物料ID">
          <template #default="{ row, $index }">
            <el-input
              v-model="row.material_id"
              placeholder="物料 ObjectID"
              @change="handleMaterialChange(row.material_id, $index)"
            />
          </template>
        </el-table-column>
        <el-table-column label="出库价" width="220">
          <template #default="{ row }">
            <el-popover
              placement="right"
              title="历史价格"
              :width="300"
              trigger="hover"
              @before-enter="getPrices(row.material_id)"
            >
              <template #reference>
                <el-input-number
                  v-model="row.price"
                  :controls="false"
                  :min="0"
                  :precision="3"
                  :step="100"
                  style="width: 100%"
                />
              </template>
              <el-tag
                v-for="(one, idx) in prices"
                :key="idx"
                class="price-tag"
                size="default"
                @click="row.price = one.price"
              >
                {{ one.price }} / {{ one.customer_name }}
              </el-tag>
              <el-text v-if="prices.length === 0" size="small">暂无</el-text>
            </el-popover>
          </template>
        </el-table-column>
        <el-table-column label="出库数量" width="220">
          <template #default="{ row }">
            <el-input-number v-model="row.quantity" :min="1" :precision="0" style="width: 100%" />
          </template>
        </el-table-column>
        <el-table-column label="金额" width="140" align="center">
          <template #default="{ row }">
            {{ (Number(row.price || 0) * Number(row.quantity || 0)).toFixed(3) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" align="center">
          <template #default="{ $index }">
            <el-button type="danger" @click="handleRemoveMaterial($index)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-form>

    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="success" :loading="submitting" @click="submitForm">提交极速出库</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { reqMaterialPrices } from '@/api/material'
import type { MaterialPrice } from '@/api/material/types'
import { reqFastDepartureOutboundOrder } from '@/api/outbound/index'
import type { FastOutboundMaterial, FastOutboundRequest } from '@/api/outbound/types'

const formRef = ref()
const dialogVisible = ref(false)
const submitting = ref(false)
const prices = ref<MaterialPrice[]>([])

const nowSeconds = () => Math.floor(Date.now() / 1000)

const createFormData = (): FastOutboundRequest => {
  const now = nowSeconds()
  return {
    code: `FAST_${Date.now()}`,
    type: '销售出库',
    customer_id: '',
    departure_time: now,
    picking_time: now,
    packing_time: undefined,
    weighing_time: undefined,
    receipt_time: now,
    materials: [] as FastOutboundMaterial[]
  }
}

const formData = reactive<FastOutboundRequest>(createFormData())

const openDialog = () => {
  dialogVisible.value = true
}

const resetForm = () => {
  Object.assign(formData, createFormData())
}

const handleAddMaterial = () => {
  formData.materials.push({
    material_id: '',
    price: 0,
    quantity: 1
  })
}

const handleRemoveMaterial = (index: number) => {
  formData.materials.splice(index, 1)
}

const findDuplicateMaterial = () => {
  const seen = new Map<string, number>()
  for (let i = 0; i < formData.materials.length; i++) {
    const materialId = formData.materials[i].material_id.trim()
    if (!materialId) continue

    const firstIndex = seen.get(materialId)
    if (firstIndex !== undefined) {
      return {
        firstIndex,
        duplicateIndex: i
      }
    }
    seen.set(materialId, i)
  }
  return null
}

const handleMaterialChange = (value: string, index: number) => {
  const materialId = value.trim()
  formData.materials[index].material_id = materialId
  if (!materialId) return

  const duplicateIndex = formData.materials.findIndex((item, idx) => (
    idx !== index && item.material_id.trim() === materialId
  ))
  if (duplicateIndex >= 0) {
    ElMessage.warning(`第 ${index + 1} 行物料已在第 ${duplicateIndex + 1} 行选择，出库物料不允许重复`)
    formData.materials[index].material_id = ''
  }
}

const normalizeTime = (value?: number) => {
  if (!value) return undefined
  return Number(value)
}

const validateTimeChain = () => {
  const now = nowSeconds()
  const departureTime = Number(formData.departure_time)
  const pickingTime = Number(formData.picking_time || departureTime)
  const receiptTime = Number(formData.receipt_time || now)
  const chain = [
    { label: '拣货时间', value: pickingTime },
    ...(formData.packing_time ? [{ label: '打包时间', value: Number(formData.packing_time) }] : []),
    ...(formData.weighing_time ? [{ label: '称重时间', value: Number(formData.weighing_time) }] : []),
    { label: '出库时间', value: departureTime },
    { label: '签收时间', value: receiptTime }
  ]

  for (const one of chain) {
    if (one.value > now) {
      ElMessage.warning(`${one.label}不能超过当前时间`)
      return false
    }
  }

  for (let i = 1; i < chain.length; i++) {
    if (chain[i].value < chain[i - 1].value) {
      ElMessage.warning(`${chain[i].label}不能早于${chain[i - 1].label}`)
      return false
    }
  }
  return true
}

const getPrices = async (materialId: string) => {
  prices.value = []
  if (!materialId) return
  if (!formData.customer_id) {
    ElMessage.warning('请先填写客户ID')
    return
  }

  const res = await reqMaterialPrices(materialId, formData.customer_id)
  if (res.code === 200) {
    prices.value = res.data || []
  } else {
    ElMessage.error(res.msg)
  }
}

const submitForm = async () => {
  if (!formRef.value || submitting.value) return

  await formRef.value.validate(async (valid: boolean) => {
    if (!valid) return

    if (formData.materials.length === 0) {
      ElMessage.warning('请添加至少一项物料明细')
      return
    }

    for (let i = 0; i < formData.materials.length; i++) {
      if (!formData.materials[i].material_id) {
        ElMessage.warning(`第 ${i + 1} 行物料ID为空`)
        return
      }
      if (Number(formData.materials[i].price) < 0) {
        ElMessage.warning(`第 ${i + 1} 行物料价格不能小于0`)
        return
      }
    }

    const duplicate = findDuplicateMaterial()
    if (duplicate) {
      ElMessage.warning(`第 ${duplicate.duplicateIndex + 1} 行物料与第 ${duplicate.firstIndex + 1} 行重复，出库物料不允许重复`)
      return
    }

    if (!validateTimeChain()) return

    try {
      submitting.value = true
      const departureTime = Number(formData.departure_time)
      const payload: FastOutboundRequest = {
        ...formData,
        departure_time: departureTime,
        picking_time: normalizeTime(formData.picking_time) || departureTime,
        packing_time: normalizeTime(formData.packing_time),
        weighing_time: normalizeTime(formData.weighing_time),
        receipt_time: normalizeTime(formData.receipt_time),
        materials: formData.materials.map(item => ({
          material_id: item.material_id.trim(),
          price: Number(item.price || 0),
          quantity: Number(item.quantity)
        }))
      }
      const res = await reqFastDepartureOutboundOrder(payload)
      if (res.code === 200) {
        ElMessage.success('极速出库成功')
        dialogVisible.value = false
      } else {
        ElMessage.error(res.msg || '出库失败')
      }
    } catch (e: any) {
      ElMessage.error(e.message || '网络错误')
    } finally {
      submitting.value = false
    }
  })
}
</script>

<style scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 18px;
  font-weight: bold;
}

.table-actions {
  margin-bottom: 10px;
}

.price-tag {
  margin: 4px;
  cursor: pointer;
}
</style>
