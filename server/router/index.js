const Router = require('koa-router');
const koaBody = require('koa-body')();

const router = new Router({
    prefix : '/pingd'
});

const user = require('./user');
const front = require('./front');

router.use('*' , koaBody , async (ctx, next) => {
    await next();
});


router.use('/api/user', user.routes(), user.allowedMethods());
router.use('/front' , front.routes(), front.allowedMethods());




module.exports = router;