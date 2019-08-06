const mongoose = require('mongoose');

const access_log = new mongoose.Schema({
    // 路径
    dm : String,
    // 统计id标识
    pvid : String ,
    // 用户标识
    uvmark : String ,
    // pathname
    url : String , 
    // URL查询参数
    arg : String ,
    v : String ,
    // 屏幕区域
    scr : String,
    // 语言
    lg : String,
    // 时区
    tz : String ,
    // 产生记录时间
    creatime : Date ,
    // ip 用户
    ip : String ,
    // 用户浏览器标识
    ua : String ,

},{ versionKey: false });


module.exports = mongoose.model('access_log', access_log , 'access_log' );