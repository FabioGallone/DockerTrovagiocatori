from fastapi import APIRouter, Depends, HTTPException
from starlette.requests import Request
from sqlalchemy.orm import Session
from typing import List
from database.connection import get_db
from database.models import Post, Comment
from schemas.comment import CommentCreate, CommentResponse
from api.dependencies import get_current_user_email
from config.settings import settings
import requests
import logging

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/posts", tags=["Commenti"])

@router.post("/{post_id}/comments/", response_model=CommentResponse)
def create_comment(post_id: int, comment: CommentCreate, request: Request, db: Session = Depends(get_db)):
    "Aggiungi un commento a un post"
    # Verifica che il post esista
    post = db.query(Post).filter(Post.id == post_id).first()
    if not post:
        raise HTTPException(status_code=404, detail="Post non trovato")
    
    # Ottieni l'email dell'utente autenticato
    user_email = get_current_user_email(request)
    
    # Crea il nuovo commento
    new_comment = Comment(
        post_id=post_id,
        autore_email=user_email,
        contenuto=comment.contenuto
    )
    
    db.add(new_comment)
    db.commit()
    db.refresh(new_comment)
    
    logger.info(f"Commento creato da {user_email} per post {post_id}")
    return new_comment


@router.get("/{post_id}/comments/", response_model=List[CommentResponse])
def get_post_comments(post_id: int, db: Session = Depends(get_db)):
    "Ottieni tutti i commenti di un post con info utente"
    # Verifica che il post esista
    post = db.query(Post).filter(Post.id == post_id).first()
    if not post:
        raise HTTPException(status_code=404, detail="Post non trovato")
    
    # Ottieni tutti i commenti del post ordinati per data di creazione
    comments = db.query(Comment).filter(
        Comment.post_id == post_id
    ).order_by(Comment.created_at.desc()).all()
    
    # Arricchisci ogni commento con le informazioni dell'utente
    enriched_comments = []
    for comment in comments:
        try:
            # Chiama l'auth-service per ottenere le info dell'utente
            encoded_email = requests.utils.quote(comment.autore_email)
            user_response = requests.get(
                f"{settings.AUTH_SERVICE_URL}/api/user/by-email?email={encoded_email}",
                timeout=5
            )
            
            if user_response.status_code == 200:
                user_data = user_response.json()
                enriched_comment = CommentResponse(
                    id=comment.id,
                    post_id=comment.post_id,
                    autore_email=comment.autore_email,
                    contenuto=comment.contenuto,
                    created_at=comment.created_at,
                    autore_username=user_data.get("username", "Utente sconosciuto"),
                    autore_nome=user_data.get("nome", ""),
                    autore_cognome=user_data.get("cognome", "")
                )
                enriched_comments.append(enriched_comment)
            else:
                # Fallback se non riesco a ottenere i dati utente
                enriched_comment = CommentResponse(
                    id=comment.id,
                    post_id=comment.post_id,
                    autore_email=comment.autore_email,
                    contenuto=comment.contenuto,
                    created_at=comment.created_at,
                    autore_username="Utente sconosciuto",
                    autore_nome="",
                    autore_cognome=""
                )
                enriched_comments.append(enriched_comment)
                
        except Exception as e:
            logger.error(f"Errore nel recupero dati utente per {comment.autore_email}: {e}")
            # Fallback con solo email
            enriched_comment = CommentResponse(
                id=comment.id,
                post_id=comment.post_id,
                autore_email=comment.autore_email,
                contenuto=comment.contenuto,
                created_at=comment.created_at,
                autore_username=comment.autore_email,
                autore_nome="",
                autore_cognome=""
            )
            enriched_comments.append(enriched_comment)
    
    return enriched_comments