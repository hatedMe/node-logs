
const { redis } = require('../RedisInit');
const UAParser = require('ua-parser-js');
const parser = new UAParser();
const ENVIRONMENT_MODEL = require('../model/environment');

class PvuvipsTask {

    // 通过redis取到相关数据
    static async getUserRepotDataForRedis() {
        const loopNum = 500;   // 每次最多拿5000次拿来循环
        const redisDataLen = await redis.lrangeAsync( 'WEB_DATA_COUNT' , 0 , -1 );
        const len = redisDataLen == 0 ? 0 : 
            redisDataLen.length > loopNum ? loopNum : redisDataLen.length;
        if( !len ) {
            console.log('redis 定时器没有发现数据');
            return ;
        }
        for (let index = 0; index < len; index++) {
            const result = await redis.rpopAsync('WEB_DATA_COUNT');
            if( !result ) return ;
            await this.filterDataAndSaveData(result);
        }
    }

    // 对单个数据进行帅选分类并保存
    static async filterDataAndSaveData ( response ) {
        const data = JSON.parse(response);
        parser.setUA(data.ua)
        const ua = parser.getResult();
        console.log( ua );
        await ENVIRONMENT_MODEL(data.pvid)({
            appId : data.pvid ,
            createTime : data.d,
            url: data.url, // 访问页面的url
            uvmark: data.uvmark, // 统一某一时间段用户标识
            browserName: ua.browser.name, // 浏览器名称
            borwserVersion: ua.browser.version, // 浏览器版本
            system: ua.os.name, // 操作系统
            systemVersion: ua.os.version, // 系统版本
            // ip: { type: String }, // 访问者IP
            // county: { type: String }, // 国家
            // province: { type: String }, // 省
            // city: { type: String }, // 市
        }).save();

    }




}



module.exports = PvuvipsTask;
