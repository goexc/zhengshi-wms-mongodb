<script setup lang="ts">

import {reqMenuList} from "@/api/acl/menu";
import {Menu, MenuListResponse, MenuRemoveRequest} from "@/api/acl/menu/types";
import {nextTick, onMounted, reactive, ref} from "vue";
import {TimeFormat} from "@/utils/time.ts";
import {Sizes, Types} from "@/utils/enum.ts";
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import {reqAddOrUpdateMenu, reqRemoveMenu} from "@/api/acl/menu";
import {ElMessage, FormInstance} from "element-plus";
import {MenuStatusRequest} from "@/api/acl/menu/types.ts";
import {rules} from "./rules";


//表格展开属性
const expand = ref<boolean>(true)
const tableVisible = ref<boolean>(true)
//菜单展开
const switchExpand = (val: boolean) => {
  tableVisible.value = false
  expand.value = val

  nextTick(() => {
    tableVisible.value = true
  })
}

const menus = ref<Menu[]>([])
const getMenus = async () => {
  let res: MenuListResponse = await reqMenuList()
  if (res.code === 200) {
    menus.value = res.data
  }
}

onMounted(() => {
  getMenus()
  loadIcons()
})

const initMenuForm = () => {
  return <Menu>({
    id: '',
    parent_id: '',
    type: 1,
    sort_id: 0,
    path: '',
    name: '',
    component: '',
    meta: {
      icon: 'search',
      hidden: false,
      fixed: true,
      is_full: false,
      perms: '',
      transition: '',
    },
    remark: '',
  })
}

//dialog可见
const visible = ref<boolean>(false)
const title = ref<string>('')
const action = ref<string>('') //表单动作：addMenu，新增菜单； editMenu，修改菜单
const menuForm = reactive<Menu>(initMenuForm())

//图标列表
const iconList = ref<Array<string>>([])
const icon = ref<string>('search')
//表单校验
const formRef = ref<FormInstance>()

//图标列表
const loadIcons = () => {
  iconList.value = []
  for (const [key] of Object.entries(ElementPlusIconsVue)) {
    iconList.value.push(key)
  }
}

//选择图标
const selectIcon = (name: string) => {
  icon.value = name
  menuForm.meta.icon = name
}


//添加子菜单
const addMenu = (parent: Menu) => {
  title.value = '新增菜单'
  action.value = 'addMenu'
  visible.value = true
  Object.assign(menuForm, initMenuForm())
  menuForm.parent_id = parent.id
}

//修改菜单
const editMenu = (menu: Menu) => {
  title.value = '修改菜单'
  action.value = 'editMenu'
  visible.value = true
  Object.assign(menuForm, menu)
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


  let res = await reqAddOrUpdateMenu(menuForm)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    handleClose()
    await getMenus()
  } else {
    ElMessage.error(res.msg)
  }
}


//删除菜单
const deleteMenu = async (id: string) => {
  let req = <MenuRemoveRequest>({id: id})
  let res = await reqRemoveMenu(req)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    await getMenus()
  } else {
    ElMessage.error(res.msg)
  }
}

//关闭表单
const handleClose = () => {
  visible.value = false
  formRef.value?.clearValidate()
  Object.assign(menuForm, initMenuForm())
}

</script>

<script lang="ts">
//菜单类型
const menuText = (t: number) => {
  let text = ''
  switch (t) {
    case 1:
      text = '菜单'
      break
    case 2:
      text = '按钮'
      break
    default:
      text = '未知'
  }
  return text
}

//菜单样式
const menuType = (t: number) => {
  let text = ''
  switch (t) {
    case 1:
      text = ''
      break
    case 2:
      text = 'warning'
      break
    default:
      text = 'danger'
  }
  return text
}


//隐藏类型
const hiddenText = (t: boolean) => {
  let text = ''
  switch (t) {
    case true:
      text = '是'
      break
    case false:
      text = '否'
      break
  }
  return text
}

//隐藏样式
const hiddenType = (t: boolean) => {
  let text = ''
  switch (t) {
    case true:
      text = 'warning'
      break
    case false:
      text = ''
      break
  }
  return text
}

//固定类型
const fixedText = (t: boolean) => {
  let text = ''
  switch (t) {
    case true:
      text = '是'
      break
    case false:
      text = '否'
      break
  }
  return text
}

//固定样式
const fixedType = (t: boolean) => {
  let text = ''
  switch (t) {
    case true:
      text = ''
      break
    case false:
      text = 'warning'
      break
  }
  return text
}


//全屏类型
const screenText = (t: boolean) => {
  let text = ''
  switch (t) {
    case true:
      text = '是'
      break
    case false:
      text = '否'
      break
  }
  return text
}

//全屏样式
const screenType = (t: boolean) => {
  let text = ''
  switch (t) {
    case true:
      text = ''
      break
    case false:
      text = 'warning'
      break
  }
  return text
}

</script>

<template>
  <div class="container">
    <div id="auth" v-auth="'privilege:menu:list'">
      <el-button plain @click="switchExpand(true)" size="default" icon="ArrowDown">展开全部</el-button>
      <el-button plain @click="switchExpand(false)" size="default" icon="ArrowUp">折叠全部</el-button>
      <perms-button
          perms="privilege:menu:add"
          :type="Types.primary"
          :size="Sizes.default"
          :plain="true"
          @action="addMenu(initMenuForm())"
      />
      <!--   展示菜单列表   -->
      <el-table
          v-if="tableVisible"
          class="table"
          stripe
          border
          :show-overflow-tooltip="true"
          tooltip-effect="dark"
          :data="menus"
          row-key="id"
          :default-expand-all="expand"
          :tree-props="{ children: 'children', hasChildren: '!!children' }"
      >

        <template #empty>
          <el-empty/>
        </template>
        <el-table-column prop="name" label="菜单名称" min-width="220px" fixed>
          <template #default="{row}">
<!--            <el-link :icon="row.icon">{{ row.name }}</el-link>-->
            <el-link>{{ row.name }}</el-link>
          </template>
        </el-table-column>
        <el-table-column label="图标" align="center">
          <template #default="{row}">
            <el-icon>
              <component :is="row.meta.icon"></component>
            </el-icon>
          </template>
        </el-table-column>
        <el-table-column label="排序" prop="sort_id" width="80px" align="center"></el-table-column>
        <el-table-column label="路由" prop="path" width="220px"></el-table-column>
        <el-table-column label="组件路径" prop="component" width="220px"></el-table-column>
        <el-table-column label="权限标识" prop="meta.perms" width="220px"></el-table-column>
        <el-table-column label="类型" prop="type" align="center" width="80px">
          <template #default="{row}">
            <el-tag size="small" :type="menuType(row.type)">{{ menuText(row.type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="特效" prop="meta.transition" min-width="120px"></el-table-column>
        <el-table-column label="隐藏" prop="hidden" width="80px" align="center">
          <template #default="{row}">
            <el-tag size="small" :type="hiddenType(row.meta.hidden)">{{ hiddenText(row.meta.hidden) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="固定" prop="fixed" width="80px" align="center">
          <template #default="{row}">
            <el-tag size="small" :type="fixedType(row.meta.fixed)">{{ fixedText(row.meta.fixed) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="全屏" prop="is_full" width="80px" align="center">
          <template #default="{row}">
            <el-tag size="small" :type="screenType(row.meta.is_full)">{{ screenText(row.meta.is_full) }}</el-tag>
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
        <el-table-column label="操作" min-width="300px" fixed="right">
          <template #default="{row}">
            <perms-button
                v-if="row.type === 1"
                perms="privilege:menu:add"
                :type="Types.primary"
                :size="Sizes.small"
                :plain="true"
                @action="addMenu(row)"
            />
            <perms-button
                perms="privilege:menu:edit"
                :type="Types.warning"
                :size="Sizes.small"
                :plain="true"
                @action="editMenu(row)"
            />
            <el-popconfirm
                :title="`确定删除菜单[${row.name}]吗?`"
                icon="InfoFilled"
                icon-color="#F56C6C"
                cancel-button-text="取消"
                confirm-button-text="确认删除"
                cancel-button-type="info"
                confirm-button-type="danger"
                @confirm="deleteMenu(row.id)"
                width="300"
            >
              <template #reference>
                <perms-button
                    perms="privilege:menu:delete"
                    :type="Types.danger"
                    :size="Sizes.small"
                    :plain="true"/>
                <!--              <el-button type="danger" plain size="small">删除</el-button>-->
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
        <el-form ref="formRef" :model="menuForm" :rules="rules" label-width="100">
          <el-row>
            <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
              <el-form-item label="菜单类型" prop="type">
                <el-radio-group v-model.number="menuForm.type" :disabled="action==='editMenu'">
                  <el-radio :label="1">菜单</el-radio>
                  <el-radio :label="2">按钮</el-radio>
                </el-radio-group>
              </el-form-item>
              <el-form-item label="菜单名称" prop="name">
                <el-input v-model="menuForm.name" style="width: 360px" placeholder="例如：权限管理"/>
              </el-form-item>
              <el-form-item label="菜单排序" prop="sort_id">
                <el-input v-model.number="menuForm.sort_id" style="width: 360px"/>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
              <!--            <el-form-item label="菜单排序" prop="sort_id">-->
              <!--              <el-input v-model.number="menuForm.sort_id" style="width: 360px"/>-->
              <!--            </el-form-item>-->
            </el-col>
          </el-row>
          <el-divider/>
          <el-row>
            <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
              <el-form-item label="菜单图标" prop="meta.icon">
                <el-popover
                    ref="popover"
                    placement="right"
                    width="880"
                    popper-class="icon-popper"
                    trigger="click"
                >
                  <template #reference>
                    <el-button :icon="menuForm.meta.icon"/>
                  </template>
                  <div class="icon-content">
                    <el-row>
                      <el-col :span="4" v-for="(name,idx) in iconList" :key="idx" :icon="name"
                              style="text-align: center">
                        <div @click="selectIcon(name)" style="border:1px solid #cccccc;padding: 10px">
                          <!--                              <el-button :key="idx" :icon="name" size="large" style="margin-bottom: 5px"></el-button>-->
                          <component :is="name" style="width: 24px"></component>
                          <div style="font-size: 12px">{{ name }}</div>
                        </div>
                      </el-col>
                    </el-row>
                  </div>
                </el-popover>
              </el-form-item>
              <el-form-item label="菜单路径" prop="path">
                <el-input v-model="menuForm.path" :disabled="menuForm.type!==1" style="width: 360px"/>
              </el-form-item>
              <el-form-item label="组件路径" prop="component">
                <el-input v-model="menuForm.component" :disabled="menuForm.type!==1" style="width: 360px"/>
              </el-form-item>
              <el-form-item label="权限标识" prop="meta.perms">
                <el-input v-model="menuForm.meta.perms" :disabled="menuForm.type===1" style="width: 360px"/>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
              <el-form-item label="是否固定" prop="meta.fixed">
                <el-radio-group v-model="menuForm.meta.fixed" :disabled="menuForm.type!==1">
                  <el-radio :label="false">不固定</el-radio>
                  <el-radio :label="true">固定</el-radio>
                </el-radio-group>
              </el-form-item>
              <el-form-item label="是否隐藏" prop="meta.hidden">
                <el-radio-group v-model="menuForm.meta.hidden">
                  <el-radio :label="false">显示</el-radio>
                  <el-radio :label="true">隐藏</el-radio>
                </el-radio-group>
              </el-form-item>
              <el-form-item label="是否全屏" prop="meta.is_full">
                <el-radio-group v-model="menuForm.meta.is_full">
                  <el-radio :label="true">是</el-radio>
                  <el-radio :label="false">否</el-radio>
                </el-radio-group>
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
.table {
  margin: 10px 0;
}

//图标气泡卡片
.icon-popper {
  max-height: 500px;
}

.icon-content {
  max-height: 650px;
  width: 100%;
  overflow: auto;
  overflow-x: hidden;
}
</style>