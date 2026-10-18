import {computed, ref} from 'vue';
import type {baseResponse} from '@/api/types.ts';
import type {OutboundRemovePreview, OutboundRemovePreviewResponse} from '@/api/outbound/types.ts';

interface RemovalApi {
  preview: (id: string) => Promise<OutboundRemovePreviewResponse>;
  remove: (id: string, token: string) => Promise<baseResponse>;
}

// A generation belongs to one open dialog/order. Late responses cannot update another order.
export function useOutboundRemoval(api: RemovalApi, onSuccess: (message: string) => void, onMissing: () => void = () => {}) {
  const preview = ref<OutboundRemovePreview | null>(null);
  const loading = ref(false);
  const submitting = ref(false);
  const error = ref('');
  let orderId = '';
  let generation = 0;

  const canSubmit = computed(() => !loading.value && !submitting.value &&
    preview.value?.allowed === true && !!preview.value.preview_token);

  function clear() {
    generation++;
    orderId = '';
    preview.value = null;
    loading.value = false;
    submitting.value = false;
    error.value = '';
  }

  async function refresh() {
    if (!orderId || loading.value || submitting.value) return;
    const ticket = ++generation;
    const id = orderId;
    preview.value = null;
    error.value = '';
    loading.value = true;
    try {
      const result = await api.preview(id);
      if (ticket !== generation) return;
      // Only a business 404 from this endpoint proves the selected order is absent.
      // A failed HTTP route or an uncertain DELETE response cannot prove that.
      if (result.code === 404) {
        clear();
        onMissing();
        return;
      }
      if (result.code !== 200) {
        error.value = result.msg || '删除预览失败，请重新预览';
        return;
      }
      const data = result.data;
      if (!data || data.id !== id || !Array.isArray(data.inventorys) ||
          (data.allowed && !data.preview_token)) {
        error.value = '删除预览不完整，请重新预览';
        return;
      }
      preview.value = data;
    } catch {
      if (ticket === generation) error.value = '删除预览失败，请检查网络后重新预览';
    } finally {
      if (ticket === generation) loading.value = false;
    }
  }

  async function select(id: string) {
    clear();
    orderId = id;
    await refresh();
  }

  async function submit() {
    if (!canSubmit.value || !preview.value) return;
    const ticket = generation;
    const id = orderId;
    const token = preview.value.preview_token;
    submitting.value = true;
    error.value = '';
    try {
      const result = await api.remove(id, token);
      if (ticket !== generation) return;
      // Never reuse a submitted token, even after a failed or uncertain response.
      preview.value = null;
      if (result.code === 200) {
        onSuccess(result.msg || '删除成功');
      } else {
        error.value = result.code === 409
          ? '单据或关联数据已变化，原预览已失效，请重新预览后确认'
          : `${result.msg || '删除失败'}，请重新预览后确认`;
      }
    } catch (cause) {
      if (ticket !== generation) return;
      preview.value = null;
      const status = (cause as {response?: {status?: number}})?.response?.status;
      error.value = status === 409
        ? '单据或关联数据已变化，原预览已失效，请重新预览后确认'
        : '删除结果未确认，请重新预览或刷新列表核对；不会自动重复提交';
    } finally {
      if (ticket === generation) submitting.value = false;
    }
  }

  return {preview, loading, submitting, error, canSubmit, select, refresh, submit, clear};
}
