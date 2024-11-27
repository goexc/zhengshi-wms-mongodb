<script setup lang="ts">
import {nextTick, ref, onMounted, reactive} from "vue";
import {Sizes, Types} from "@/utils/enum.ts";
import {TimeFormat} from "@/utils/time.ts";
import {Api, ApiListResponse} from "@/api/acl/api/types.ts";
import {reqApiList, reqDeleteApi} from "@/api/acl/api";
import {ElMessage, FormInstance} from "element-plus";
import {reqAddOrUpdateApi} from "@/api/acl/api";
import {rules} from "@/views/acl/api/rules.ts";
import {Methods} from "@/enums/http.ts";
import {sortTree} from "@/utils/sortTree.ts";


//表格展开属性
const expand = ref<boolean>(true)
const tableVisible = ref<boolean>(true)
//Api展开
const switchExpand = (val: boolean) => {
  tableVisible.value = false
  expand.value = val

  nextTick(() => {
    tableVisible.value = true
  })
}

const apis = ref<Api[]>([])
const getApis = async () => {
  let res: ApiListResponse = await reqApiList()
  if (res.code === 200) {
    apis.value = sortTree(res.data)
  }
}

onMounted(() => {
  getApis()
})


const initApiForm = () => {
  return <Api>({
    id: '',
    parent_id: '',
    type: 1,//类型：1.模块，2.API
    sort_id: 1,
    required: false,
    uri: '',
    name: '',
    method: Methods.EMPTY,
    remark: '',
  })
}

//dialog可见
const visible = ref<boolean>(false)
const title = ref<string>('')
const action = ref<string>('') //表单动作：addApi，新增子Api； editApi，修改Api
const apiForm = reactive<Api>(initApiForm())

//表单校验
const formRef = ref<FormInstance>()

//添加子Api
const addApi = (parent: Api) => {
  title.value = '新增Api'
  action.value = 'addApi'
  visible.value = true
  Object.assign(apiForm, initApiForm())
  apiForm.parent_id = parent.id
}

//修改Api
const editApi = (api: Api) => {
  title.value = '修改Api'
  action.value = 'editApi'
  visible.value = true
  Object.assign(apiForm, api)
}

//删除Api
const deleteApi = async (id: string) => {

  let res = await reqDeleteApi({id:id})
  if (res.code === 200) {
    ElMessage.success(res.msg)
    await getApis()
  } else {
    ElMessage.error(res.msg)
  }
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
  console.log('表单校验:', valid)


  let res = await reqAddOrUpdateApi(apiForm)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    handleClose()
    await getApis()
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
  // Object.assign(apiForm, initApiForm())
}
</script>

<script lang="ts">

</script>

<template>
  <div class="container">
    <div id="auth">
      <el-button plain @click="switchExpand(true)" size="default" icon="ArrowDown">展开全部</el-button>
      <el-button plain @click="switchExpand(false)" size="default" icon="ArrowUp">折叠全部</el-button>

      <perms-button
          perms="privilege:api:add"
          :type="Types.primary"
          :size="Sizes.default"
          :plain="true"
          @action="addApi(initApiForm())"
      />
      <!--   展示Api列表   -->
      <el-table
          v-if="tableVisible"
          class="table"
          stripe
          border
          :show-overflow-tooltip="true"
          tooltip-effect="dark"
          :data="apis"
          row-key="id"
          :default-expand-all="expand"
          :tree-props="{ children: 'children', hasChildren: '!!children' }"
      >

        <template #empty>
          <el-empty/>
        </template>
        <el-table-column prop="name" label="名称" min-width="220px" fixed>
          <template #default="{row}">
            <el-link :icon="row.icon">{{ row.name }}</el-link>
          </template>
        </el-table-column>
        <el-table-column label="类型" prop="type" width="90px" align="center">
          <template #default="{row}">
            <el-tag size="default" :type="row.type===1?'success':'warning'">{{row.type===1?'菜单':'API'}}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="排序" prop="sort_id" width="80px" align="center"></el-table-column>
        <el-table-column label="URI" prop="uri" width="220px"></el-table-column>
        <el-table-column label="方法" prop="method" align="center" width="90px"></el-table-column>
        <el-table-column label="必选" prop="required" width="220px"></el-table-column>
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
            <perms-button
                v-if="row.type === 1"
                perms="privilege:api:add"
                :type="Types.primary"
                :size="Sizes.small"
                :plain="true"
                @action="addApi(row)"
            />
            <perms-button
                perms="privilege:api:edit"
                :type="Types.warning"
                :size="Sizes.small"
                :plain="true"
                @action="editApi(row)"
            />
            <el-popconfirm
                :title="`确定删除Api[${row.name}]吗?`"
                icon="InfoFilled"
                icon-color="#F56C6C"
                cancel-button-text="取消"
                confirm-button-text="确认删除"
                cancel-button-type="info"
                confirm-button-type="danger"
                @confirm="deleteApi(row.id)"
                width="300"
            >
              <template #reference>
                <perms-button
                    perms="privilege:api:delete"
                    :type="Types.danger"
                    :size="Sizes.small"
                    :plain="true"/>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
      <el-dialog
          v-model.trim="visible"
          :title="title"
          draggable
          width="800"
          :close-on-click-modal="false"
          @close="handleClose"
      >
        <el-form ref="formRef" :model="apiForm" :rules="rules" label-width="100">
          <el-row>
            <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
              <el-form-item label="Api类型" prop="type">
                <el-radio-group v-model.number="apiForm.type" :disabled="action==='editApi'">
                  <el-radio :label="1">模块</el-radio>
                  <el-radio :label="2">API</el-radio>
                </el-radio-group>
              </el-form-item>
              <el-form-item label="必选" prop="required">
                <el-radio-group v-model.number="apiForm.required">
                  <el-radio :label="true">是</el-radio>
                  <el-radio :label="false">否</el-radio>
                </el-radio-group>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
              <el-form-item label="Api名称" prop="name">
                <el-input v-model.trim="apiForm.name" placeholder="例如：权限管理"/>
              </el-form-item>
              <el-form-item label="Api排序" prop="sort_id">
                <el-input v-model.number="apiForm.sort_id"/>
              </el-form-item>
            </el-col>
          </el-row>
          <el-divider/>
          <el-row>
            <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
              <el-form-item label="方法" prop="method" v-if="apiForm.type!==1">
                <el-select filterable v-model.trim="apiForm.method" :disabled="apiForm.type===1">
                  <el-option v-for="($item, $index) in Methods" :key="$index" :label="$item" :value="$item"></el-option>
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
              <el-form-item label="URI" prop="uri" v-if="apiForm.type!==1">
                <el-input v-model.trim="apiForm.uri" :disabled="apiForm.type===1"/>
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

.body-card {
  margin: 10px 0;
}

.table {
  margin: 10px 0;
}

</style>