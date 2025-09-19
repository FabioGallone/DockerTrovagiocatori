import requests
from typing import Dict, List
from datetime import datetime
import logging
from database.connection import SessionLocal
from database.models import ChatMessage
from chat.socketio_app import sio
from config.settings import settings

logger = logging.getLogger(__name__)

# Dizionario per tenere traccia degli utenti connessi
connected_users: Dict[str, Dict] = {}

async def get_user_info_from_auth_service(session_cookie: str) -> Dict:
    "Ottiene le informazioni dell'utente dall'auth service"
    try:
        response = requests.get(
            f"{settings.AUTH_SERVICE_URL}/api/user",
            cookies={"session_id": session_cookie},
            timeout=5
        )
        if response.status_code == 200:
            return response.json()
        else:
            logger.error(f"Auth service returned {response.status_code}")
            return None
    except Exception as e:
        logger.error(f"Errore nel recupero info utente: {e}")
        return None

def save_message_to_database(sender_email: str, recipient_email: str, content: str, chat_type: str = "friend") -> int:
    "Salva il messaggio nel database"
    db = SessionLocal()
    try:
        chat_message = ChatMessage(
            sender_email=sender_email,
            recipient_email=recipient_email,
            content=content,
            chat_type=chat_type
        )
        db.add(chat_message)
        db.commit()
        db.refresh(chat_message)
        
        logger.info(f"Messaggio {chat_type} salvato con ID {chat_message.id}")
        return chat_message.id
    except Exception as e:
        logger.error(f"Errore salvataggio messaggio: {e}")
        db.rollback()
        return None
    finally:
        db.close()

def get_chat_history(user_email1: str, user_email2: str, limit: int = 50) -> List[Dict]:
    "Recupera la cronologia chat tra due utenti"
    db = SessionLocal()
    try:
        messages = db.query(ChatMessage).filter(
            ((ChatMessage.sender_email == user_email1) & (ChatMessage.recipient_email == user_email2)) |
            ((ChatMessage.sender_email == user_email2) & (ChatMessage.recipient_email == user_email1))
        ).order_by(ChatMessage.created_at.asc()).limit(limit).all()
        
        result = []
        for msg in messages:
            result.append({
                'id': str(msg.id),
                'sender_email': msg.sender_email,
                'recipient_email': msg.recipient_email,
                'content': msg.content,
                'timestamp': msg.created_at.isoformat(),
                'read': msg.is_read
            })
        
        logger.info(f"Recuperati {len(result)} messaggi tra {user_email1} e {user_email2}")
        return result
    except Exception as e:
        logger.error(f"Errore recupero cronologia: {e}")
        return []
    finally:
        db.close()

@sio.event
async def connect(sid, environ, auth):
    "Gestisce la connessione di un nuovo utente"
    try:
        logger.info(f"Tentativo di connessione per sid: {sid}")
        
        session_cookie = None
        if auth and isinstance(auth, dict):
            session_cookie = auth.get('session_cookie')
        
        if not session_cookie:
            logger.warning(f"Connessione rifiutata per {sid}: manca session cookie")
            await sio.disconnect(sid)
            return False
        
        # Ottieni info utente dall'auth service
        user_info = await get_user_info_from_auth_service(session_cookie)
        if not user_info:
            logger.warning(f"Connessione rifiutata per {sid}: utente non autenticato")
            await sio.disconnect(sid)
            return False
        
        user_email = user_info['email']
        logger.info(f"Utente identificato: {user_email}")
        
        # Salva le informazioni nella sessione del socket
        await sio.save_session(sid, {
            'user_email': user_email,
            'session_cookie': session_cookie
        })
        
        # Aggiorna lo stato dell'utente connesso
        connected_users[user_email] = {
            'sid': sid,
            'online': True,
            'connected_at': datetime.utcnow().isoformat()
        }
        
        # Il socket si unisce a una room con il proprio user_email
        user_room = f"user_{user_email.replace('@', '_').replace('.', '_')}"
        await sio.enter_room(sid, user_room)
        
        logger.info(f"Utente {user_email} connesso con sid {sid}")
        
        # Emetti evento di connessione riuscita
        await sio.emit('connected', {
            'message': 'Connesso con successo',
            'user_email': user_email,
            'timestamp': datetime.utcnow().isoformat()
        }, room=sid)
        
        # Notifica agli altri utenti che questo utente è online
        await sio.emit('user_online', {
            'user_email': user_email,
            'timestamp': datetime.utcnow().isoformat()
        }, skip_sid=sid)
        
        return True
        
    except Exception as e:
        logger.error(f"Errore durante connessione {sid}: {e}")
        await sio.disconnect(sid)
        return False

@sio.event
async def disconnect(sid):
    "Gestisce la disconnessione di un utente"
    try:
        session = await sio.get_session(sid)
        if session:
            user_email = session.get('user_email')
            if user_email and user_email in connected_users:
                connected_users[user_email]['online'] = False
                connected_users[user_email]['disconnected_at'] = datetime.utcnow().isoformat()
                
                await sio.emit('user_offline', {
                    'user_email': user_email,
                    'timestamp': datetime.utcnow().isoformat()
                })
                
                logger.info(f"Utente {user_email} disconnesso")
                
    except Exception as e:
        logger.error(f"Errore durante disconnessione {sid}: {e}")

@sio.event
async def join_chat(sid, data):
    "Un utente vuole iniziare/partecipare a una chat con un altro utente"
    try:
        session = await sio.get_session(sid)
        if not session:
            await sio.emit('error', {'message': 'Sessione non valida'}, room=sid)
            return
        
        user_email = session['user_email']
        other_user_email = data['other_user_email']
        
        logger.info(f"Utente {user_email} vuole entrare nella chat con {other_user_email}")
        
        # Crea il nome della room privata
        sorted_emails = sorted([user_email, other_user_email])
        room_name = f"chat_{sorted_emails[0].replace('@', '_').replace('.', '_')}_{sorted_emails[1].replace('@', '_').replace('.', '_')}"
        
        # Aggiungi il socket alla room
        await sio.enter_room(sid, room_name)
        
        # Invia la cronologia messaggi esistenti
        chat_history = get_chat_history(user_email, other_user_email)
        
        if chat_history:
            await sio.emit('chat_history', {
                'messages': chat_history
            }, room=sid)
        
        logger.info(f"Utente {user_email} entrato nella room {room_name}")
        
    except Exception as e:
        logger.error(f"Errore join_chat per {sid}: {e}")
        await sio.emit('error', {'message': f'Errore nell\'entrare in chat: {str(e)}'}, room=sid)

@sio.event
async def send_private_message(sid, data):
    "Invia un messaggio privato"
    try:
        session = await sio.get_session(sid)
        if not session:
            await sio.emit('error', {'message': 'Sessione non valida'}, room=sid)
            return
        
        user_email = session['user_email']
        recipient_email = data['recipient_email']
        message_content = data['message'].strip() #rimuove spazi bianchi all’inizio e alla fine della stringa
        
        if not message_content:
            await sio.emit('error', {'message': 'Messaggio vuoto'}, room=sid)
            return
        
        logger.info(f"Messaggio da {user_email} a {recipient_email}: {message_content[:50]}...")
        
        # Salva il messaggio nel database
        message_id = save_message_to_database(user_email, recipient_email, message_content, "friend")
        if not message_id:
            await sio.emit('error', {'message': 'Errore nel salvataggio del messaggio'}, room=sid)
            return
        
        # Crea il nome della room
        sorted_emails = sorted([user_email, recipient_email])
        room_name = f"chat_{sorted_emails[0].replace('@', '_').replace('.', '_')}_{sorted_emails[1].replace('@', '_').replace('.', '_')}"
        
        # Crea il messaggio completo
        message = {
            'id': str(message_id),
            'sender_email': user_email,
            'recipient_email': recipient_email,
            'content': message_content,
            'timestamp': datetime.utcnow().isoformat(),
            'read': False
        }
        
        # Invia il messaggio a tutta la room
        await sio.emit('new_private_message', message, room=room_name)
        
        logger.info(f"Messaggio inviato a room {room_name}")
        
        # Conferma di invio al mittente
        await sio.emit('message_sent', {
            'message_id': message['id'],
            'status': 'sent',
            'timestamp': message['timestamp']
        }, room=sid)
        
    except Exception as e:
        logger.error(f"Errore send_private_message per {sid}: {e}")
        await sio.emit('error', {'message': f'Errore nell\'invio messaggio: {str(e)}'}, room=sid)

@sio.event
async def typing_start(sid, data):
    "Notifica che l'utente sta scrivendo"
    try:
        session = await sio.get_session(sid)
        if not session:
            return
        
        user_email = session['user_email']
        recipient_email = data['recipient_email']
        
        # Crea il nome della room
        sorted_emails = sorted([user_email, recipient_email])
        room_name = f"chat_{sorted_emails[0].replace('@', '_').replace('.', '_')}_{sorted_emails[1].replace('@', '_').replace('.', '_')}"
        
        # Invia notifica di typing (escludendo il mittente)
        await sio.emit('user_typing', {
            'user_email': user_email,
            'typing': True
        }, room=room_name, skip_sid=sid)
        
    except Exception as e:
        logger.error(f"Errore typing_start per {sid}: {e}")

@sio.event
async def typing_stop(sid, data):
    "Notifica che l'utente ha smesso di scrivere"
    try:
        session = await sio.get_session(sid)
        if not session:
            return
        
        user_email = session['user_email']
        recipient_email = data['recipient_email']
        
        # Crea il nome della room
        sorted_emails = sorted([user_email, recipient_email])
        room_name = f"chat_{sorted_emails[0].replace('@', '_').replace('.', '_')}_{sorted_emails[1].replace('@', '_').replace('.', '_')}"
        
        # Invia notifica di stop typing (escludendo il mittente)
        await sio.emit('user_typing', {
            'user_email': user_email,
            'typing': False
        }, room=room_name, skip_sid=sid)
        
    except Exception as e:
        logger.error(f"Errore typing_stop per {sid}: {e}")

@sio.event
async def get_online_users(sid):
    "Restituisce la lista degli utenti online"
    try:
        online_users = [
            {
                'email': email,
                'online': data['online'],
                'connected_at': data.get('connected_at')
            }
            for email, data in connected_users.items()
            if data['online']
        ]
        
        await sio.emit('online_users', {
            'users': online_users,
            'count': len(online_users)
        }, room=sid)
        
        logger.info(f"Inviata lista utenti online: {len(online_users)} utenti")
        
    except Exception as e:
        logger.error(f"Errore get_online_users per {sid}: {e}")

@sio.event
async def ping(sid):
    "Risponde al ping del client"
    try:
        await sio.emit('pong', {'timestamp': datetime.utcnow().isoformat()}, room=sid)
    except Exception as e:
        logger.error(f"Errore ping per {sid}: {e}")

logger.info("Socket.IO handler configurato correttamente")