const config = require('../conf/config');
const jwt = require('jsonwebtoken');
const crypto = require('crypto');
const { Schema } = require('../DBInit');
const APPID_MODEL = require('../model/appid');
const appIdsModle = require('../model/appid');
const { redis } = require('../RedisInit');
const Status = require('../util/response');


function md5 (str) {
    return crypto.createHash('md5').update(str, 'utf8').digest('hex').toUpperCase();
}

const createRandom = () => {
    return Math.random().toString(36).slice(2,16);
}

class Users {

    static async login (ctx , next) {
        const { username , password} = ctx.request.body;
        if( !username || !password) return ctx.body = Status.error( 400 , '缺少相关参数');
        const result = await APPID_MODEL.findOne({ username , password : md5( password ) });
        if( !result ) return ctx.body = Status.error( 400 ,'账号密码不对称');
        const token = jwt.sign({ userInfo: `${username}` }, 'token', { expiresIn: 7200 });
        await redis.set('USER_LOGIN_TOKEN_' + username , token,'EX', 7200 );
        ctx.body = Status.success({
            token , 
            username,
            expiresIn : 7200
        });
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