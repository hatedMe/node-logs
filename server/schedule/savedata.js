const fs = require('fs')
const schedule = require('node-schedule');
var Redis , { redis } = require('../RedisInit');
const DB = require('../DBinit');
const APPID_MODLE = require('../model/appid');
Redis.connect();
DB.connect();


const rule = new schedule.RecurrenceRule();
// rule.dayOfWeek = [0, new schedule.Range(1, 6)];
// rule.hour = globalData.sendTime.EmailHour;
// rule.minute = globalData.sendTime.EmialMinminute;


async function init () {
    const usersLists = await APPID_MODLE.find();
    for (let index = 0; index < usersLists.length; index++) {
        console.log(usersLists[index].appid);
    }
}

const scheduleCronstyle = async () =>{
    await schedule.scheduleJob( "30 * * * * *" , ()=>{
        console.log('提取数据redis数据定时任务执行时间 --------------->' + new Date() );
        init();
    });
}

scheduleCronstyle();