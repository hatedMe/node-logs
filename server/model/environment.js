
const mongoose = require('mongoose');

const WebEnvironmentSchema = new mongoose.Schema({
    appId: { type: String }, // 所属系统
    createTime: { type: Date, default: Date.now }, // 用户访问时间
    url: { type: String }, // 访问页面的url
    markPage: { type: String }, // 所有资源页面统一标识 html img css js 用户系统信息等
    uvmark: { type: String }, // 统一某一时间段用户标识
    uv: { type: String }, // 统一uv标识
    browserName: { type: String }, // 浏览器名称
    borwserVersion: { type: String }, // 浏览器版本
    system: { type: String }, // 操作系统
    systemVersion: { type: String }, // 系统版本
    ip: { type: String }, // 访问者IP
    county: { type: String }, // 国家
    province: { type: String }, // 省
    city: { type: String }, // 市
}, {
    shardKey: { _id: 'hashed' },
    versionKey: false ,
});

WebEnvironmentSchema.index({ appId: 1, createTime: -1 });
WebEnvironmentSchema.index({ url: 1, createTime: -1 });
WebEnvironmentSchema.index({ ip: 1, createTime: -1 });
WebEnvironmentSchema.index({ createTime: -1 });
WebEnvironmentSchema.index({ markPage: 1 });
WebEnvironmentSchema.index({ uvmark: 1 });


module.exports = WebEnvironment = function(appId) {
    return mongoose.model(`atomweb_environment_${appId}`, WebEnvironmentSchema);
};
