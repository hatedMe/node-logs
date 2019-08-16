
const Router = require('koa-router');
const router = new Router();


const userController = require('../controller/front');

router.get('/web' , userController.web );

module.exports = router;







