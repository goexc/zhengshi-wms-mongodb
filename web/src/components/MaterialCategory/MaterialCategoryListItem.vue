<script setup lang="ts">
/*物料分类*/
import {MaterialCategory} from "@/api/material/types.ts";
import {onMounted, ref} from "vue";
import {CascaderValue, ElMessage, FormInstance} from "element-plus";
import {reqMaterialCategoryList} from "@/api/material";

//不带分页的物料分类下拉菜单
defineOptions({
  name: 'MaterialCategoryListItem'
})

let props = defineProps(['form'])
let formRef = ref<FormInstance>()

let categorys = ref<MaterialCategory[]>([])

//查询物料分类列表
const getMaterialCategorys = async () => {
  let res = await reqMaterialCategoryList()
  if (res.code === 200) {
    categorys.value = res.data
  } else {
    categorys.value = []
    ElMessage.error(res.msg)
  }
}

onMounted(()=>{
  getMaterialCategorys()
})
</script>

<template>
  <el-form-item label="物料分类" prop="category_id">
    <el-tree-select
        ref="formRef"
        v-model="form.category_id"
        :default-checked-keys="[form.category_id]"
        default-expand-all
        :data="categorys"
        show-checkbox
        node-key="id"
        check-strictly
        :props="{checkStrictly: true,label: 'name', value: 'id'}"
        :render-after-expand="false"
    />
  </el-form-item>
</template>

<style scoped lang="scss">

</style>