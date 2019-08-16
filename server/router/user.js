
const Router = require('koa-router');

const router = new Router();

const userController = require('../controller/user');


router.post('/register' , userController.register );
router.post('/login' , userController.login );




module.exports = router;







