const env = process.env

module.exports = {
    serverPort: env.serverPort || 4010,

    mongoHost: env.mongoHost || '127.0.0.1',
    mongoDatabase: env.mongoDatabase || 'nodelogs',
    mongoPort: env.mongoPort || 27017,

    redisHost: env.redisHost || '127.0.0.1',
    redisPort: env.redisPort || 6379,
    redisPassword: env.redisPassword || 'password',

    tokenSecret: env.tokenSecret || 'tokensecret',
    tokenExpiresIn: env.tokenExpiresIn || 7200 * 2,

    defaultAdminName: env.defaultAdminName || 'admin',
    defaultAdminPassword: env.defaultAdminPassword || '123456'
}