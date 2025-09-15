from fastapi import FastAPI, Depends
import socketio
from database.connection import engine, get_db, Base
from database.models import *
from api.routes import posts, comments, fields, admin
from chat.socketio_app import sio
from services.field import load_sport_fields
import logging

# Configurazione logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Crea l'app FastAPI
app = FastAPI(
    title="TrovaGiocatori API", 
    description="API per la gestione di eventi sportivi e campi"
)

# Creazione delle tabelle nel database
Base.metadata.create_all(bind=engine)

# Includi i router delle API
app.include_router(fields.router)
app.include_router(posts.router)
app.include_router(comments.router)
app.include_router(admin.router)

@app.on_event("startup")
async def startup_event():
    """Inizializzazione dell'applicazione"""
    # Carica i campi sportivi
    db = next(get_db())
    try:
        load_sport_fields(db)
        logger.info("Campi sportivi caricati con successo")
    except Exception as e:
        logger.error(f"Errore nel caricamento dei campi sportivi: {e}")
    finally:
        db.close()
    
    logger.info("Socket.IO server integrato correttamente")

@app.get("/")
def root():
    """Endpoint di base per verificare che l'API sia attiva"""
    return {"message": "TrovaGiocatori API è attiva!", "status": "OK"}

@app.get("/ws/test")
async def test_socket_endpoint():
    """Endpoint di test per verificare che Socket.IO sia configurato"""
    return {
        "message": "Socket.IO is running",
        "socketio_path": "/ws/socket.io/",
        "status": "OK"
    }

# Monta Socket.IO come applicazione ASGI
sio_asgi_app = socketio.ASGIApp(sio, other_asgi_app=app, socketio_path="/ws/socket.io")

# Esporta l'app ASGI per uvicorn
app = sio_asgi_app