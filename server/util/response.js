


module.exports = {
    success (data) {
        return {
            code : 200 ,
            message : 'ok' ,
            data ,
        }
    },
    error (code , message , data = null) {
        return {
            code : code || 400400,
            message : message || 'request is error' ,
            data,
        }
    }
};

