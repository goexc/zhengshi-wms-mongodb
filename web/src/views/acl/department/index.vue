<script setup lang="ts">

import {nextTick, onMounted, ref} from "vue";
import {Sizes, Types} from "@/utils/enum.ts";
import {ElMessage, FormInstance} from "element-plus";
import {Department, DepartmentListResponse, DepartmentRemoveRequest} from "@/api/acl/department/types.ts";
import {reqAddOrUpdateDepartment, reqDepartmentList, reqRemoveDepartment} from "@/api/acl/department";
import {rules} from "@/views/acl/department/rules.ts";

//表格展开属性
const expand = ref<boolean>(true)
const tableVisible = ref<boolean>(true)

//dialog可见
const visible = ref<boolean>(false)
const title = ref<string>('')
const action = ref<string>('') //表单
const departments = ref<Department[]>([])
//部门列表
const hashTable = new Map<string, Department>()

//表单校验
const formRef = ref<FormInstance>()

const initDepartmentForm = () => {
  return <Department>({
    id: '',
    parent_id: '',
    sort_id: 0,
    path: '',
    name: '',
    component: '',
    icon: 'search',
    hidden: false,
    fixed: true,
    is_full: false,
    perms: '',
    transition: '',
    remark: '',
  })
}
const departmentForm = ref<Department>(initDepartmentForm())
//上级部门名称
const parentDepartment = ref<string>('')

//部门展开
const switchExpand = (val: boolean) => {
  tableVisible.value = false
  expand.value = val

  nextTick(() => {
    tableVisible.value = true
  })
}

//添加子部门
const addDepartment = (parent: Department) => {
  nextTick(() => {
    formRef.value?.clearValidate()
  })

  title.value = '新增部门'
  action.value = 'addDepartment'
  visible.value = true
  departmentForm.value = initDepartmentForm()
  departmentForm.value.parent_id = parent.id as string

  if (parent.id.length > 0) {
    parentDepartment.value = hashTable.get(parent.id)?.name
  } else {
    parentDepartment.value = ''
  }
}

//修改部门
const editDepartment = (department: Department) => {
  nextTick(() => {
    formRef.value?.clearValidate()
  })

  title.value = '修改部门'
  action.value = 'editDepartment'
  Object.assign(departmentForm.value, department)//对象浅拷贝

  parentDepartment.value = hashTable.get(department.parent_id)?.name

  visible.value = true
}

//删除部门
const deleteDepartment = async (id: string) => {
  let req = <DepartmentRemoveRequest>({id: id})
  let res = await reqRemoveDepartment(req)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    await getDepartments()
  } else {
    ElMessage.error(res.msg)
  }
}

//关闭表单
const handleClose = () => {
  visible.value = false
  nextTick(() => {
    formRef.value?.clearValidate()
  })
  departmentForm.value = initDepartmentForm()
}

//提交表单
const handleSubmit = async () => {
  //表单校验
  let valid = await formRef.value?.validate((isValid) => {
    if (!isValid) {
    }
    return
  })

  if (!valid) {
    return
  }

  let res = await reqAddOrUpdateDepartment(departmentForm.value)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    handleClose()
    await getDepartments()
  } else {
    ElMessage.error(res.msg)
  }
}

const getDepartments = async () => {
  let res: DepartmentListResponse = await reqDepartmentList()
  if (res.code === 200) {
    // departments.value = res.data?.sort((a, b) => a.sort_id - b.sort_id)
    departments.value = res.data
    await getHashTable(departments.value)
  }
}

const getHashTable = async (tree: Department[]) => {
  tree.forEach(one => {
    hashTable.set(one.id, one)

    if (one.children && one.children.length > 0) {
      getHashTable(one.children)
    }
  })
}

onMounted(() => {
  getDepartments()
})

</script>

<template>
  <div>
    <perms-button
        perms="privilege:department:add"
        :type="Types.primary"
        :size="Sizes.default"
        :plain="true"
        @action="addDepartment(initDepartmentForm())"/>
    <el-button plain icon="Bottom" @click="switchExpand(true)" size="default">展开全部</el-button>
    <el-button plain icon="Top" @click="switchExpand(false)" size="default">折叠全部</el-button>
    <!--   展示部门列表   -->
    <el-table
        v-if="tableVisible"
        class="table"
        stripe
        border
        :show-overflow-tooltip="true"
        tooltip-effect="dark"
        :data="departments"
        row-key="id"
        :default-expand-all="expand"
        :tree-props="{ children: 'children', hasChildren: '!!children' }"
    >

      <template #empty>
        <el-empty/>
      </template>
      <el-table-column prop="name" label="名称" min-width="220px" fixed/>
      <el-table-column prop="sort_id" label="排序" width="80px" align="center"/>
      <el-table-column prop="code" label="编码" min-width="220px" fixed/>
      <el-table-column label="操作" width="300px" fixed="right">
        <template #default="{row}">
          <el-button type="primary" plain size="small" icon="Plus" @click="addDepartment(row)">添加子部门</el-button>
          <el-button type="warning" plain size="small" icon="Edit" @click="editDepartment(row)">编辑</el-button>
          <el-popconfirm
              :title="`确定删除部门[${row.name}]吗?`"
              icon="InfoFilled"
              icon-color="#F56C6C"
              cancel-button-text="取消"
              confirm-button-text="确认删除"
              cancel-button-type="info"
              confirm-button-type="danger"
              @confirm="deleteDepartment(row.id)"
              width="300"
          >
            <template #reference>
              <perms-button
                  perms="privilege:department:delete"
                  :type="Types.danger"
                  :size="Sizes.small"
                  :plain="true"/>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>

    </el-table>
    <el-dialog
        v-model="visible"
        :title="title"
        draggable
        width="800"
        :close-on-click-modal="false"
        @close="handleClose"
    >
      <el-form ref="formRef" :model="departmentForm" :rules="rules" label-width="100">
        <el-row>
          <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
            <el-form-item label="上级部门">
              {{ parentDepartment }}
            </el-form-item>
            <el-form-item label="部门编码" prop="code">
              <el-input v-model="departmentForm.code"/>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
            <el-form-item label="部门名称" prop="name">
              <el-input v-model="departmentForm.name"/>
            </el-form-item>
            <el-form-item label="部门排序" prop="sort_id">
              <el-input v-model.number="departmentForm.sort_id"/>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="备注">
          <el-input type="textarea" rows="5" placeholder="请输入备注"></el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="handleClose">取消</el-button>
        <el-button type="primary" plain @click="handleSubmit">提交</el-button>
      </template>

    </el-dialog>
  </div>
</template>

<style scoped lang="scss">

.table {
  margin: 10px 0;
}
</style>