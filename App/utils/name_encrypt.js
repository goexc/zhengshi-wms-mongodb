
//名称加密
export const nameEncrypt = (name) => {
		name = name.replaceAll('有限责任公司', '')
		name = name.replaceAll('有限公司', '')
		name = name.replaceAll('股份', '')
		
		name = name.replaceAll('科技', '')
		name = name.replaceAll('机械', '')
		name = name.replaceAll('制造', '')
		name = name.replaceAll('部件', '')
		name = name.replaceAll('汽车', '')
		name = name.replaceAll('农业', '')
		name = name.replaceAll('装备', '')
		name = name.replaceAll('工贸', '')
		
		name = name.replaceAll('山东省', '')
		name = name.replaceAll('山东', '')
		name = name.replaceAll('潍坊市', '')
		name = name.replaceAll('潍坊', '')
		name = name.replaceAll('诸城市', '')
		name = name.replaceAll('诸城', '')
		
		return name
}