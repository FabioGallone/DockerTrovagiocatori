#!/usr/bin/env python3
"""
Script di test per verificare il funzionamento della chat Socket.IO
Simula due utenti che chattano su un post
"""

import asyncio
import socketio
import json
from datetime import datetime

# Configurazione
SERVER_URL = "http://localhost:8000"
SOCKET_PATH = "/ws/socket.io/"

# Dati di test
USER1_SESSION = "test_session_user1"
USER2_SESSION = "test_session_user2"
TEST_POST_ID = 1
POST_AUTHOR_EMAIL = "organizzatore@test.com"

async def test_chat_functionality():
    """Test completo della funzionalità chat"""
    
    print("🚀 Avvio test chat Socket.IO...")
    
    # Crea due client Socket.IO
    client1 = socketio.AsyncClient()
    client2 = socketio.AsyncClient()
    
    messages_received = []
    
    # Event handlers per client1 (organizzatore)
    @client1.event
    async def connect():
        print("✅ Client1 (Organizzatore) connesso")
    
    @client1.event 
    async def new_private_message(data):
        print(f"📨 Client1 ricevuto messaggio: {data['content']} da {data['sender_email']}")
        messages_received.append(('client1', data))
    
    @client1.event
    async def user_typing(data):
        print(f"✍️  Client1: {data['user_email']} sta scrivendo...")
    
    # Event handlers per client2 (partecipante)
    @client2.event
    async def connect():
        print("✅ Client2 (Partecipante) connesso")
    
    @client2.event
    async def new_private_message(data):
        print(f"📨 Client2 ricevuto messaggio: {data['content']} da {data['sender_email']}")
        messages_received.append(('client2', data))
        
    @client2.event
    async def user_typing(data):
        print(f"✍️  Client2: {data['user_email']} sta scrivendo...")
    
    try:
        # Connetti i client
        await client1.connect(
            f"{SERVER_URL}",
            socketio_path=SOCKET_PATH,
            auth={'session_cookie': USER1_SESSION}
        )
        
        await client2.connect(
            f"{SERVER_URL}",
            socketio_path=SOCKET_PATH, 
            auth={'session_cookie': USER2_SESSION}
        )
        
        await asyncio.sleep(1)
        
        # Test 1: Join nella chat del post
        print("\n📋 Test 1: Join nella chat del post")
        
        await client1.emit('join_post_chat', {
            'post_id': TEST_POST_ID,
            'post_author_email': POST_AUTHOR_EMAIL
        })
        
        await client2.emit('join_post_chat', {
            'post_id': TEST_POST_ID,
            'post_author_email': POST_AUTHOR_EMAIL
        })
        
        await asyncio.sleep(1)
        
        # Test 2: Invio messaggi
        print("\n💬 Test 2: Invio messaggi")
        
        # Client1 invia messaggio
        await client1.emit('send_private_message', {
            'post_id': TEST_POST_ID,
            'recipient_email': 'partecipante@test.com',
            'message': 'Ciao! Sono l\'organizzatore dell\'evento 🏆'
        })
        
        await asyncio.sleep(0.5)
        
        # Client2 risponde
        await client2.emit('send_private_message', {
            'post_id': TEST_POST_ID,
            'recipient_email': POST_AUTHOR_EMAIL,
            'message': 'Ciao! Grazie per l\'invito, non vedo l\'ora! ⚽'
        })
        
        await asyncio.sleep(0.5)
        
        # Test 3: Typing indicators
        print("\n✍️  Test 3: Typing indicators")
        
        await client1.emit('typing_start', {
            'post_id': TEST_POST_ID,
            'recipient_email': 'partecipante@test.com'
        })
        
        await asyncio.sleep(1)
        
        await client1.emit('typing_stop', {
            'post_id': TEST_POST_ID,
            'recipient_email': 'partecipante@test.com'
        })
        
        # Test 4: Messaggio più lungo
        await client1.emit('send_private_message', {
            'post_id': TEST_POST_ID,
            'recipient_email': 'partecipante@test.com',
            'message': 'Perfetto! L\'evento è domani alle 18:30 al campo San Siro. Ci vediamo negli spogliatoi 15 minuti prima. Porta scarpe da calcio e una maglietta di riserva! 👕⚽'
        })
        
        await asyncio.sleep(1)
        
        # Test 5: Leave chat
        print("\n👋 Test 5: Leave chat")
        
        await client1.emit('leave_post_chat', {
            'post_id': TEST_POST_ID,
            'recipient_email': 'partecipante@test.com'
        })
        
        await asyncio.sleep(1)
        
        print(f"\n📊 Risultati test:")
        print(f"   Messaggi ricevuti: {len(messages_received)}")
        for client, msg in messages_received:
            timestamp = datetime.fromisoformat(msg['timestamp'].replace('Z', '+00:00'))
            print(f"   {client}: '{msg['content']}' ({timestamp.strftime('%H:%M:%S')})")
        
        print("\n✅ Test completato con successo!")
        
    except Exception as e:
        print(f"❌ Errore durante il test: {e}")
        import traceback
        traceback.print_exc()
    
    finally:
        # Disconnetti i client
        await client1.disconnect()
        await client2.disconnect()
        print("🔌 Client disconnessi")

if __name__ == "__main__":
    asyncio.run(test_chat_functionality())