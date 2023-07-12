<script setup lang="ts">
import {reactive, ref} from "vue";
import useUserStore from "@/store/module/account.ts";
import {useRoute, useRouter} from "vue-router";
import {ElNotification, FormInstance, FormRules} from "element-plus";

let loading = ref<boolean>(false)

//表单数据
let loginForm = reactive({name: 'lisi', password: '111111'})

//引入用户相关的小仓库
let userStore = useUserStore()

//获取路由器
let router = useRouter()
let route = useRoute()

//表单校验
const formRef = ref<FormInstance>()
//校验规则
const rules = reactive<FormRules>({
  name: [
    { required: true, message: '请填写用户名', trigger: ['blur', 'change'] },
    { min: 2, max: 21, message: '用户名应为 2 ~ 21 个字符', trigger: ['blur', 'change'] },
  ],
  password: [
    { required: true, type:"string", message: '请填写密码', trigger: ['blur', 'change'] },
    { min: 6, max: 16, message: '密码应为 6 ~ 16 个字符', trigger: ['blur', 'change'] },
  ]
})

//登录请求
const login = async () => {
  //表单校验
  let validate = await formRef.value?.validate((valid, fields) => {
    if(valid){

    }else{
      console.log('fields:', fields)
    }
    return
  })

  if(!validate){
    return
  }

  loading.value = true
  try {
    //保证登录成功
     await userStore.login(loginForm)

    let redirect = route.query.redirect as string
    await router.push({path: redirect||'/'})

    ElNotification({
      type: 'success',
      title: '你好',
      message: '欢迎回来'
    })
  } catch (e) {
    //登录失败，关闭加载效果
    loading.value = false

    ElNotification({
      type: 'error',
      message: (e as Error).message
    })
  }

}
</script>

<template>
  <div class="login_container">
    <el-row>
      <el-col :span="12" :xs="0" class="container_left">
        <div class="logo"></div>
      </el-col>
      <el-col :span="12" :xs="24">
        <el-form
            class="login_form"
            ref="formRef"
            :model="loginForm"
            :rules="rules"
        >
          <h1>你好</h1>
          <h2>欢迎来到 正时WMS</h2>
          <el-form-item prop="name">
            <el-input prefix-icon="User" v-model="loginForm.name" placeholder="请填写用户名"></el-input>
          </el-form-item>
          <el-form-item prop="password">
            <el-input prefix-icon="Lock" :show-password="true" v-model="loginForm.password" placeholder="请填写密码"></el-input>
          </el-form-item>
          <el-form-item>
            <el-button :loading="loading" class="login_btn" type="primary" size="default" @click="login">登录
            </el-button>
          </el-form-item>
        </el-form>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped lang="scss">
.login_container {
  width: 100%;
  height: 100vh;
  background: url("@/assets/images/background.jpg") no-repeat;
  background-size: cover;
}

.el-row {
  height: 100%;
}

.logo {
  position: relative;
  width: 80%;
  top: 20vh;
  height: 80vh;
  background: url("@/assets/images/team.svg") no-repeat;
  background-size: cover;
}

.login_form {
  position: relative;
  width: 80%;
  top: 30vh;
  background: url("@/assets/images/login_form.png") no-repeat;
  background-size: cover;
  padding: 40px;

  h1 {
    color: white;
    font-size: 40px;
  }

  h2 {
    color: white;
    font-size: 20px;
    margin: 20px 0;
  }
}

.login_btn {
  width: 100%;
}
</style>