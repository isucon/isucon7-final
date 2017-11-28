const path = require('path')
const http = require('http')
const bigint = require('bigint')
const Koa = require('koa')
const router = require('koa-route')
const websockify = require('koa-websocket')
const serve = require('koa-static')
const mysql = require('mysql2/promise')
const Game = require('./Game')

const app = websockify(new Koa())
const pool = mysql.createPool({
  connectionLimit: 20,
  host: process.env.ISU_DB_HOST || '127.0.0.1',
  port: process.env.ISU_DB_PORT || '3306',
  user: process.env.ISU_DB_USER || 'root',
  password: process.env.ISU_DB_PASSWORD || '',
  database: 'isudb',
  charset: 'utf8mb4',
})

const getInitializeHandler = async (ctx) => {
  await pool.query('TRUNCATE TABLE adding')
  await pool.query('TRUNCATE TABLE buying')
  await pool.query('TRUNCATE TABLE room_time')
  ctx.status = 204
}

const getRoomHandler = async (ctx, roomName) => {
  roomName = typeof roomName !== 'string' ? '' : roomName
  ctx.body = {
    host: '',
    path: `/ws/${roomName}`
  }
}

const wsGameHandler = async (ctx, roomName) => {
  roomName = typeof roomName !== 'string' ? '' : roomName

  ctx.websocket.on('message', async (message) => {
    try {
      const { request_id, action, time, isu, item_id, count_bought } = JSON.parse(message)
      let is_success = false
      switch (action) {
        case 'addIsu':
          is_success = await game.addIsu(bigint(isu), time)
          break;
        case 'buyItem':
          is_success = await game.buyItem(item_id, count_bought || 0, time)
          break;
        default:
          console.error('Invalid Action')
      }

      if (is_success) {
        // GameResponse を返却する前に 反映済みの GameStatus を返す
        await send(ctx.websocket, await game.getStatus())
      }

      await send(ctx.websocket, { request_id, is_success })
    } catch (e) {
      console.error(e)
      ctx.app.emit('error', e, ctx)
      ctx.throw(e)
    }
  })

  ctx.websocket.on('close', async () => {
    clearTimeout(tid)
  })

  const send = (ws, messageObj) => {
    if (ws.readyState === ws.constructor.OPEN) {
      return new Promise((resolve, reject) => {
        ws.send(JSON.stringify(messageObj), (e) => {
          e ? reject(e) : resolve()
        })
      })
    }
    console.log('Connection already closed')
    return Promise.resolve()
  }
  const loop = async () => {
    if (ctx.websocket.readyState === ctx.websocket.constructor.OPEN) {
      await send(ctx.websocket, await game.getStatus())
    }

    if (![ctx.websocket.constructor.CLOSING, ctx.websocket.constructor.CLOSED].includes(ctx.websocket.readyState)) {
      tid = setTimeout(loop, 500)
    }
  }
  const game = new Game(roomName, pool)
  let tid = setTimeout(loop, 500)

  await send(ctx.websocket, await game.getStatus())
}

app
  .use(serve(path.resolve(__dirname, '..', 'public')))
  .use(router.get('/initialize', getInitializeHandler))
  .use(router.get('/room', getRoomHandler))
  .use(router.get('/room/:room_name', getRoomHandler))

app.ws
  .use(router.all('/ws', wsGameHandler))
  .use(router.all('/ws/:room_name', wsGameHandler))

// const server = http.createServer(app.callback()).listen(5000)
app.listen(5000)
