
const { redis } = require('../RedisInit');

class PvuvipsTask {





    // 通过redis取到相关数据
    static async getUserRepotDataForRedis() {
        const result = await redis.rpopAsync('WEB_DATA_COUNT');
        console.log( 'redis rpop value' , result );
        if( !result ) return ;
        return await this.filterDataAndSaveData(result);
    }

    // 帅选数据分类并保存
    static filterDataAndSaveData ( response ) {

    }


}



module.exports = PvuvipsTask;
