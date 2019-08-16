const Koa = require('koa');
const Router = require('koa-router');
const koaBody = require('koa-body')();

const Redis = require('./RedisInit');
const { redis } = require('./RedisInit');

const DB = require('./DBInit');
const logsModel = require('./model/access_log');
const WebEnvironment = require('./model/environment');

const app = new Koa();
const router = new Router({
    prefix : '/tongji'
});

Redis.connect();
DB.connect();

function getClientIp(req) {
    return req.headers['x-forwarded-for'] ||
        req.connection.remoteAddress ||
        req.socket.remoteAddress ||
        req.connection.socket.remoteAddress;
};

router.get('/findnum' , async ctx => {

    ctx.type = "text/html;";
    ctx.body = "";

    const { dm , pvid , uvmark , url , arg , v , scr , lg , tz } = ctx.request.query;
    const ua = ctx.headers['user-agent'];
    const ip = getClientIp(ctx.req);

    // 先验证appid 
    if( !pvid ) return ;
    // 验证appid是否在已经注册
    

    // 消息生产者
    const userdataStr = JSON.stringify( { dm , pvid , uvmark ,  url , arg , v , scr , lg , tz , ua , ip } );
    const redisKey = `WEB_DATA_COUNT_${pvid}`;
    await redis.rpush( redisKey , [ userdataStr ]);
    const lists = await redis.lrangeAsync( redisKey , 0 , -1 );
    redis.expire(redisKey , 1 * 60 * 60 * 24 * 7 );  // 设置过期时间为7天

    // new WebEnvironment(pvid)({
    //     app_id : pvid
    // }).save();
    
});

app.use(router.routes()).use(router.allowedMethods());
app.listen(3000);