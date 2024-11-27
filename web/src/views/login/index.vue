<script setup lang="ts">
import {onMounted, reactive, ref} from "vue";
import {useRoute, useRouter} from "vue-router";
import {ElMessage, ElNotification, FormInstance, FormRules} from "element-plus";
import {useUserStore} from "@/store/modules/user.ts";
import {reqAccountInfo, reqLogin} from "@/api/user";
import {initDynamicRouter} from "@/router/modules/dynamicRouter.ts";
import {HOME_URL} from "@/config";

let loading = ref<boolean>(false)

//表单数据
let loginForm = reactive({mobile: '', password: ''})

//引入用户相关的小仓库
let userStore = useUserStore()

//获取路由器
let router = useRouter()
let route = useRoute()

//表单校验
const formRef = ref<FormInstance>()
//校验规则
const rules = reactive<FormRules>({
  mobile: [
    { required: true, message: '请填写手机号码', trigger: ['blur', 'change'] },
    { len: 11, message: '手机号码格式错误', trigger: ['blur', 'change'] },
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
    // 1.执行登录接口
   let res =  await reqLogin(loginForm)
   if(res.code != 200){
     ElMessage.error(res.msg)
     return
   }
    userStore.setToken(res.data.token);

    // 2.获取用户信息
    let { data } = await reqAccountInfo()
    userStore.setAccount(data)

    // 3.添加动态路由
    await initDynamicRouter();



    // 3.清空 tabs、keepAlive 数据
    // tabsStore.closeMultipleTab();
    // keepAliveStore.setKeepAliveName();

    // 4.跳转到首页
    let redirect = route.query.redirect as string
    await router.push({path: redirect||HOME_URL})

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
  } finally {
    loading.value = false;
  }

}

onMounted(()=>{
  // 监听 enter 事件（调用登录）
  document.onkeydown = (e: KeyboardEvent) => {
    e = (window.event as KeyboardEvent) || e;
    if (e.code === "Enter" || e.code === "enter" || e.code === "NumpadEnter") {
      if (loading.value) return;
      login()
    }
  };
})
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
          <el-form-item prop="mobile">
            <el-input prefix-icon="User" v-model.trim="loginForm.mobile" placeholder="请填写手机号码"></el-input>
          </el-form-item>
          <el-form-item prop="password">
            <el-input prefix-icon="Lock" :show-password="true" v-model.trim="loginForm.password" placeholder="请填写密码"></el-input>
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