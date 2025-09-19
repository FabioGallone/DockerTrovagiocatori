import requests
from config.settings import settings
import logging

logger = logging.getLogger(__name__)

def get_participants_count(post_id: int) -> int:
    "Ottiene il numero di partecipanti per un evento"
    try:
        response = requests.get(
            f"{settings.AUTH_SERVICE_URL}/events/{post_id}/participants", 
            timeout=3
        )
        if response.status_code == 200:
            data = response.json()
            return data.get("count", 0)
        return 0
    except Exception as e:
        logger.error(f"Errore nel recupero partecipanti per post {post_id}: {e}")
        return 0

def enrich_post_with_participants(post) -> dict:
    "Arricchisce un post con informazioni sui partecipanti"
    participants_count = get_participants_count(post.id)
    posti_disponibili = max(0, post.numero_giocatori - participants_count)
    
    return {
        "id": post.id,
        "titolo": post.titolo,
        "provincia": post.provincia,
        "citta": post.citta,
        "sport": post.sport,
        "data_partita": post.data_partita.strftime("%Y-%m-%d"),
        "ora_partita": post.ora_partita.strftime("%H:%M"),
        "commento": post.commento,
        "autore_email": post.autore_email,
        "campo_id": post.campo_id,
        "campo": {
            "nome": post.campo.nome,
            "indirizzo": post.campo.indirizzo
        } if post.campo else None,
        "livello": post.livello,
        "numero_giocatori": post.numero_giocatori,
        "partecipanti_iscritti": participants_count,
        "posti_disponibili": posti_disponibili,
        "is_full": posti_disponibili == 0
    }