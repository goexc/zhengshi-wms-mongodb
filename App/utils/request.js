//封装请求方法
import {base_api} from '@/config/index.js'
import useUserStore from '@/store/user.js'

let $request = function(url,data,method='GET'){
	return new Promise((resolve)=>{
		let userStore = useUserStore()
		uni.request({
			header:{
				Authorization: userStore.user.token
			},
			url: base_api+url,
			data:data,
			method:method,
			success:({data})=>{
				resolve(data)
			}
		})
	})
}

uni.$request = $request

uni.$get = function(url,data){
	return $request(url, data, 'GET')
}

uni.$post = function(url,data){
	return $request(url, data, 'POST')
}