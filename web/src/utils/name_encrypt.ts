export const nameEncrypt = (name:string) => {
    if(!name){
        return ''
    }

    name = name.replace(/有限责任公司/g, '')
    name = name.replace(/有限公司/g, '')
    name = name.replace(/股份/g, '')

    name = name.replace(/科技/g, '')
    name = name.replace(/机械/g, '')
    name = name.replace(/制造/g, '')
    name = name.replace(/部件/g, '')
    name = name.replace(/汽车/g, '')
    name = name.replace(/农业/g, '')
    name = name.replace(/装备/g, '')
    name = name.replace(/工贸/g, '')

    name = name.replace(/山东省/g, '')
    name = name.replace(/山东/g, '')
    name = name.replace(/潍坊市/g, '')
    name = name.replace(/潍坊/g, '')
    name = name.replace(/诸城市/g, '')
    name = name.replace(/诸城/g, '')

    return name
}