const Koa = require('koa');
const Router = require('koa-router');
const koaBody = require('koa-body')();

const Redis = require('./RedisInit');
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

    console.log( dm , pvid , uvmark , url , arg , v , scr , lg , tz );

    await new logsModel({ 
        dm , pvid , uvmark , url , arg , v , scr , lg , tz ,
        creatime : Date.now() ,
        ua , 
        ip ,
     }).save();

    // await client.rpush("rpush", [ JSON.stringify( { dm , pvid , url , arg , v , scr , lg , tz , random } ) ]);

    // const redisKey = "DATA_" + new Date().getHours();
    // await client.rpush( redisKey , [ JSON.stringify( { dm , pvid , url , arg , v , scr , lg , tz , random } ) ]);

    // const lists = await client.lrange( redisKey , 0 , -1 ,function (err, v) { 
    //     console.log( err ,v  );
    // });
    // client.expire(redisKey , 1800000 );

    
});

router.post('/findnum' , async ctx => {
    ctx.type = "text/html;";
    ctx.body = "";
    const { dm , pvid , url , arg , v , scr , lg , tz , random } = ctx.request.query;
    console.log( dm , pvid , url , arg , v , scr , lg , tz , random , "post" );
});

app.use(router.routes()).use(router.allowedMethods());
app.listen(3000);