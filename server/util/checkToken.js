




const client = require('../model/redis');



module.exports = async function checkToken(ctx,next) {
    let isToken = () =>{
        return new Promise((resolve, reject) => {
            client.get('token',(err,token) => {
                resolve(err !== null ? err : token)
            });
        })
    }
    
    let accessToken = ctx.request.header['access-token'] 
        || ctx.request.body['access-token'] 
        || (typeof ctx.request.body.fields === 'undefined' ?  undefined : ctx.request.body.fields['access-token']);


    if( ctx.request.method === 'POST' && ( typeof accessToken === 'undefined' || await isToken() !== accessToken ) ) {
        //return ctx.body = JSON.parse(`{"status": "400120","message":"token is error"}`);
        ctx.response.state = 417
        return ctx.throw(417,'token is error')
    }else{
        return next();
    }
    //return next();
}