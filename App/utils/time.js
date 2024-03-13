import * as dayjs from 'dayjs'
 
//时间格式
export const timeFormat = (timestamp) => {
  // return dayjs.unix(timestamp).format('YYYY-MM-DD HH:mm:ss')
  if(!timestamp){
	  return '-'
  }
  return dayjs.unix(timestamp).format('YYYY-MM-DD HH:mm:ss')
}
 
//日期格式
export const dateFormat = (timestamp) => {
  // return dayjs.unix(timestamp).format('YYYY-MM-DD HH:mm:ss')
  if(!timestamp){
	  return '-'
  }
  return dayjs.unix(timestamp).format('YYYY-MM-DD')
}