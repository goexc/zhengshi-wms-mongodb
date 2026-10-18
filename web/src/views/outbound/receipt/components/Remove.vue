<script setup lang="ts">
import {onBeforeUnmount, watch} from 'vue';
import {ElMessage} from 'element-plus';
import {reqOutboundRemovePreview, reqRemoveOutboundOrder} from '@/api/outbound';
import {useOutboundRemoval} from './useOutboundRemoval';

const props = defineProps<{id: string; code: string}>();
const emit = defineEmits<{
  cancel: [];
  success: [];
  missing: [];
  busy: [value: boolean];
}>();
const {preview, loading, submitting, error, canSubmit, select, refresh, submit, clear} =
  useOutboundRemoval({preview: reqOutboundRemovePreview, remove: reqRemoveOutboundOrder}, (message) => {
    ElMessage.success(message);
    emit('success');
  }, () => {
    ElMessage.info('该出库单已不存在，列表已更新');
    emit('missing');
  });

watch(() => props.id, (id) => { void select(id); }, {immediate: true, flush: 'sync'});
watch(submitting, (value) => emit('busy', value), {flush: 'sync'});
onBeforeUnmount(clear);
const money = (value: number) => Number.isFinite(value) ? `￥${value.toFixed(4)}` : '未提供';
</script>

<template>
  <div class="remove-preview" v-loading="loading" :aria-busy="loading || submitting">
    <el-alert
      type="warning"
      :closable="false"
      show-icon
      :title="`确认删除出库单 ${code}？`"
      description="删除将移除该单的应收及核价调整流水，客户回款保留，并按下方预览处理库存。删除后无法在页面撤销。"
    />
    <el-alert v-if="error" type="error" :closable="false" show-icon :title="error" role="alert" />
    <template v-if="preview">
      <el-descriptions :column="2" border>
        <el-descriptions-item label="出库单号">{{ preview.code }}</el-descriptions-item>
        <el-descriptions-item label="当前状态">{{ preview.status }}</el-descriptions-item>
        <el-descriptions-item label="客户" :span="2">{{ preview.customer_name || '—' }}</el-descriptions-item>
        <el-descriptions-item label="删除应收及核价调整流水">{{ preview.transaction_count }} 条</el-descriptions-item>
        <el-descriptions-item label="净减少应收">{{ money(preview.receivable_reduction) }}</el-descriptions-item>
        <el-descriptions-item label="当前应收">{{ money(preview.receivable_before) }}</el-descriptions-item>
        <el-descriptions-item label="删除后应收">{{ money(preview.receivable_after) }}</el-descriptions-item>
        <el-descriptions-item label="当前贷项余额">{{ money(preview.credit_before) }}</el-descriptions-item>
        <el-descriptions-item label="删除后贷项余额">{{ money(preview.credit_after) }}</el-descriptions-item>
      </el-descriptions>
      <el-alert v-if="preview.credit_after > 0" type="info" :closable="false" show-icon title="保留的回款等贷项超过剩余应收时，差额计入客户贷项余额。删除出库单不会退款。" />
      <el-alert v-if="!preview.allowed" type="error" :closable="false" show-icon :title="preview.block_reason || '当前出库单不可删除'" />
      <div>
        <h4>库存处理明细</h4>
        <el-table :data="preview.inventorys" border size="small" max-height="300" empty-text="无需恢复或释放库存">
          <el-table-column label="物料" min-width="145">
            <template #default="{row}">
              <div>{{ row.name }}</div>
              <el-text type="info" size="small">{{ row.model }}</el-text>
            </template>
          </el-table-column>
          <el-table-column prop="warehouse_name" label="仓库" min-width="100" />
          <el-table-column prop="inventory_id" label="库存批次 ID" min-width="205" />
          <el-table-column prop="unit" label="单位" width="65" />
          <el-table-column prop="release_locked_quantity" label="释放锁定数量" min-width="110" align="right" />
          <el-table-column prop="restore_available_quantity" label="恢复可用数量" min-width="110" align="right" />
        </el-table>
      </div>
    </template>
    <div class="remove-actions">
      <el-button :disabled="submitting" @click="emit('cancel')">取消</el-button>
      <el-button :disabled="loading || submitting" :loading="loading" @click="refresh">重新预览</el-button>
      <el-button type="danger" :disabled="!canSubmit" :loading="submitting" @click="submit">确认删除</el-button>
    </div>
  </div>
</template>

<style scoped>
.remove-preview {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 120px;
}
h4 {
  margin: 0 0 8px;
}
.remove-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}
.remove-actions .el-button {
  margin-left: 0;
}
</style>
