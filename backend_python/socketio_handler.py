"""
Socket.IO handler per la chat live tra utenti
Gestisce le connessioni, i messaggi privati e le room di chat con persistenza nel database
"""
import socketio
import requests
import json
from typing import Dict, List
from datetime import datetime
import logging
from database import SessionLocal
from models import ChatMessage

# Configurazione logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# CORREZIONE: Crea il server Socket.IO con configurazione corretta
sio = socketio.AsyncServer(
    async_mode="asgi",
    cors_allowed_origins="*",
    logger=True,
    engineio_logger=True,  # Abilita temporaneamente per debug
    path="/ws/socket.io"  # AGGIUNGI QUESTA LINEA
)

# Dizionario per tenere traccia degli utenti connessi
# Struttura: {user_email: {sid: socket_id, username: username, online: bool}}
connected_users: Dict[str, Dict] = {}

# Dizionario per tenere traccia delle chat attive per ogni post
# Struttura: {post_id: {participants: [email1, email2], room_name: string}}
active_chats: Dict[int, Dict] = {}


async def get_user_info_from_auth_service(session_cookie: str) -> Dict:
    """Ottiene le informazioni dell'utente dall'auth service usando il session cookie"""
    try:
        # CORREZIONE: Usa il nome del container interno Docker
        response = requests.get(
            "http://auth-service:8080/api/user",
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


def create_private_room_name(email1: str, email2: str, post_id: int) -> str:
    """Crea un nome univoco per la room privata tra due utenti"""
    # Ordina le email per garantire sempre lo stesso nome room indipendentemente dall'ordine
    sorted_emails = sorted([email1, email2])
    
    # Gestisce sia chat legate a post che chat generiche
    if post_id == -1:
        # Chat generica tra amici
        return f"friend_chat_{sorted_emails[0].replace('@', '_').replace('.', '_')}_{sorted_emails[1].replace('@', '_').replace('.', '_')}"
    else:
        # Chat legata a un post
        return f"post_{post_id}_chat_{sorted_emails[0].replace('@', '_').replace('.', '_')}_{sorted_emails[1].replace('@', '_').replace('.', '_')}"


def save_message_to_database(post_id: int, sender_email: str, recipient_email: str, content: str) -> int:
    """Salva il messaggio nel database e restituisce l'ID"""
    db = SessionLocal()
    try:
        chat_message = ChatMessage(
            post_id=post_id,
            sender_email=sender_email,
            recipient_email=recipient_email,
            content=content
        )
        db.add(chat_message)
        db.commit()
        db.refresh(chat_message)
        
        logger.info(f"[CHAT DB] Messaggio salvato con ID {chat_message.id}")
        return chat_message.id
    except Exception as e:
        logger.error(f"[CHAT DB] Errore salvataggio messaggio: {e}")
        db.rollback()
        return None
    finally:
        db.close()


def get_chat_history(post_id: int, user_email1: str, user_email2: str, limit: int = 50) -> List[Dict]:
    """Recupera la cronologia chat tra due utenti per un post"""
    db = SessionLocal()
    try:
        messages = db.query(ChatMessage).filter(
            ChatMessage.post_id == post_id,
            ((ChatMessage.sender_email == user_email1) & (ChatMessage.recipient_email == user_email2)) |
            ((ChatMessage.sender_email == user_email2) & (ChatMessage.recipient_email == user_email1))
        ).order_by(ChatMessage.created_at.asc()).limit(limit).all()
        
        result = []
        for msg in messages:
            result.append({
                'id': str(msg.id),
                'post_id': msg.post_id,
                'sender_email': msg.sender_email,
                'recipient_email': msg.recipient_email,
                'content': msg.content,
                'timestamp': msg.created_at.isoformat(),
                'read': msg.is_read
            })
        
        logger.info(f"[CHAT DB] Recuperati {len(result)} messaggi per post {post_id}")
        return result
    except Exception as e:
        logger.error(f"[CHAT DB] Errore recupero cronologia: {e}")
        return []
    finally:
        db.close()


@sio.event
async def connect(sid, environ, auth):
    """Gestisce la connessione di un nuovo utente"""
    try:
        logger.info(f"[SOCKET] Tentativo di connessione per sid: {sid}")
        logger.info(f"[SOCKET] Environ: {environ.get('PATH_INFO')}")
        logger.info(f"[SOCKET] Auth: {auth}")
        
        # Verifica autenticazione
        session_cookie = None
        if auth and isinstance(auth, dict):
            session_cookie = auth.get('session_cookie')
        
        if not session_cookie:
            logger.warning(f"[SOCKET] Connessione rifiutata per {sid}: manca session cookie")
            await sio.disconnect(sid)
            return False
        
        logger.info(f"[SOCKET] Session cookie ricevuto: {session_cookie[:20]}...")
        
        # Ottieni info utente dall'auth service
        user_info = await get_user_info_from_auth_service(session_cookie)
        if not user_info:
            logger.warning(f"[SOCKET] Connessione rifiutata per {sid}: utente non autenticato")
            await sio.disconnect(sid)
            return False
        
        user_email = user_info['email']
        logger.info(f"[SOCKET] Utente identificato: {user_email}")
        
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
        
        # Il socket si unisce automaticamente a una room con il proprio user_email
        await sio.enter_room(sid, f"user_{user_email.replace('@', '_').replace('.', '_')}")
        
        logger.info(f"[SOCKET] ✅ Utente {user_email} connesso con sid {sid}")
        
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
        logger.error(f"[SOCKET] Errore durante connessione {sid}: {e}")
        await sio.disconnect(sid)
        return False


@sio.event
async def disconnect(sid):
    """Gestisce la disconnessione di un utente"""
    try:
        session = await sio.get_session(sid)
        if session:
            user_email = session.get('user_email')
            if user_email and user_email in connected_users:
                # Aggiorna lo stato offline
                connected_users[user_email]['online'] = False
                connected_users[user_email]['disconnected_at'] = datetime.utcnow().isoformat()
                
                # Notifica agli altri che l'utente è offline
                await sio.emit('user_offline', {
                    'user_email': user_email,
                    'timestamp': datetime.utcnow().isoformat()
                })
                
                logger.info(f"[SOCKET] Utente {user_email} disconnesso")
                
                # Cleanup: rimuovi dopo un po' di tempo
                # Per ora lo lasciamo per tracking
                
    except Exception as e:
        logger.error(f"[SOCKET] Errore durante disconnessione {sid}: {e}")


@sio.event
async def join_post_chat(sid, data):
    """
    Un utente vuole iniziare/partecipare alla chat di un post
    data = {
        'post_id': int,
        'post_author_email': str  # Email dell'autore del post
    }
    """
    try:
        session = await sio.get_session(sid)
        if not session:
            await sio.emit('error', {'message': 'Sessione non valida'}, room=sid)
            return
        
        user_email = session['user_email']
        post_id = data['post_id']
        post_author_email = data['post_author_email']
        
        logger.info(f"[SOCKET] Utente {user_email} vuole entrare nella chat del post {post_id}")
        
        # Crea il nome della room privata
        room_name = create_private_room_name(user_email, post_author_email, post_id)
        
        # Aggiungi il socket alla room
        await sio.enter_room(sid, room_name)
        
        # Aggiorna il tracking delle chat attive
        if post_id not in active_chats:
            active_chats[post_id] = {
                'participants': list(set([user_email, post_author_email])),  # Evita duplicati
                'room_name': room_name,
                'created_at': datetime.utcnow().isoformat()
            }
        else:
            # Aggiungi il partecipante se non è già presente
            if user_email not in active_chats[post_id]['participants']:
                active_chats[post_id]['participants'].append(user_email)
        
        # NUOVO: Invia la cronologia messaggi esistenti
        other_user = post_author_email if user_email != post_author_email else user_email
        chat_history = get_chat_history(post_id, user_email, other_user)
        
        if chat_history:
            await sio.emit('chat_history', {
                'post_id': post_id,
                'messages': chat_history
            }, room=sid)
        
        # Notifica che l'utente è entrato nella chat
        await sio.emit('chat_joined', {
            'post_id': post_id,
            'room_name': room_name,
            'participants': active_chats[post_id]['participants'],
            'user_email': user_email,
            'message': f'{user_email} è entrato nella chat'
        }, room=room_name)
        
        logger.info(f"[SOCKET] ✅ Utente {user_email} entrato nella room {room_name}")
        
    except Exception as e:
        logger.error(f"[SOCKET] Errore join_post_chat per {sid}: {e}")
        await sio.emit('error', {'message': f'Errore nell\'entrare in chat: {str(e)}'}, room=sid)


@sio.event
async def send_private_message(sid, data):
    """
    Invia un messaggio privato nella chat del post
    data = {
        'post_id': int,
        'recipient_email': str,
        'message': str
    }
    """
    try:
        session = await sio.get_session(sid)
        if not session:
            await sio.emit('error', {'message': 'Sessione non valida'}, room=sid)
            return
        
        user_email = session['user_email']
        post_id = data['post_id']
        recipient_email = data['recipient_email']
        message_content = data['message'].strip()
        
        if not message_content:
            await sio.emit('error', {'message': 'Messaggio vuoto'}, room=sid)
            return
        
        logger.info(f"[SOCKET] Messaggio da {user_email} a {recipient_email} per post {post_id}: {message_content[:50]}...")
        
        # NUOVO: Salva il messaggio nel database
        message_id = save_message_to_database(post_id, user_email, recipient_email, message_content)
        if not message_id:
            await sio.emit('error', {'message': 'Errore nel salvataggio del messaggio'}, room=sid)
            return
        
        # Crea il nome della room
        room_name = create_private_room_name(user_email, recipient_email, post_id)
        
        # Crea il messaggio
        message = {
            'id': str(message_id),  # Usa l'ID dal database
            'post_id': post_id,
            'sender_email': user_email,
            'recipient_email': recipient_email,
            'content': message_content,
            'timestamp': datetime.utcnow().isoformat(),
            'read': False
        }
        
        # FIX: Invia il messaggio SOLO al destinatario, NON al mittente
        recipient_room = f"user_{recipient_email.replace('@', '_').replace('.', '_')}"
        await sio.emit('new_private_message', message, room=recipient_room)
        
        # Se il destinatario è online, invia anche una notifica diretta
        if recipient_email in connected_users and connected_users[recipient_email]['online']:
            await sio.emit('message_notification', {
                'from': user_email,
                'post_id': post_id,
                'preview': message_content[:50] + ('...' if len(message_content) > 50 else ''),
                'timestamp': message['timestamp']
            }, room=recipient_room)
        
        logger.info(f"[SOCKET] ✅ Messaggio inviato da {user_email} a {recipient_email}")
        
        # Conferma di invio al mittente (senza il contenuto del messaggio per evitare duplicati)
        await sio.emit('message_sent', {
            'message_id': message['id'],
            'status': 'sent',
            'timestamp': message['timestamp']
        }, room=sid)
        
    except Exception as e:
        logger.error(f"[SOCKET] Errore send_private_message per {sid}: {e}")
        await sio.emit('error', {'message': f'Errore nell\'invio messaggio: {str(e)}'}, room=sid)


@sio.event
async def typing_start(sid, data):
    """Notifica che l'utente sta scrivendo"""
    try:
        session = await sio.get_session(sid)
        if not session:
            return
        
        user_email = session['user_email']
        post_id = data['post_id']
        recipient_email = data['recipient_email']
        
        room_name = create_private_room_name(user_email, recipient_email, post_id)
        
        # Invia notifica di typing (escludendo il mittente)
        await sio.emit('user_typing', {
            'user_email': user_email,
            'post_id': post_id,
            'typing': True
        }, room=room_name, skip_sid=sid)
        
        logger.debug(f"[SOCKET] {user_email} ha iniziato a scrivere in post {post_id}")
        
    except Exception as e:
        logger.error(f"[SOCKET] Errore typing_start per {sid}: {e}")


@sio.event
async def typing_stop(sid, data):
    """Notifica che l'utente ha smesso di scrivere"""
    try:
        session = await sio.get_session(sid)
        if not session:
            return
        
        user_email = session['user_email']
        post_id = data['post_id']
        recipient_email = data['recipient_email']
        
        room_name = create_private_room_name(user_email, recipient_email, post_id)
        
        # Invia notifica di stop typing (escludendo il mittente)
        await sio.emit('user_typing', {
            'user_email': user_email,
            'post_id': post_id,
            'typing': False
        }, room=room_name, skip_sid=sid)
        
        logger.debug(f"[SOCKET] {user_email} ha smesso di scrivere in post {post_id}")
        
    except Exception as e:
        logger.error(f"[SOCKET] Errore typing_stop per {sid}: {e}")


@sio.event
async def get_online_users(sid):
    """Restituisce la lista degli utenti online"""
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
        
        logger.info(f"[SOCKET] Inviata lista utenti online: {len(online_users)} utenti")
        
    except Exception as e:
        logger.error(f"[SOCKET] Errore get_online_users per {sid}: {e}")


@sio.event
async def leave_post_chat(sid, data):
    """L'utente esce dalla chat del post"""
    try:
        session = await sio.get_session(sid)
        if not session:
            return
        
        user_email = session['user_email']
        post_id = data['post_id']
        recipient_email = data['recipient_email']
        
        room_name = create_private_room_name(user_email, recipient_email, post_id)
        
        # Esci dalla room
        await sio.leave_room(sid, room_name)
        
        # Notifica l'uscita
        await sio.emit('user_left_chat', {
            'user_email': user_email,
            'post_id': post_id,
            'timestamp': datetime.utcnow().isoformat(),
            'message': f'{user_email} ha lasciato la chat'
        }, room=room_name)
        
        logger.info(f"[SOCKET] Utente {user_email} è uscito dalla chat del post {post_id}")
        
    except Exception as e:
        logger.error(f"[SOCKET] Errore leave_post_chat per {sid}: {e}")


@sio.event
async def ping(sid):
    """Risponde al ping del client per mantenere la connessione attiva"""
    try:
        await sio.emit('pong', {'timestamp': datetime.utcnow().isoformat()}, room=sid)
    except Exception as e:
        logger.error(f"[SOCKET] Errore ping per {sid}: {e}")


# Event handler per gestire errori generici
@sio.event
async def connect_error(sid, data):
    """Gestisce errori di connessione"""
    logger.error(f"[SOCKET] Errore di connessione per {sid}: {data}")


# Funzione utility per ottenere statistiche
async def get_chat_stats():
    """Restituisce statistiche sulle chat attive"""
    try:
        online_count = len([u for u in connected_users.values() if u['online']])
        
        return {
            'connected_users': online_count,
            'total_users_ever_connected': len(connected_users),
            'active_chats': len(active_chats),
            'chat_details': {
                post_id: {
                    'participants_count': len(chat_info['participants']),
                    'room_name': chat_info['room_name'],
                    'created_at': chat_info['created_at']
                }
                for post_id, chat_info in active_chats.items()
            }
        }
    except Exception as e:
        logger.error(f"[SOCKET] Errore nel calcolo statistiche: {e}")
        return {
            'error': str(e),
            'connected_users': 0,
            'total_users_ever_connected': 0,
            'active_chats': 0
        }


# Funzione di cleanup periodica 
async def cleanup_inactive_users():
    """Pulisce gli utenti inattivi dalle strutture dati"""
    try:
        current_time = datetime.utcnow()
        inactive_threshold = 3600  # 1 ora in secondi
        
        users_to_remove = []
        for email, user_data in connected_users.items():
            if not user_data['online'] and 'disconnected_at' in user_data:
                disconnect_time = datetime.fromisoformat(user_data['disconnected_at'])
                if (current_time - disconnect_time).total_seconds() > inactive_threshold:
                    users_to_remove.append(email)
        
        for email in users_to_remove:
            del connected_users[email]
            
        if users_to_remove:
            logger.info(f"[SOCKET] Cleanup: rimossi {len(users_to_remove)} utenti inattivi")
            
    except Exception as e:
        logger.error(f"[SOCKET] Errore durante cleanup: {e}")


logger.info("[SOCKET] ✅ Socket.IO handler configurato correttamente con persistenza messaggi")