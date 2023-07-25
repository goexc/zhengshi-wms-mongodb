
<script setup lang="ts">
import { PropType, ref} from "vue";
import {Types} from "@/utils/enum";


defineProps({
  type: {//按钮类型
    type: String as PropType<Types>,
    default: Types.primary,
  },
  loading: {  // 按钮加载标识
    type: Boolean,
    default: false
  },
  disabled: {  // 按钮是否禁用
    type: Boolean,
    default: false
  },
  perms: {  // 按钮权限标识，外部使用者传入
    type: String,
    default: null
  }
})

const label = ref<string>('') //按钮显示文本
const icon = ref<string>('')
const emit = defineEmits(['action'])
const handleClick = () => {
  // console.log('子组件向父组件传递消息')
  emit('action')
}
/*

const hasPerms = (perms:string) => {
  let res = menuStore.buttons.filter((b)=>b.perms === perms)
  if(res.length < 1){
    return false
  }else{
    let btn = res.pop() as Menu
    icon.value = btn.icon
    label.value = btn.name
    return true
  }
}
*/


</script>

<template>
  <!-- 带权限按钮 -->
  <el-link
      :type="type"
      :loading="loading"
      :icon="icon"
      class="link"
      @click="handleClick">
    {{ label }}
  </el-link>
<!--  <el-link
      v-if="hasPerms(perms)"
      :type="type"
      :loading="loading"
      :icon="icon"
      class="link"
      @click="handleClick">
    {{ label }}
  </el-link>-->
</template>



<style scoped lang="scss">
.link{
  padding-right: 10px;
}
</style>