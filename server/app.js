const Koa = require('koa');
const Router = require('koa-router');
const koaBody = require('koa-body')();

const Redis = require('./RedisInit');
const { redis } = require('./RedisInit');

const DB = require('./DBinit');
const logsModel = require('./model/access_log');

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

    const { dm , pvid , uvmark ,  url , arg , v , scr , lg , tz } = ctx.request.query;
    const ua = ctx.headers['user-agent'];
    const ip = getClientIp(ctx.req);

    const val = await redis.getAsync( 'UVMARK_' + uvmark );
    if( val ) return ;
    await redis.set('UVMARK_' + uvmark , uvmark ,'EX',  1 );
    
    const userdataStr = JSON.stringify( { dm , pvid , uvmark ,  url , arg , v , scr , lg , tz , ua , ip } );
    const redisKey = "DATA_" + new Date().getHours();
    await redis.rpush( redisKey , [ userdataStr ]);

    const lists = await redis.lrangeAsync( redisKey , 0 , -1 );
    redis.expire(redisKey , 7200 );
    
});

app.use(router.routes()).use(router.allowedMethods());
app.listen(3000);