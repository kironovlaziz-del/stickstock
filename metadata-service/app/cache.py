import json
from typing import Any
import redis.asyncio as redis
from app.config import settings

class Cache:
    def __init__(self):
        self.client = None

    async def connect(self):
        if not self.client:
            self.client = redis.from_url(settings.redis_url, decode_responses=True)

    async def get(self, key: str):
        await self.connect()
        data = await self.client.get(key)
        return json.loads(data) if data else None

    async def set(self, key: str, value: Any, ttl: int):
        await self.connect()
        await self.client.setex(key, ttl, json.dumps(value, default=str))

    async def close(self):
        if self.client:
            await self.client.close()
            self.client = None

cache = Cache()
