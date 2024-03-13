import {defineStore} from 'pinia'


const useUserStore = defineStore('user', {
	state: () => {
		return {
			//登录用户信息
			user: {
				name: '',
				avatar: '',
				mobile: '',
				email: '',
				department_id: '', //部门id
				department_name: '', //部门名称
				token: '',
				exp: 0
			}
		}
	},
	actions: {
		setUser(user){
			this.user = {
				name: user.name,
				avatar: user.avatar,
				mobile: user.mobile,
				email: user.email,
				department_id: user.department_id, //部门id
				department_name: user.department_name, //部门名称
				token: user.token,
				exp: user.exp
			}
		},
		logout(){
			console.log('清空登录信息')
			this.user = {
				name: '',
				avatar: '',
				mobile: '',
				email: '',
				department_id: '', //部门id
				department_name: '', //部门名称
				token: '',
				exp: 0
			}
		}
	},
	unistorage: true // 开启后对 state 的数据读写都将持久化
})

export default useUserStore;