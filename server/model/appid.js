const mongoose = require('mongoose');

const appId = new mongoose.Schema({
    appId : { type : Number , default : 800800 } ,  // 应用id 
    secretKey : String ,  // 应用唯一secretKey 
    name : String ,  // 应用名字
    category : String ,  // 应用类型 
    domain : String ,   // 站点域名 
    username : String ,  // 用户名
    password : String ,  // 用户密码
    customEvent : { type : Boolean , default : false } , // 是否开启自定义事件
    performance : { type : Boolean , default : false },  // 是否开启性能监控
    createTime: { type: Date, default: Date.now }, // 创建事件
},{
    shardKey: { _id: 'hashed' },
    versionKey: false , 
});

appId.index({ appId: 1, createTime: -1 });

module.exports = mongoose.model('appId', appId );