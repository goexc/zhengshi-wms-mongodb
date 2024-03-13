<script setup lang="ts">
import {PropType, ref} from "vue";
import {Sizes, Types} from "@/utils/enum";
import {useAuthStore} from "@/store/modules/auth.ts";

const authStore = useAuthStore()

defineProps({
  type: {//按钮类型
    type: String as PropType<Types>,
    default: Types.primary,
  },
  size: {//按钮尺寸
    type: String as PropType<Sizes>,
    default: 'default',
  },
  plain: {  // 是否为朴素按钮
    type: Boolean,
    default: false
  },
  circle: {  // 是否为圆形按钮
    type: Boolean,
    default: false
  },
  loading: {  // 按钮加载标识
    type: Boolean,
    default: false
  },
  disabled: {  // 按钮是否禁用
    type: Boolean,
    default: false
  },
  icon: {//按钮图标
    type: String,
    default: ''
  },
  perms: {  // 按钮权限标识，外部使用者传入
    type: String,
    default: null
  },
  text: {//按钮文本
    type: String,
    default: ''
  }
})

const label = ref<string>('') //按钮显示文本
const btnIcon = ref<string>('')
const emit = defineEmits(['action'])
const handleClick = () => {
  // console.log('子组件向父组件传递消息')
  emit('action')
}

const hasPerms = (perms: string) => {
  let res = authStore.authButtonListGet.filter((b) => b.perms === perms)
  if (res.length < 1) {
    return false
  } else {
    // let btn = res.pop() as Menu
    let btn = res.pop()!
    btnIcon.value = btn.icon
    label.value = btn.name
    return true
  }
}

</script>

<template>
  <!-- 带权限按钮 -->
  <el-button
      v-if="hasPerms(perms)"
      :size="size"
      :type="type"
      :plain="plain"
      :circle="circle"
      :loading="loading"
      :disabled="disabled"
      :icon="icon.length>0?icon:btnIcon"
      @click="handleClick">
    <template #default v-if="!circle">
      {{  text.length > 0 ? text : label }}
    </template>
  </el-button>
</template>


<style scoped lang="scss">

</style>