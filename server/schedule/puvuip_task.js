const schedule = require('node-schedule');

// const rule = new schedule.RecurrenceRule();
// rule.dayOfWeek = [0, new schedule.Range(1, 6)];
// rule.hour = 12;
// rule.minute = [ 5 , 10 , 15 , 20 , 25 , 32 ,38 , 40 , 45 ,50 ,55 , 60 ];

const PvuvipsTask = require('../service/puuvips_task');
module.exports = {
    rule : "15 * * * * *",
    async task () {
        console.log('提取数据redis数据定时任务执行时间 --------------->' + new Date() );
        await Promise.resolve( PvuvipsTask.getUserRepotDataForRedis() );
    }
};
