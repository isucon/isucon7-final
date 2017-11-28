#!/usr/bin/env python3

import asyncio
from pathlib import Path
import urllib.parse

from aiohttp import web

import game


public_dir = (Path(__file__) / '../../public').resolve()


async def initialize_handler(request):
    game.initialize()
    return web.HTTPNoContent()


async def index_handler(request):
    return web.FileResponse(public_dir / 'index.html')


async def room_handler(request):
    room_name = request.match_info.get('room_name', '')
    res = {"host": "", "path": "/ws/" + urllib.parse.quote(room_name)}
    return web.json_response(res)


async def game_handler(request):
    room_name = request.match_info.get("room_name", "")
    ws = web.WebSocketResponse()
    await ws.prepare(request)
    await game.serve(ws, room_name)
    return ws


def main():
    app = web.Application()
    app.router.add_get("/initialize", initialize_handler)
    app.router.add_get("/room/{room_name}", room_handler)
    app.router.add_get("/room/", room_handler)
    app.router.add_get("/ws/{room_name}", game_handler)
    app.router.add_get("/ws/", game_handler)
    app.router.add_get('/', index_handler)
    app.router.add_static('/', path=public_dir, name="static")
    web.run_app(app, host='0.0.0.0', port=5000)


if __name__ == '__main__':
    main()
