import socketio
from config.settings import settings

# Crea il server Socket.IO
sio = socketio.AsyncServer(
    async_mode="asgi",
    cors_allowed_origins=settings.CORS_ORIGINS,
    logger=True,
    engineio_logger=True,
    path=settings.SOCKETIO_PATH
)

# Importa i gestori di eventi
from chat.handlers import *