

const Koa = require('koa');
const schedule = require('node-schedule');
const path = require('path');
const fs = require('fs');
const Redis = require('./RedisInit');
const DB = require('./DBInit');
const router = require('./router/index');
Redis.connect();
DB.connect();
const app = new Koa();
app.use(router.routes()).use(router.allowedMethods());
app.listen(3005);

// 批量执行定时器
function getFileList(dir_path) {
    let file_list = []
    if (fs.existsSync(dir_path)) {
        file_list = fs.readdirSync(dir_path);
    }
    return file_list;
};

const schedules = getFileList( path.join( __dirname , './schedule' ) );
schedules.forEach(item => {
    const scheduleItem = require(  path.join( __dirname , './schedule/' + item ) );
    schedule.scheduleJob( scheduleItem.rule, scheduleItem.task );
});



