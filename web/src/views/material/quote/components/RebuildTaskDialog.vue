<script setup lang="ts">
import { computed } from "vue";
import { MaterialDeliveryRebuildTask } from "@/api/material/types.ts";
import { TimeFormat } from "@/utils/time.ts";

const props = defineProps<{
  modelValue: boolean;
  task: MaterialDeliveryRebuildTask | null;
}>();

const emit = defineEmits<{
  (event: "update:modelValue", value: boolean): void;
  (event: "refresh"): void;
}>();

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit("update:modelValue", value),
});

const rebuildTaskStatusText: Record<string, string> = {
  queued: "等待执行",
  running: "执行中",
  success: "执行成功",
  failed: "执行失败",
};

const rebuildTaskStatusType: Record<
  string,
  "info" | "warning" | "success" | "danger"
> = {
  queued: "info",
  running: "warning",
  success: "success",
  failed: "danger",
};

const statusLabel = computed(() => {
  const status = props.task?.status || "";
  return rebuildTaskStatusText[status] || status || "-";
});

const statusType = computed(() => {
  const status = props.task?.status || "";
  return rebuildTaskStatusType[status] || "info";
});

const taskMessage = computed(() => {
  if (!props.task) return "";
  return props.task.status === "failed"
    ? props.task.error_message
    : props.task.message;
});
</script>

<template>
  <el-dialog
    v-model="visible"
    title="重建任务详情"
    width="680px"
    destroy-on-close
    class="rebuild-task-dialog"
  >
    <el-empty v-if="!task" description="暂无重建任务" />
    <template v-else>
      <div class="task-head">
        <div>
          <span class="task-title">最近重建任务</span>
          <el-tag size="small" :type="statusType">
            {{ statusLabel }}
          </el-tag>
        </div>
        <el-button link icon="Refresh" @click="emit('refresh')">
          刷新状态
        </el-button>
      </div>
      <div class="task-grid">
        <div>
          <span>扫描出库单</span>
          <strong>{{ task.order_count }}</strong>
        </div>
        <div>
          <span>生成记录</span>
          <strong>{{ task.delivery_count }}</strong>
        </div>
        <div>
          <span>创建时间</span>
          <strong>{{ TimeFormat(task.created_at) }}</strong>
        </div>
        <div>
          <span>更新时间</span>
          <strong>{{ TimeFormat(task.updated_at) }}</strong>
        </div>
      </div>
      <div class="task-message" :class="{ error: task.status === 'failed' }">
        {{ taskMessage || "-" }}
      </div>
    </template>

    <template #footer>
      <el-button @click="visible = false">关闭</el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.task-head {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;

  > div {
    display: flex;
    gap: 10px;
    align-items: center;
    min-width: 0;
  }
}

.task-title {
  font-weight: 600;
}

.task-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(120px, 1fr));
  gap: 12px;
  margin-top: 16px;

  div {
    min-width: 0;
    padding: 10px;
    background: var(--el-fill-color-lighter);
    border-radius: 6px;
  }

  span {
    display: block;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  strong {
    display: block;
    margin-top: 4px;
    overflow: hidden;
    font-variant-numeric: tabular-nums;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.task-message {
  min-height: 40px;
  padding: 10px 12px;
  margin-top: 12px;
  color: var(--el-text-color-secondary);
  word-break: break-all;
  background: var(--el-fill-color-lighter);
  border-radius: 6px;

  &.error {
    color: var(--el-color-danger);
    background: var(--el-color-danger-light-9);
  }
}

@media (width <= 768px) {
  .task-head {
    align-items: flex-start;
  }

  .task-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
