const config = require('../conf/config');
const jwt = require('jsonwebtoken');
const crypto = require('crypto');
const { Schema } = require('../DBinit');
const APPID_MODEL = require('../model/appid');
const appIdsModle = require('../model/appid');
const { redis } = require('../RedisInit');
const Status = require('../util/response');

function getClientIp(req) {
    return req.headers['x-forwarded-for'] ||
        req.connection.remoteAddress ||
        req.socket.remoteAddress ||
        req.connection.socket.remoteAddress;
};

class Front {

    static async web (ctx , next) {
        ctx.type = "text/html;";
        ctx.body = "";

        const { dm , pvid , uvmark , url , arg , v , scr , lg , tz } = ctx.request.query;
        const ua = ctx.headers['user-agent'];
        const ip = getClientIp(ctx.req);

        // 先验证appid 
        if( !pvid ) return ;
        // 验证appid是否在已经注册

        // 消息生产者  
        const userdataStr = JSON.stringify( Object.assign({} , { ip , ua } , ctx.request.query) );
        const redisKey = `WEB_DATA_COUNT`;
        await redis.lpush( redisKey , [ userdataStr ]);
        const lists = await redis.lrangeAsync( redisKey , 0 , -1 );
        redis.expire(redisKey , 1 * 60 * 60 * 24 * 7 );  // 设置过期时间为7天
    }

}


module.exports = Front;