const Router = require('koa-router');
const koaBody = require('koa-body')();

const router = new Router({
    prefix : '/api'
});

const user = require('./user');

router.use('*' , koaBody , async (ctx, next) => {
    await next();
});


router.use('/user', user.routes(), user.allowedMethods());




module.exports = router;