<script setup lang="ts">

import {onMounted, ref} from "vue";

//带分页的物料下拉菜单
defineOptions({
  name: 'MaterialPageItem'
})
//物料列表
import {Material, MaterialsRequest} from "@/api/material/types.ts";
import {reqMaterials} from "@/api/material";
import {ElMessage} from "element-plus";

defineProps(['form'])

const initMaterialsForm = () => {
  return {
    page: 1,
    size: 10,
    name: '',
    image: '',
    material: '',
    specification: '',
    model: '',
    surface_treatment: '',
    strength_grade: '',
  }
}
let materials = ref<Material[]>([])
let materialsForm = ref<MaterialsRequest>(initMaterialsForm())
let materialsTotal = ref<number>(0)

//查询物料列表
const getMaterials = async () => {
  let res = await reqMaterials(materialsForm.value)
  if (res.code === 200) {
    materials.value = res.data.list
    materialsTotal.value = res.data.total
  } else {
    materials.value = []
    materialsTotal.value = 0
    ElMessage.error(res.msg)
  }
}
onMounted(()=>{
  getMaterials()
})
</script>

<template>
  <el-form-item label="物料列表" prop="material_id">
    <el-select filterable v-model.trim="form.material_id" autocomplete="off" clearable>
      <el-pagination
          v-model:page-size="materialsForm.size"
          v-model:current-page="materialsForm.page"
          :total="materialsTotal"
          style="width: 100%"
          layout="prev, pager, next"
          @current-change="getMaterials"
      />
      <el-option v-for="(one,idx) in materials"
                 :label="`${materialsForm.size * (materialsForm.page-1) + idx+1}. ${one.name}`"
                 :value="one.id" :key="idx"/>
    </el-select>
  </el-form-item>
</template>

<style scoped lang="scss">

</style>