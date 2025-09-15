from fastapi import HTTPException, Depends
from starlette.requests import Request
import requests
from config.settings import settings
import logging

logger = logging.getLogger(__name__)

def get_current_user_email(request: Request) -> str:
    """Ottieni l'email dell'utente autenticato dall'auth service"""
    session_cookie = request.cookies.get("session_id")
    
    if not session_cookie:
        raise HTTPException(status_code=401, detail="Session cookie not found")

    try:
        auth_url = f"{settings.AUTH_SERVICE_URL}/api/user"
        response = requests.get(auth_url, cookies={"session_id": session_cookie}, timeout=10)
        
        if response.status_code != 200:
            raise HTTPException(status_code=401, detail=f"Invalid session - Auth service returned {response.status_code}")
        
        user_data = response.json()
        return user_data["email"]
        
    except requests.exceptions.RequestException as e:
        logger.error(f"Errore connessione auth service: {e}")
        raise HTTPException(status_code=500, detail="Auth service unavailable")

def verify_admin_user(request: Request) -> str:
    """Verifica che l'utente sia un amministratore"""
    try:
        user_email = get_current_user_email(request)
        
        session_cookie = request.cookies.get("session_id")
        auth_response = requests.get(
            f"{settings.AUTH_SERVICE_URL}/profile",
            cookies={"session_id": session_cookie},
            timeout=5
        )
        
        if auth_response.status_code != 200:
            raise HTTPException(status_code=403, detail="Non autorizzato")
        
        user_data = auth_response.json()
        if not user_data.get("is_admin", False):
            raise HTTPException(status_code=403, detail="Privilegi amministratore richiesti")
        
        return user_email
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Errore verifica admin: {e}")
        raise HTTPException(status_code=500, detail="Errore verifica privilegi admin")