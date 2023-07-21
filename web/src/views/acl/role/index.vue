<script setup lang="ts">
import {nextTick, onMounted, ref} from "vue";
import {reqChangeRoleApis, reqRoleApis, reqRoles} from "@/api/acl/role";
import {roleRules} from "./rules";
import {ElMessage, FormInstance} from "element-plus";
import {reqAddOrUpdateRole} from "@/api/acl/role";
import {TimeFormat} from "@/utils/time";
import {reqChangeRoleMenus} from "@/api/acl/role";
import {baseResponse} from "@/api/types";
import {reqChangeRoleStatus, reqRoleMenus} from "@/api/acl/role";
import {Menu, MenuListResponse} from "@/api/acl/menu/types.ts";
import {reqMenuList} from "@/api/acl/menu";
import {
  Role, RoleApisResponse,
  RoleMenusRequest,
  RoleMenusResponse,
  RoleRequest,
  RolesRequest,
  RolesResponse, RoleStatusRequest
} from "@/api/acl/role/types.ts";
import {Sizes, Types} from "@/utils/enum.ts";
import {Api, ApiListResponse} from "@/api/acl/api/types.ts";
import {reqApiList} from "@/api/acl/api";

const page = ref<number>(1)
const size = ref<number>(10)
const total = ref<number>(0)
const roles = ref<Role[]>([])

const initRole = () => {
  return {
    id: '',
    parent_id: '',
    name: '',
    status: '启用',
    remark: '',
  }
}

//添加、修改角色信息
const roleForm = ref<RoleRequest>(initRole())
const roleRef = ref<FormInstance>()

const getRoles = async () => {
  let req = <RolesRequest>({
    page: page.value,
    size: size.value,
    name: roleForm.value.name,
  })
  let res: RolesResponse = await reqRoles(req)
  if (res.code === 200) {
    total.value = res.data.total
    roles.value = res.data.list
  }
}

//查询菜单列表
const getMenus = async () => {
  let res: MenuListResponse = await reqMenuList()
  menus.value = []
  if (res.code === 200) {
    menus.value = res.data
  } else {
    ElMessage.error(res.msg)
  }
}

//查询角色的菜单列表
const getRoleMenus = async (id: string) => {
  let res: RoleMenusResponse = await reqRoleMenus({id: id})
  menusId.value = []
  if (res.code === 200) {
    //计算叶子节点
    let allNodes  = menuTreeRef.value.store._getAllNodes()
    let allLeaf = allNodes.filter((item:any)=>item.isLeaf).map((item:any)=>item.data.id)
    //只筛选叶子节点
    menusId.value = res.data.filter(menuId=>allLeaf.includes(menuId))
  } else {
    ElMessage.error(res.msg)
  }
}


//重置角色分页
const resetMenus = () => {
  page.value = 1
  size.value = 10
  roleForm.value.name = ''
  getRoles()
}

onMounted(() => {
  getRoles()
})

const handleSizeChange = () => {
  getRoles()
}
const handleCurrentChange = () => {
  getRoles()
}

//抽屉
const visible = ref<boolean>(false)
const title = ref<string>('')

//添加角色
const addRole = () => {
  title.value = '添加角色'
  visible.value = true
  nextTick(() => {
    roleRef.value?.clearValidate()
  })
}
//修改角色信息
const editRole = (row: Role) => {
  title.value = '修改角色'
  Object.assign(roleForm.value, row)
  visible.value = true
}
//关闭抽屉回调函数
const close = () => {
  visible.value = false
  roleForm.value = initRole()
}

//提交
const confirm = async () => {
  //1.表单校验
  let valid = await roleRef.value?.validate((valid, fields) => {
    if (valid) {

    } else {
      console.log('fields:', fields)
    }
    return
  })

  if (!valid) {
    return
  }

  let res = await reqAddOrUpdateRole(roleForm.value)
  if (res.code === 200) {
    ElMessage({
      type: "success",
      message: "成功"
    })
    //清理表单
    close()
    //刷新分页
    await   getRoles()
  } else {
    ElMessage({
      type: "error",
      message: res.msg
    })
  }
}

//角色状态样式
const statusType = (status: string) => {
  let t = ''
  switch (status) {
    case '启用':
      t = 'success'
      break
    case '禁用':
      t = 'warning'
      break
    case '删除':
      t = 'danger'
      break
    default:
      t = ''
  }
  return t
}

//分配菜单
const initRoles = () => {
  return {
    id: '',
    name: '',
    roles_id: []
  }
}
//不确定样式
const rolesForm = ref(initRoles())
const menus = ref<Menu[]>([])//菜单列表
const menusId = ref<string[]>([])
const menusVisible = ref<boolean>(false)
const menuTreeRef = ref<FormInstance>()

//给角色分配菜单
const setMenus = async (row: Role) => {
  menusVisible.value = true

  //查询菜单树
  await getMenus()
  //查询角色的菜单列表
  await getRoleMenus(row.id)

  rolesForm.value.id = row.id
  rolesForm.value.name = row.name
}
const menusClose = () => {
  //清空选中的节点。
  //setCheckedKeys	设置目前选中的节点，
  // 使用此方法必须设置 node-key 属性	(keys, leafOnly) 接收两个参数:
  // 1. 一个需要被选中的多节点 key 的数组
  // 2. 布尔类型的值 如果设置为 true，将只设置选中的叶子节点状态。 默认值是 false.
  menuTreeRef.value.setCheckedKeys([], false)
  menusVisible.value = false
}

//保存角色菜单
const menusConfirm = async () => {
  //(leafOnly) 接收一个布尔类型参数，默认为 false.
  // 如果参数是 true, 它只返回当前选择的子节点数组。
  //  menusId.value =  menuTreeRef.value.getCheckedKeys(true)
  let leafKeys = menuTreeRef.value.getCheckedKeys() as string[]
  let halfKeys = menuTreeRef.value.getHalfCheckedKeys() as string[]
  let msId = []
  msId.push(...leafKeys)
  msId.push(...halfKeys)
  // return
  // console.log('叶子节点：', leafKeys)
  // console.log('半选中节点：', halfKeys)
  // console.log('全部节点：', msId)
  if (msId.length === 0) {
    ElMessage({
      type: "warning",
      message: '请选择至少 1 个菜单'
    })
    return
  }
  let req = <RoleMenusRequest>({id: rolesForm.value.id, menus_id: msId})
  let res: baseResponse = await reqChangeRoleMenus(req)
  if (res.code === 200) {
    menusClose()
    ElMessage({
      type: "success",
      message: res.msg
    })
  } else {
    ElMessage({
      type: "error",
      message: res.msg
    })
  }
}

//删除角色
const removeRole = async (row: Role) => {
  let req = <RoleStatusRequest>({id: [row.id], status: '删除'})
  let res = await reqChangeRoleStatus(req)
  if (res.code === 200) {
    ElMessage({
      type: "success",
      message: res.msg
    })
    await getRoles()
  } else {
    ElMessage({
      type: "error",
      message: res.msg
    })
  }
}

//批量选择角色
const selectRoles = ref<string[]>([])
const handleSelectRole = (val: Role[]) => {
  console.log('选择角色：', val.map(one => one.id))
  selectRoles.value = val.map(one => one.id)
}

//批量删除角色
const batchRemoveRole = async () => {
  if (selectRoles.value.length === 0) {
    ElMessage({
      type: "warning",
      message: "请选择要删除的角色"
    })
    return
  }

  let req = <RoleStatusRequest>({id: selectRoles.value, status: '删除'})
  let res = await reqChangeRoleStatus(req)
  if (res.code === 200) {
    ElMessage({
      type: "success",
      message: res.msg
    })
    await getRoles()
  } else {
    ElMessage({
      type: "error",
      message: res.msg
    })
  }
}

const apis = ref<Api[]>([])//菜单列表
const apisId = ref<string[]>([])
const apisVisible = ref<boolean>(false)
const apiTreeRef = ref<FormInstance>()



//查询API列表
const getApis = async () => {
  let res: ApiListResponse = await reqApiList()
  apis.value = []
  if (res.code === 200) {
    apis.value = res.data
  } else {
    ElMessage.error(res.msg)
  }
}

//查询角色的API列表
const getRoleApis = async (id: string) => {
  let res: RoleApisResponse = await reqRoleApis({id: id})
  apisId.value = []
  if (res.code === 200) {
    //计算叶子节点
    let allNodes  = apiTreeRef.value.store._getAllNodes()
    let allLeaf = allNodes.filter((item:any)=>item.isLeaf).map((item:any)=>item.data.id)
    //只筛选叶子节点
    apisId.value = res.data.filter(apiId=>allLeaf.includes(apiId))
  } else {
    ElMessage.error(res.msg)
  }
}



//给角色分配api
const setApis = async (row: Role) => {
  console.log('给角色分配api:', row)
  apisVisible.value = true

  //查询API树
  await getApis()
  //查询角色的菜单列表
  await getRoleApis(row.id)

  rolesForm.value.id = row.id
  rolesForm.value.name = row.name
}

const apisClose = () => {
  //清空选中的节点。
  //setCheckedKeys	设置目前选中的节点，
  // 使用此方法必须设置 node-key 属性	(keys, leafOnly) 接收两个参数:
  // 1. 一个需要被选中的多节点 key 的数组
  // 2. 布尔类型的值 如果设置为 true，将只设置选中的叶子节点状态。 默认值是 false.
  apiTreeRef.value.setCheckedKeys([], false)
  apisVisible.value = false
}


//保存角色API
const apisConfirm = async () => {
  //(leafOnly) 接收一个布尔类型参数，默认为 false.
  // 如果参数是 true, 它只返回当前选择的子节点数组。
  //  apisId.value =  apiTreeRef.value.getCheckedKeys(true)
  let leafKeys = apiTreeRef.value.getCheckedKeys() as string[]
  let halfKeys = apiTreeRef.value.getHalfCheckedKeys() as string[]
  let asId = []
  asId.push(...leafKeys)
  asId.push(...halfKeys)
  console.log('叶子节点：', leafKeys)
  console.log('半选中节点：', halfKeys)
  console.log('全部节点：', asId)
  // return
  if (asId.length === 0) {
    ElMessage({
      type: "warning",
      message: '请选择至少 1 个API'
    })
    return
  }
  let req = <RoleMenusRequest>({id: rolesForm.value.id, apis_id: asId})
  let res: baseResponse = await reqChangeRoleApis(req)
  if (res.code === 200) {
    apisClose()
    ElMessage({
      type: "success",
      message: res.msg
    })
  } else {
    ElMessage({
      type: "error",
      message: res.msg
    })
  }
}



</script>

<template>

  <div>
    <el-card class="form-card">
      <el-form inline class="form">
        <el-form-item label="角色名称">
          <el-input v-model.trim="roleForm.name" clearable placeholder="请输入角色名称"></el-input>
        </el-form-item>
        <el-form-item>
          <perms-button
              @click="getRoles"
              perms="privilege:role:list"
              :type="Types.primary"
              :size="Sizes.default"
              :plain="true"/>
          <perms-button
              @click="resetMenus"
              perms="privilege:role:list"
              type=""
              :size="Sizes.default"
              :plain="true"
              icon="RefreshRight"
              text="重置"
          />
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="body-card">
      <el-button type="primary" plain @click="addRole">添加角色</el-button>
      <el-popconfirm
          :title="`确认批量删除这些角色吗？`"
          @confirm="batchRemoveRole"
          width="360px"
      >
        <template #reference>
          <el-button type="danger" plain>批量删除</el-button>
        </template>
      </el-popconfirm>
      <!--   展示角色分页   -->
      <el-table
          class="table"
          stripe
          border
          :show-overflow-tooltip="true"
          tooltip-effect="dark"
          @selection-change="handleSelectRole"
          :data="roles"
      >
        <el-table-column type="selection" align="center" fixed></el-table-column>
        <el-table-column type="index" label="#" align="center" width="70px" fixed></el-table-column>
        <el-table-column label="角色名称" prop="name" width="120px" fixed></el-table-column>
        <el-table-column label="状态" prop="status">
          <template #default="{row}">
            <el-tag size="small" :type="statusType(row.status)">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="备注" prop="remark" min-width="120px"></el-table-column>
        <el-table-column label="创建时间" prop="created_at" width="180px">
          <template #default="{row}">
            {{ TimeFormat(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="修改时间" prop="updated_at" width="180px">
          <template #default="{row}">
            {{ TimeFormat(row.updated_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="370px" fixed="right">
          <template #default="{row}">
            <div v-if="row.status !== '删除'">
              <el-button type="primary" plain size="small" icon="Menu" @click="setMenus(row)">分配菜单</el-button>
              <el-button type="primary" plain size="small" icon="Menu" @click="setApis(row)">分配API</el-button>
              <el-button type="primary" plain size="small" icon="Edit" @click="editRole(row)">编辑</el-button>
              <el-popconfirm
                  v-if="row.status==='启用'"
                  :title="`确认删除角色 [${row.name}] 吗？`"
                  @confirm="removeRole(row)"
                  width="360px"
              >
                <template #reference>
                  <perms-button
                      perms="privilege:role:delete"
                      :type="Types.danger"
                      :size="Sizes.small"
                      :plain="true"/>
                </template>
              </el-popconfirm>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
          v-model:current-page="page"
          v-model:page-size="size"
          :page-sizes="[10, 20, 30, 50]"
          :background="true"
          layout="total, sizes, prev, pager, next, ->,jumper"
          :total="total"
          :pager-count="9"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
      />

    </el-card>
    <!--  添加、修改角色信息  -->
    <el-drawer
        direction="rtl"
        v-model="visible"
        :title="title"
        :before-close="close"
    >
      <template #default>
        <el-form
            label-width="80px"
            :model="roleForm"
            ref="roleRef"
            :rules="roleRules"
        >
          <el-form-item
              label="角色名称"
              prop="name"
          >
            <el-input
                v-model="roleForm.name"
                minlength="2"
                maxlength="21"
                :show-word-limit="true"
                placeholder="请填写角色名称，例如：运营、财务、运维"/>
          </el-form-item>
          <el-form-item label="状态" prop="status">
            <el-select v-model="roleForm.status" clearable placeholder="请选择状态">
              <el-option label="启用" value="启用"></el-option>
              <el-option label="禁用" value="禁用"></el-option>
            </el-select>
          </el-form-item>
          <el-form-item v-model="roleForm.remark" label="备注" prop="remark">
            <el-input v-model="roleForm.remark" type="textarea" rows="3" maxlength="125" :show-word-limit="true"
                      placeholder="请填写备注"/>
          </el-form-item>
        </el-form>

      </template>
      <template #footer>
        <div>
          <el-button plain @click="close">取消</el-button>
          <el-button type="primary" plain @click="confirm">保存</el-button>
        </div>
      </template>
    </el-drawer>
    <!--  分配菜单  -->
    <el-drawer
        direction="rtl"
        v-model="menusVisible"
        title="分配菜单"
        :before-close="menusClose"
    >
      <template #default>
        <el-form label-width="80px">
          <el-form-item label="角色名称">
            {{ rolesForm.name }}
          </el-form-item>
          <el-form-item label="菜单列表">
            <el-tree
                ref="menuTreeRef"
                :data="menus"
                show-checkbox
                node-key="id"
                default-expand-all
                :default-checked-keys="menusId"
                :props="{children:'children', label:'name'}"
                :check-strictly="false"
            >

            </el-tree>
          </el-form-item>
        </el-form>
      </template>
      <template #footer>
        <div>
          <el-button plain @click="menusClose">取消</el-button>
          <el-button type="primary" plain @click="menusConfirm">保存</el-button>
        </div>
      </template>
    </el-drawer>

    <!--  分配API  -->
    <el-drawer
        direction="rtl"
        v-model="apisVisible"
        title="分配API"
        :before-close="apisClose"
    >
      <template #default>
        <el-form label-width="80px">
          <el-form-item label="角色名称">
            {{ rolesForm.name }}
          </el-form-item>
          <el-form-item label="API列表">
            <el-tree
                ref="apiTreeRef"
                :data="apis"
                show-checkbox
                node-key="id"
                default-expand-all
                :default-checked-keys="apisId"
                :props="{children:'children', label:'name'}"
                :check-strictly="false"
            >

            </el-tree>
          </el-form-item>
        </el-form>
      </template>
      <template #footer>
        <div>
          <el-button plain @click="apisClose">取消</el-button>
          <el-button type="primary" plain @click="apisConfirm">保存</el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<style scoped lang="scss">
.form-card {
  height: 80px;
}

.form {
  display: flex;
  justify-content: flex-start;
  align-items: center;
}

.body-card {
  margin: 10px 0;
}

.table {
  margin: 10px 0;
}
</style>