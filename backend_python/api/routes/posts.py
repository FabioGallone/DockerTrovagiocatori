from fastapi import APIRouter, Depends, HTTPException
from starlette.requests import Request
from sqlalchemy.orm import Session
from typing import List
from database.connection import get_db
from database.models import Post, SportField
from schemas.post import PostCreate, PostResponse
from api.dependencies import get_current_user_email
from services.post import enrich_post_with_participants, get_participants_count
import requests
from config.settings import settings
import logging

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/posts", tags=["Post/Eventi"])

@router.post("/", response_model=PostResponse)
def create_post(post: PostCreate, request: Request, db: Session = Depends(get_db)):
    """Crea un nuovo post/evento sportivo"""
    user_email = get_current_user_email(request)
    logger.info(f"Creazione post da utente: {user_email}")

    # Verifica che il campo sportivo esista se specificato
    if post.campo_id:
        field = db.query(SportField).filter(SportField.id == post.campo_id).first()
        if not field:
            raise HTTPException(status_code=404, detail="Campo sportivo non trovato")
        
        # Verifica che il campo supporti lo sport selezionato
        if post.sport not in field.sports_disponibili:
            raise HTTPException(
                status_code=400, 
                detail=f"Il campo '{field.nome}' non supporta lo sport '{post.sport}'. Sport disponibili: {', '.join(field.sports_disponibili)}"
            )

    # Crea il nuovo post
    new_post = Post(
        titolo=post.titolo,
        provincia=post.provincia,
        citta=post.citta,
        sport=post.sport,
        data_partita=post.data_partita,
        ora_partita=post.ora_partita,
        commento=post.commento,
        autore_email=user_email,
        campo_id=post.campo_id,
        livello=post.livello,
        numero_giocatori=post.numero_giocatori
    )
    
    db.add(new_post)
    db.commit()
    db.refresh(new_post)
    
    logger.info(f"Post creato con ID: {new_post.id}, livello: {new_post.livello}, giocatori: {new_post.numero_giocatori}")
    return new_post

@router.get("/by-user")
def get_posts_by_user(request: Request, db: Session = Depends(get_db)):
    """Ottieni tutti i post creati dall'utente autenticato"""
    user_email = get_current_user_email(request)
    logger.info(f"Recupero post per utente: {user_email}")
    
    posts = db.query(Post).filter(Post.autore_email == user_email).order_by(Post.data_partita.desc()).all()
    
    if not posts:
        return []
    
    enriched_posts = []
    for post in posts:
        enriched_post = enrich_post_with_participants(post)
        enriched_posts.append(enriched_post)
    
    return enriched_posts

@router.get("/search")
def search_posts(provincia: str, sport: str, db: Session = Depends(get_db)):
    """Cerca post per provincia, sport e opzionalmente per livello"""
    query = db.query(Post).filter(
        Post.provincia == provincia, 
        Post.sport == sport
    )
   
    
    posts = query.all()
    
    if not posts:
        raise HTTPException(status_code=404, detail="Nessun post trovato per i criteri specificati")
    
    enriched_posts = []
    for post in posts:
        enriched_post = enrich_post_with_participants(post)
        enriched_posts.append(enriched_post)
    
    return enriched_posts

@router.get("/{post_id}", response_model=PostResponse)
def get_post(post_id: int, db: Session = Depends(get_db)):
    "Ottieni un post specifico"
    post = db.query(Post).filter(Post.id == post_id).first()
    if not post:
        raise HTTPException(status_code=404, detail="Post non trovato")
    return post

@router.get("/{post_id}/participants-count")
def get_post_participants_count(post_id: int):
    """Ottiene il numero di partecipanti iscritti a un evento"""
    try:
        response = requests.get(f"{settings.AUTH_SERVICE_URL}/events/{post_id}/participants", timeout=5)
        if response.status_code == 200:
            data = response.json()
            participants = data.get("participants", [])
            
            return {
                "success": True,
                "post_id": post_id,
                "count": len(participants), 
                "participants": participants
            }
        else:
            return {
                "success": False,
                "post_id": post_id,
                "count": 0,  
                "participants": []
            }
    except Exception as e:
        logger.error(f"Errore nel recupero partecipanti per post {post_id}: {e}")
        return {
            "success": False,
            "post_id": post_id,
            "count": 0,  
            "participants": []
        }

@router.get("/{post_id}/availability")
def get_post_availability(post_id: int, db: Session = Depends(get_db)):
    "Calcola i posti disponibili per un evento"
    post = db.query(Post).filter(Post.id == post_id).first()
    if not post:
        raise HTTPException(status_code=404, detail="Post non trovato")
    
    participants_count = get_participants_count(post_id)
    posti_disponibili = max(0, post.numero_giocatori - participants_count)
    
    return {
        "success": True,
        "post_id": post_id,
        "numero_giocatori_richiesti": post.numero_giocatori,
        "partecipanti_iscritti": participants_count,
        "posti_disponibili": posti_disponibili,
        "is_full": posti_disponibili == 0
    }

@router.get("/{post_id}/details")
def get_post_with_participants(post_id: int, db: Session = Depends(get_db)):
    "Ottieni un post specifico con informazioni sui partecipanti"
    post = db.query(Post).filter(Post.id == post_id).first()
    if not post:
        raise HTTPException(status_code=404, detail="Post non trovato")
    
    # Ottieni informazioni sui partecipanti
    try:
        response = requests.get(f"{settings.AUTH_SERVICE_URL}/events/{post_id}/participants", timeout=5)
        if response.status_code == 200:
            participants_data = response.json()
            participants = participants_data.get("participants", [])
            participants_count = participants_data.get("count", 0)
        else:
            participants = []
            participants_count = 0
    except Exception as e:
        logger.error(f"Errore nel recupero partecipanti: {e}")
        participants = []
        participants_count = 0
    
    return {
        "post": post,
        "participants": participants,
        "participants_count": participants_count,
        "posti_disponibili": max(0, post.numero_giocatori - participants_count),
        "is_full": participants_count >= post.numero_giocatori
    }