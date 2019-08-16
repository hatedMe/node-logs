const config = require('../conf/config');
const jwt = require('jsonwebtoken');
const crypto = require('crypto');
const { Schema } = require('../DBinit');
// const User = require('../model/user');
const appIdsModle = require('../model/appid');
const Redis = require('../redisInit');
const Status = require('../util/response');


function md5 (str) {
    return crypto.createHash('md5').update(str, 'utf8').digest('hex').toUpperCase();
}

const createRandom = () => {
    return Math.random().toString(36).slice(2,16);
}

class Users {
    static async login (ctx , next) {
        const {name, password} = ctx.request.body;
        const result = await User.findOne({name, password});
        if( !name || !password) return ctx.body = Status.error( 400 , '缺少相关参数');
        if( !result ) return ctx.body = Status.error( 400 ,'账号密码不对称');
        const token = jwt.sign({ userInfo: `${name}` }, 'token',{ expiresIn: config.tokenExpiresIn });
        await Redis.client.set('token', token,'EX', config.tokenExpiresIn);
        ctx.body = Status.success({token});
    }

    static async register (ctx ,next ) {
        const { name , category , username , password , domain } = ctx.request.body;

        if( !name || !category || !username || !password ) {
            return ctx.body = Status.error(400450 , '缺少必传参数');
        }

        const appid_number = await appIdsModle.findOne().sort({ "appId" : -1 }).limit(1);

        const _data = {};
        if( appid_number ) {
            _data.appId = appid_number.appId + 1 ;
        }

        await new appIdsModle(Object.assign({
            secretKey : md5( createRandom() ),
            name ,
            domain ,
            category , 
            username , 
            password : md5( password ),
        }, _data)).save();

        return ctx.body = Status.success();
    }

}


module.exports = Users;