#!/usr/bin/env python3
"""
Script di test SEMPLIFICATO per verificare il funzionamento della chat Socket.IO
"""

import asyncio
import socketio
import json
from datetime import datetime

# Configurazione
SERVER_URL = "http://localhost:8000"

async def test_simple_connection():
    """Test di connessione semplice"""
    
    print("🚀 Avvio test connessione Socket.IO...")
    
    # Crea un client Socket.IO
    client = socketio.AsyncClient()
    
    # Event handlers
    @client.event
    async def connect():
        print("✅ Client connesso con successo!")
    
    @client.event
    async def connected(data):
        print(f"📡 Ricevuto evento 'connected': {data}")
    
    @client.event
    async def disconnect():
        print("❌ Client disconnesso")
    
    @client.event
    async def connect_error(data):
        print(f"💥 Errore di connessione: {data}")
    
    @client.event
    async def error(data):
        print(f"❌ Errore: {data}")
    
    try:
        # Test connessione SENZA autenticazione (dovrebbe fallire)
        print("\n🧪 Test 1: Connessione senza autenticazione (dovrebbe fallire)")
        
        try:
            await client.connect(SERVER_URL, socketio_path="/ws/socket.io/")
            await asyncio.sleep(2)
            print("⚠️  Connessione riuscita senza auth (unexpected)")
        except Exception as e:
            print(f"✅ Connessione fallita come previsto: {e}")
        
        await client.disconnect()
        await asyncio.sleep(1)
        
        # Test connessione CON autenticazione fittizia
        print("\n🧪 Test 2: Connessione con auth fittizia")
        
        await client.connect(
            SERVER_URL,
            socketio_path="/ws/socket.io/",
            auth={'session_cookie': 'test_session_123'}
        )
        
        await asyncio.sleep(3)
        
        print("\n📊 Test completato!")
        
    except Exception as e:
        print(f"💥 Errore durante il test: {e}")
        import traceback
        traceback.print_exc()
    
    finally:
        try:
            await client.disconnect()
        except:
            pass
        print("🔌 Client disconnesso")


async def test_endpoint_availability():
    """Testa la disponibilità degli endpoint"""
    import aiohttp
    
    print("\n🌐 Test disponibilità endpoint...")
    
    endpoints_to_test = [
        f"{SERVER_URL}/",
        f"{SERVER_URL}/health",
        f"{SERVER_URL}/ws/test",
    ]
    
    async with aiohttp.ClientSession() as session:
        for endpoint in endpoints_to_test:
            try:
                async with session.get(endpoint, timeout=5) as response:
                    print(f"✅ {endpoint}: {response.status}")
                    if response.status == 200:
                        text = await response.text()
                        print(f"   Response: {text[:100]}...")
            except Exception as e:
                print(f"❌ {endpoint}: {e}")


if __name__ == "__main__":
    print("=" * 60)
    print("🧪 TEST SOCKET.IO - TROVAGIOCATORI")
    print("=" * 60)
    
    # Prima testa gli endpoint HTTP
    asyncio.run(test_endpoint_availability())
    
    # Poi testa Socket.IO
    asyncio.run(test_simple_connection())