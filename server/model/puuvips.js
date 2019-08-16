const mongoose = require('mongoose');

const webDefaultCount = new mongoose.Schema({
    appId : { type : Number } ,         // 应用所属id 
    pv: { type: Number },                // PV统计
    uv: { type: Number },               // uv统计
    ip: { type: String },               // ip统计
    bounce: { type: String },           // 跳出率
    createTime: { type: Date, default: Date.now },     // 创建事件
},{
    shardKey: { _id: 'hashed' },
    versionKey: false , 
});

webDefaultCount.index({ appId: 1, createTime: -1 });
webDefaultCount.index({ ip : 1 });
webDefaultCount.index({ appId: 1, ip : -1 });

module.exports = appId => {
    return mongoose.model(`webDefaultCount_${appId}`, webDefaultCount );
}