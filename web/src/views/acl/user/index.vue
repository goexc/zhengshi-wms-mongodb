<script setup lang="ts">
import {nextTick, onMounted, ref} from "vue";
import {reqUsers} from "@/api/acl/user";
import {userRules} from "./rules";
import {CascaderInstance, CascaderValue, ElMessage, FormInstance} from "element-plus";
import {reqAddOrUpdateUser} from "@/api/acl/user";
import {TimeFormat} from "@/utils/time";
import {reqRoleList} from "@/api/acl/role";
import {reqChangeUserRoles} from "@/api/acl/user";
import {baseResponse} from "@/api/types";
import {reqChangeUserStatus} from "@/api/acl/user";
import {
  User,
  UserRequest,
  UserRolesRequest,
  UsersRequest,
  UsersResponse,
  UserStatusRequest
} from "@/api/acl/user/types.ts";
import {Role, RoleListRequest} from "@/api/acl/role/types.ts";
import {Department, DepartmentListResponse} from "@/api/acl/department/types.ts";
import {reqDepartmentList} from "@/api/acl/department";

const page = ref<number>(1)
const size = ref<number>(10)
const total = ref<number>(0)
const users = ref<User[]>([])

const initUser = () => {
  return {
    id: '',
    name: '',
    password: '',
    sex: '',
    department_id: '',
    roles_id: [],
    mobile: '',
    email: '',
    status: '',
    remark: '',
    created_at: 0,
    updated_at: 0,
  }
}

//查询表单
const searchForm = ref<UserRequest>(initUser())

//添加、修改用户信息
const userForm = ref<UserRequest>(initUser())
const userRef = ref<FormInstance>()

const getUsers = async () => {
  let req = <UsersRequest>({
    page: page.value,
    size: size.value,
    name: searchForm.value.name,
    mobile: searchForm.value.mobile
  })
  let res: UsersResponse = await reqUsers(req)
  if (res.code === 200) {
    total.value = res.data.total
    users.value = res.data.list
  }
}

//重置用户分页
const resetUsers = () => {
  page.value = 1
  size.value = 10
  searchForm.value.name = ''
  searchForm.value.mobile = ''
  getUsers()
}

onMounted(() => {
  getUsers()
  getDepartments()
})

const handleSizeChange = () => {
  getUsers()
}
const handleCurrentChange = () => {
  getUsers()
}

//抽屉
const visible = ref<boolean>(false)
const title = ref<string>('')

//添加用户
const addUser = () => {
  title.value = '添加用户'
  visible.value = true
  nextTick(() => {
    userRef.value?.clearValidate()
  })
}
//修改用户信息
const editUser = (row: User) => {
  title.value = '修改用户'
  Object.assign(userForm.value, row)

  //角色列表属性
  isIndeterminate.value = userForm.value.roles_id.length > 0 && userForm.value.roles_id.length < roles.value.length
  checkAllRoles.value = userForm.value.roles_id.length > 0 && userForm.value.roles_id.length === roles.value.length


  visible.value = true
}

//关闭抽屉回调函数
const close = () => {
  visible.value = false
  userForm.value = initUser()
}

//提交
const confirm = async () => {
  //1.表单校验
  let valid = await userRef.value?.validate((valid, fields) => {
    if (valid) {

    } else {
      console.log('fields:', fields)
    }
    return
  })

  if (!valid) {
    return
  }

  let res = await reqAddOrUpdateUser(userForm.value)
  if (res.code === 200) {
    ElMessage({
      type: "success",
      message: "成功"
    })
    //清理表单
    close()
    //刷新分页
    await getUsers()
  } else {
    ElMessage({
      type: "error",
      message: res.msg
    })
  }
}

//部门树
const departments = ref<Department[]>([])
const props = ref({label: 'name', value: 'id', checkStrictly: true})
const departmentRef = ref<CascaderInstance>()

const changeDepartment = (value: CascaderValue) => {
  if (value && (value as string[]).length > 0) {
    userForm.value.department_id = (value as string[]).pop() || ''
  } else {
    userForm.value.department_id = ''
  }
}

const getDepartments = async () => {
  let res: DepartmentListResponse = await reqDepartmentList()
  if (res.code === 200) {
    // departments.value = res.data?.sort((a, b) => a.sort_id - b.sort_id)
    departments.value = res.data
  }
}


//角色列表
const roles = ref<Role[]>([])

//查询角色列表
const getRoles = async () => {
  let req = <RoleListRequest>({name: ''})
  let res = await reqRoleList(req)
  if (res.code === 200) {
    roles.value = res.data.list
  }
}

//半选样式
const isIndeterminate = ref<boolean>(false)
const rolesVisible = ref<boolean>(false)
const checkAllRoles = ref<boolean>(false)

const setRoles = (row: User) => {
  Object.assign(userForm.value, row)
  //判断isIndeterminate状态
  isIndeterminate.value = userForm.value.roles_id.length > 0 && userForm.value.roles_id.length < roles.value.length
  checkAllRoles.value = userForm.value.roles_id.length > 0 && userForm.value.roles_id.length === roles.value.length

  rolesVisible.value = true
}
const rolesClose = () => {
  rolesVisible.value = false
}

//角色全选
const handleCheckAllChange = (val: boolean) => {
  userForm.value.roles_id = val ? roles.value.map((one) => one.id) : []
  isIndeterminate.value = false
}

//选择角色
const handleCheckedRolesChange = (val: string[]) => {
  const checkedCount = val.length
  checkAllRoles.value = checkedCount === roles.value.length
  isIndeterminate.value = checkedCount > 0 && checkedCount < roles.value.length
}

//保存用户角色
const rolesConfirm = async () => {
  if (!userForm.value.roles_id || userForm.value.roles_id.length === 0) {
    ElMessage({
      type: "warning",
      message: '请选择至少 1 个角色'
    })
    return
  }
  let req = <UserRolesRequest>({id: userForm.value.id, roles_id: userForm.value.roles_id})
  let res: baseResponse = await reqChangeUserRoles(req)
  if (res.code === 200) {
    await getUsers()
    rolesClose()
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

//删除用户
const removeUser = async (row: User) => {
  let req = <UserStatusRequest>({id: [row.id], status: '删除'})
  let res = await reqChangeUserStatus(req)
  if (res.code === 200) {
    ElMessage({
      type: "success",
      message: res.msg
    })
    await getUsers()
  } else {
    ElMessage({
      type: "error",
      message: res.msg
    })
  }
}

//批量选择用户
const selectUsers = ref<string[]>([])
const handleSelectUser = (val: User[]) => {
  selectUsers.value = val.map(one => one.id)
}

//批量删除用户
const batchRemoveUser = async () => {
  if (selectUsers.value.length === 0) {
    ElMessage({
      type: "warning",
      message: "请选择要删除的用户"
    })
    return
  }

  let req = <UserStatusRequest>({id: selectUsers.value, status: '删除'})
  let res = await reqChangeUserStatus(req)
  if (res.code === 200) {
    ElMessage({
      type: "success",
      message: res.msg
    })
    await getUsers()
  } else {
    ElMessage({
      type: "error",
      message: res.msg
    })
  }
}

//用户状态样式
const statusType = (status: string) => {
  let t: string
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
</script>

<template>

  <div>
    <el-card class="form-card">
      <el-form inline class="form">
        <el-form-item label="用户名称">
          <el-input v-model.trim="searchForm.name" clearable placeholder="请输入用户名称"></el-input>
        </el-form-item>
        <el-form-item label="手机号码">
          <el-input v-model.trim="searchForm.mobile" clearable placeholder="请输入手机号码"></el-input>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" plain @click="getUsers">查询</el-button>
          <el-button plain @click="resetUsers">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="body-card">
      <el-button type="primary" plain @click="addUser">添加用户</el-button>
      <el-popconfirm
          :title="`确认批量删除这些用户吗？`"
          @confirm="batchRemoveUser"
          width="360px"
      >
        <template #reference>
          <el-button type="danger" plain>批量删除</el-button>
        </template>
      </el-popconfirm>
      <!--   展示用户分页   -->
      <el-table
          class="table"
          stripe
          border
          :show-overflow-tooltip="true"
          tooltip-effect="dark"
          @selection-change="handleSelectUser"
          :data="users"
      >
        <el-table-column type="selection" align="center" fixed></el-table-column>
        <el-table-column type="index" label="#" align="center" width="70px" fixed></el-table-column>
        <el-table-column label="用户名称" prop="name" width="120px" fixed></el-table-column>
        <el-table-column label="手机号码" prop="mobile" width="160px"></el-table-column>
        <el-table-column label="Email" prop="email" width="220px"></el-table-column>
        <el-table-column label="状态" prop="status">
          <template #default="{row}">
            <el-tag size="small" :type="statusType(row.status)">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="部门" prop="department_name" width="120px"></el-table-column>
        <el-table-column label="用户角色" prop="roles" min-width="150px">
          <template #default="{row}">
            {{ (row as User).roles_name?.join(', ') }}
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
        <el-table-column label="操作" width="280px" fixed="right">
          <template #default="{row}">
            <el-button v-if="row.status !=='删除'" type="primary" plain size="small" icon="User" @click="setRoles(row)">
              分配角色
            </el-button>
            <el-button v-if="row.status !=='删除'" type="primary" plain size="small" icon="Edit" @click="editUser(row)">
              编辑
            </el-button>
            <el-popconfirm
                v-if="row.status !=='删除'"
                :title="`确认删除用户 [${row.name}] 吗？`"
                @confirm="removeUser(row)"
                width="360px"
            >
              <template #reference>
                <el-button type="danger" plain size="small" icon="Delete">删除</el-button>
              </template>
            </el-popconfirm>
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
    <!--  添加、修改用户信息  -->
    <el-drawer
        direction="rtl"
        v-model.trim="visible"
        :title="title"
        @open="getRoles"
        :before-close="close"
    >
      <template #default>
        <el-form
            label-width="80px"
            :model="userForm"
            ref="userRef"
            :rules="userRules"
        >
          <el-form-item
              label="账号名称"
              prop="name"
          >
            <el-input
                v-model.trim="userForm.name"
                minlength="2"
                maxlength="21"
                :show-word-limit="true"
                placeholder="请填写账号名称，例如：马化腾"/>
          </el-form-item>
          <el-form-item v-if="userForm.id.trim().length===0" label="账号密码" prop="password">
            <el-input
                v-model.trim="userForm.password"
                :show-password="true"
                minlength="6"
                :show-word-limit="true"
                placeholder="请填写账号密码"/>
          </el-form-item>
          <el-form-item label="手机号码" prop="mobile">
            <el-input v-model.trim="userForm.mobile" placeholder="请填写手机号码，例如：+8618810509066"/>
          </el-form-item>
          <el-form-item label="Email" prop="email">
            <el-input v-model.trim="userForm.email" placeholder="请填写Email，例如：mahuateng@qq.com"/>
          </el-form-item>
          <el-form-item label="角色列表" prop="roles_id">
            <el-checkbox
                v-model.trim="checkAllRoles"
                :indeterminate="isIndeterminate"
                @change="handleCheckAllChange"
            >
              全选
            </el-checkbox>
          </el-form-item>
          <el-form-item label="">
            <el-checkbox-group
                v-model.trim="userForm.roles_id"
                @change="handleCheckedRolesChange"
            >
              <el-checkbox
                  v-for="(role, index) in roles"
                  :key="index"
                  :label="role.id"
                  :disabled="role.status !== '启用'"
                  class="role"
              >
                {{ role.name }}
              </el-checkbox>
            </el-checkbox-group>
          </el-form-item>
          <el-form-item label="性别" prop="sex">
            <el-select v-model.trim="userForm.sex" clearable placeholder="请选择性别">
              <el-option label="男" value="男"></el-option>
              <el-option label="女" value="女"></el-option>
            </el-select>
          </el-form-item>
          <el-form-item label="部门" prop="department_id">
            <el-cascader
                :options="departments"
                :props="props"
                ref="departmentRef"
                clearable
                @change="changeDepartment"
            >

            </el-cascader>
          </el-form-item>
          <el-form-item label="状态" prop="status">
            <el-select v-model.trim="userForm.status" clearable placeholder="请选择状态">
              <el-option label="启用" value="启用"></el-option>
              <el-option label="禁用" value="禁用"></el-option>
            </el-select>
          </el-form-item>
          <el-form-item v-model.trim="userForm.remark" label="备注" prop="remark">
            <el-input v-model.trim="userForm.remark" type="textarea" rows="3" maxlength="125" :show-word-limit="true"
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
    <!--  分配角色  -->

    <el-drawer
        direction="rtl"
        v-model.trim="rolesVisible"
        title="分配角色"
        @open="getRoles"
        :before-close="rolesClose"
    >
      <template #default>
        <el-form label-width="80px">
          <el-form-item label="用户账号">
            <el-input :model-value="userForm.name" disabled/>
          </el-form-item>
          <el-form-item label="角色列表">
            <el-checkbox
                v-model.trim="checkAllRoles"
                :indeterminate="isIndeterminate"
                @change="handleCheckAllChange"
            >
              全选
            </el-checkbox>
          </el-form-item>
          <el-form-item>
            <el-checkbox-group
                v-model.trim="userForm.roles_id"
                @change="handleCheckedRolesChange"
            >
              <el-checkbox
                  v-for="(role, index) in roles"
                  :key="index" :label="role.id"
                  :disabled="role.status !== '启用'"
                  class="role"
              >
                {{ role.name }}
              </el-checkbox>
            </el-checkbox-group>
          </el-form-item>
        </el-form>
      </template>
      <template #footer>
        <div>
          <el-button plain @click="rolesClose">取消</el-button>
          <el-button type="primary" plain @click="rolesConfirm">保存</el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<style scoped>
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

.role {
  float: left;
  width: 110px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>