//系统设置
import {defineStore} from "pinia";

const useSettingStore = defineStore('Setting', {
  state: () => {
    return {
      theme: '', //默认为空。dark:夜间模式
      color: '', //自定义主题颜色
    }
  },
  actions: {
    //切换主题
    changeTheme(night: boolean) {
      //获取html根节点
      let html = document.documentElement

      night ? this.theme = 'dark' : this.theme = ''
      night ? html.className = 'dark' : html.className = '';
    },
    //更改主题颜色
    changeColor  (color:string)  {
      //获取html根节点
      let html = document.documentElement

      //通知js修改根节点的样式
      this.color = color
      html.style.setProperty('--el-color-primary', color)
    },
    //初始化设置
    initSetting(){
      console.log('初始化设置')
      //获取html根节点
      let html = document.documentElement
      html.className = this.theme
      html.style.setProperty('--el-color-primary', this.color)
    }
  },
  getters: {},
  persist: {
    enabled: true,
    strategies: [
      {
        key: "p_setting",
        storage: localStorage,
        // 可以选择哪些进入local存储，这样就不用全部都进去存储了
        // 默认是全部进去存储
        // paths: ['token']
      },
    ],
  }
})

export default useSettingStore;