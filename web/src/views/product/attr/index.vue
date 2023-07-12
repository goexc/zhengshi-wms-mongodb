<script setup lang="ts">

//编辑
import { reactive, watch} from "vue";
import {ElMessage} from "element-plus";
import {ref} from "vue";
import useCategoryStore from "@/store/module/category";
import {reqAddOrUpdateWarehouseBin, reqChangeWarehouseBinStatus, reqWarehouseBins} from "@/api/warehouse_bin";
import {warehouseBinRules} from "./rules";
import {
  WarehouseBin, WarehouseBinListRequest, WarehouseBinRequest,
  WarehouseBinsRequest,
  WarehouseBinsResponse,
  WarehouseBinStatusRequest
} from "@/api/warehouse_bin/types.ts";

let categoryStore =useCategoryStore()

const initBin = ()=>{
  return <WarehouseBin>{
    id: '',
    warehouse_id: '',
    warehouse_name: '',
    warehouse_zone_id: '',
    warehouse_zone_name: '',
    warehouse_rack_id: '',
    warehouse_rack_name: '',
    name: '',
    code: '',
    capacity: 0,
    capacity_unit: '',
    status: '',
    remark: '',
    create_by: '',
    created_at: 0,
    updated_at: 0,
  }
}

const title = ref<string>('')
const visible = ref<boolean>(false)
const bin = reactive<WarehouseBin>(initBin())
const page = ref<number>(1)
const size = ref<number>(10)

//货位分页
const getWarehouseBins = async ()=>{
  //清空下级列表
  categoryStore.warehouse_bins = []
  //获取货位列表
  let req = <WarehouseBinsRequest>({
    page: page.value,
    size: size.value,
    warehouse_rack_id: categoryStore.warehouse_rack_id
  })
  let res:WarehouseBinsResponse = await reqWarehouseBins(req)
  if(res.code===200){
    Object.assign(categoryStore.warehouse_bins, res.data.list)
  }
}

//监听货架切换
watch(()=>categoryStore.warehouse_rack_id, ()=>{
  if(categoryStore.warehouse_rack_id.trim().length>0){
    getWarehouseBins()
  }
})

//添加货位信息
const add = ()=>{
  Object.assign(bin, initBin())

  //收集货架信息
  bin.warehouse_rack_id = categoryStore.warehouse_rack_id
  bin.warehouse_name = categoryStore.getWarehouseName
  bin.warehouse_zone_name = categoryStore.getWarehouseZoneName
  bin.warehouse_rack_name = categoryStore.getWarehouseRackName
  console.log('categoryStore.getWarehouseName:', categoryStore.getWarehouseName)
  scene.value = false

}

//编辑货位信息
const edit = (row: WarehouseBin) => {
  //预先清空表单校验提示信息
  // nextTick(() => {
  //   formRef.value.clearValidate()
  // })

  title.value = '修改货位'
  visible.value = true
  Object.assign(bin, row)
  console.log('货位初始值：', bin)
  console.log('id:', bin.id)
  //切换场景
  scene.value = false
}
//删除
const remove = async (id: string) => {
  let req = <WarehouseBinStatusRequest>{id: id, status: '删除'}
  let res = await reqChangeWarehouseBinStatus(req)
  console.log(res)
  if(res.code === 200){
    await getWarehouseBins()
    ElMessage({
      type: 'success',
      message: '成功',
    })
  }else{
    ElMessage({
      type: 'error',
      message: res.msg,
    })
  }
}

//场景切换：列表/添加
const scene = ref<boolean>(true)


//取消 添加货位操作
const cancel = ()=>{
  scene.value = true
}

//提交添加/修改的货位信息
const save =async ()=>{
  let req = <WarehouseBinRequest>({
    id: bin.id,
    warehouse_rack_id: bin.warehouse_rack_id,
    name: bin.name,
    code: bin.code,
    capacity: bin.capacity,
    capacity_unit: bin.capacity_unit,
    remark: bin.remark,
  })
  let res = await reqAddOrUpdateWarehouseBin(req)
  if(res.code === 200){
    scene.value = true

    //获取货位列表
    let req = <WarehouseBinListRequest>({
      warehouse_rack_id: categoryStore.warehouse_rack_id
    })
    await categoryStore.getWarehouseBinList(req)
  }
}
</script>

<template>
  <div>
<!--  三级分类全局组件  -->
    <Category :scene="scene"></Category>
    <el-card class="body">
      <div
          v-show="scene"
      >
        <el-button
            type="primary"
            plain icon="Plus"
            :disabled="categoryStore.warehouse_rack_id.trim().length===0"
            @click="add"
        >添加货位</el-button>
        <el-table
            stripe
            border
            class="table"
            :data="categoryStore.warehouse_bins"
        >
          <el-table-column type="index" align="center" label="序号" width="80"></el-table-column>
          <el-table-column prop="warehouse_name" label="仓库"></el-table-column>
          <el-table-column prop="warehouse_zone_name" label="库区"></el-table-column>
          <el-table-column prop="warehouse_rack_name" label="货架"></el-table-column>
          <el-table-column prop="name" label="货位"></el-table-column>
          <el-table-column prop="code" label="货位编号"></el-table-column>
          <el-table-column prop="capacity" label="货位容量"></el-table-column>
          <el-table-column prop="capacity_unit" label="容量单位"></el-table-column>
          <el-table-column prop="status" label="货位状态"></el-table-column>
          <el-table-column prop="create_by" label="创建人"></el-table-column>
          <el-table-column prop="remark" label="备注"></el-table-column>
          <el-table-column prop="created_at" label="创建时间"></el-table-column>
          <el-table-column prop="updated_at" label="修改时间"></el-table-column>
          <el-table-column prop="op" label="操作" align="center" width="180">
            <template #default="{row}">
              <el-button type="primary" size="small" @click="edit(row)" icon="Edit" plain>编辑</el-button>
              <el-popconfirm
                  :title="`确认删除货位 [${row.name}] 吗？`"
                  @confirm="remove(row.id)"
                  width="360px"
              >
                <template #reference>
                  <el-button type="danger" size="small" icon="Delete" plain>删除</el-button>
                </template>
              </el-popconfirm>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <div
      v-show="!scene"
      >
<!--展示添加货位的相关信息-->
        <el-form
            class="form"
            label-width="100px"
            :model="bin"
            :rules="warehouseBinRules"
        >
          <el-form-item prop="warehouse_name" label="仓库">
            <el-input v-model="bin.warehouse_name" disabled placeholder=""/>
          </el-form-item>
          <el-form-item prop="warehouse_zone_name" label="库区">
            <el-input v-model="bin.warehouse_zone_name" disabled placeholder=""/>
          </el-form-item>
          <el-form-item prop="warehouse_rack_name" label="货架">
            <el-input v-model="bin.warehouse_rack_name" disabled placeholder=""/>
          </el-form-item>
          <el-form-item prop="name" label="货位名称">
            <el-input v-model="bin.name" placeholder="请输入货位名称"/>
          </el-form-item>
          <el-form-item prop="code" label="货位编号">
            <el-input v-model="bin.code" placeholder=""/>
          </el-form-item>
          <el-form-item prop="capacity" label="货位容量">
            <el-input-number v-model.number="bin.capacity" :precision="3" :controls="false" placeholder=""/>
          </el-form-item>
          <el-form-item prop="capacity_unit" label="容量单位">
            <el-input v-model="bin.capacity_unit" placeholder=""/>
          </el-form-item>
          <el-form-item prop="status" label="货位状态">
            <el-input v-model="bin.status" placeholder=""/>
          </el-form-item>
          <el-form-item prop="remark" label="备注">
            <el-input v-model="bin.remark" placeholder=""/>
          </el-form-item>
        </el-form>
        <el-table
            stripe
            border
            class="table">
          <el-table-column type="index" label="序号" align="center" width="80"></el-table-column>
          <el-table-column label="货位名称"></el-table-column>
          <el-table-column label="操作"></el-table-column>
        </el-table>
        <el-button type="primary" plain icon="Check" @click="save">保存</el-button>
        <el-button type="warning" plain icon="Close" @click="cancel">取消</el-button>
      </div>
    </el-card>
  </div>

</template>

<style scoped>
.body {
  margin: 10px 0;
}

.table {
  margin: 10px 0;
}

.form{
  width: 500px;
}
</style>