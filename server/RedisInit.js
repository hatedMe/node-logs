const redis = require('redis');
const bluebird = require('bluebird');
const config = require('./conf/config.js');

bluebird.promisifyAll(redis.RedisClient.prototype);
bluebird.promisifyAll(redis.Multi.prototype);

const auth = { password : config.redisPassword };
const client = redis.createClient(Object.assign({}, auth, {
    host: config.redisHost,
    port: config.redisPort,
    // db : 1 ,
}));

client.on('error', function(err) {
   console.error('Redis Error ' + err)
})

const connect = () => {
    client.on('connect', function() {
        console.log('Redis is ready')
    })
}


exports.client = client;
exports.redis = client;
exports.connect = connect;