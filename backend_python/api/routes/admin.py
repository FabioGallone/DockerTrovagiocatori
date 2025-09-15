from fastapi import APIRouter, Depends, HTTPException
from starlette.requests import Request
from sqlalchemy.orm import Session
from datetime import datetime
from database.connection import get_db
from database.models import Post, Comment, SportField
from api.dependencies import verify_admin_user
from services.post import get_participants_count
from config.settings import settings
import requests
import logging

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/admin", tags=["Amministrazione"])

@router.delete("/posts/{post_id}")
def delete_post(post_id: int, request: Request, db: Session = Depends(get_db)):
    """Elimina un post (solo amministratori)"""
    admin_email = verify_admin_user(request)
    
    # Trova il post
    post = db.query(Post).filter(Post.id == post_id).first()
    if not post:
        raise HTTPException(status_code=404, detail="Post non trovato")
    
    logger.info(f"Eliminazione post {post_id}: '{post.titolo}' di {post.autore_email}")
    
    # Elimina il post (i commenti vengono eliminati automaticamente per cascade)
    db.delete(post)
    db.commit()
    
    logger.info(f"Post {post_id} eliminato dall'amministratore {admin_email}")
    
    return {
        "success": True,
        "message": f"Post {post_id} eliminato con successo",
        "deleted_by": admin_email
    }

@router.delete("/comments/{comment_id}")
def delete_comment(comment_id: int, request: Request, db: Session = Depends(get_db)):
    """Elimina un commento (solo amministratori)"""
    admin_email = verify_admin_user(request)
    
    # Trova il commento
    comment = db.query(Comment).filter(Comment.id == comment_id).first()
    if not comment:
        raise HTTPException(status_code=404, detail="Commento non trovato")
    
    logger.info(f"Eliminazione commento {comment_id} del post {comment.post_id}")
    
    # Elimina il commento
    db.delete(comment)
    db.commit()
    
    logger.info(f"Commento {comment_id} eliminato dall'amministratore {admin_email}")
    
    return {
        "success": True,
        "message": f"Commento {comment_id} eliminato con successo",
        "deleted_by": admin_email
    }

@router.get("/posts/by-user/{user_email}")
def get_posts_by_user_admin(user_email: str, request: Request, db: Session = Depends(get_db)):
    """Ottieni post di un utente specifico (admin)"""
    verify_admin_user(request)
    
    try:
        posts = db.query(Post).filter(Post.autore_email == user_email).order_by(Post.created_at.desc()).all()
        
        posts_data = []
        for post in posts:
            participants_count = get_participants_count(post.id)
            
            post_data = {
                "id": post.id,
                "titolo": post.titolo,
                "sport": post.sport,
                "citta": post.citta,
                "provincia": post.provincia,
                "data_partita": post.data_partita.strftime("%Y-%m-%d"),
                "ora_partita": post.ora_partita.strftime("%H:%M"),
                "numero_giocatori": post.numero_giocatori,
                "partecipanti_iscritti": participants_count,
                "livello": post.livello,
                "created_at": post.created_at if hasattr(post, 'created_at') else datetime.utcnow()
            }
            posts_data.append(post_data)
            
        return posts_data
        
    except Exception as e:
        logger.error(f"Errore recupero post utente {user_email}: {e}")
        raise HTTPException(status_code=500, detail="Errore recupero post utente")

@router.get("/posts")
def get_all_posts_for_admin(request: Request, db: Session = Depends(get_db)):
    """Ottiene tutti i post per il pannello admin"""
    try:
        verify_admin_user(request)
        
        posts = db.query(Post).order_by(Post.id.desc()).all()
        
        admin_posts = []
        for post in posts:
            participants_count = get_participants_count(post.id)
            posti_disponibili = max(0, post.numero_giocatori - participants_count)
            status = "Completo" if posti_disponibili == 0 else "Aperto"
            
            admin_post = {
                "id": post.id,
                "titolo": post.titolo,
                "autore_email": post.autore_email,
                "sport": post.sport,
                "citta": post.citta,
                "provincia": post.provincia,
                "data_creazione": post.created_at if hasattr(post, 'created_at') else datetime.utcnow(),
                "data_partita": post.data_partita,
                "ora_partita": post.ora_partita,
                "numero_giocatori": post.numero_giocatori,
                "partecipanti_iscritti": participants_count,
                "posti_liberi": posti_disponibili,
                "status": status,
                "livello": getattr(post, 'livello', 'Intermedio'),
                "commento": post.commento
            }
            admin_posts.append(admin_post)
        
        logger.info(f"Restituiti {len(admin_posts)} post per admin")
        return admin_posts
        
    except Exception as e:
        logger.error(f"Errore nel recupero post admin: {e}")
        return []

@router.get("/comments")
def get_all_comments_for_admin(request: Request, db: Session = Depends(get_db)):
    """Ottiene tutti i commenti per il pannello admin"""
    try:
        verify_admin_user(request)
        
        comments_query = db.query(Comment).join(Post, Comment.post_id == Post.id).order_by(Comment.created_at.desc())
        comments = comments_query.all()
        
        admin_comments = []
        for comment in comments:
            admin_comment = {
                "id": comment.id,
                "post_id": comment.post_id,
                "post_titolo": comment.post.titolo if comment.post else f"Post {comment.post_id}",
                "autore_email": comment.autore_email,
                "contenuto": comment.contenuto,
                "data_creazione": comment.created_at,
                "contenuto_preview": comment.contenuto[:100] + "..." if len(comment.contenuto) > 100 else comment.contenuto
            }
            admin_comments.append(admin_comment)
        
        logger.info(f"Restituiti {len(admin_comments)} commenti per admin")
        return admin_comments
        
    except Exception as e:
        logger.error(f"Errore nel recupero commenti admin: {e}")
        return []

@router.get("/stats")
def get_admin_stats(request: Request, db: Session = Depends(get_db)):
    """Statistiche dettagliate per amministratori"""
    verify_admin_user(request)
    
    try:
        # Statistiche base
        total_posts = db.query(Post).count()
        total_comments = db.query(Comment).count()
        total_fields = db.query(SportField).count()
        
        # Statistiche aggiuntive
        posts_today = db.query(Post).filter(
            Post.created_at >= datetime.utcnow().date()
        ).count() if hasattr(Post, 'created_at') else 0
        
        comments_today = db.query(Comment).filter(
            Comment.created_at >= datetime.utcnow().date()
        ).count()
        
        # Sport più popolari
        sport_stats = db.execute("""
            SELECT sport, COUNT(*) as count 
            FROM posts 
            GROUP BY sport 
            ORDER BY count DESC 
            LIMIT 5
        """).fetchall()
        
        # Province più attive
        province_stats = db.execute("""
            SELECT provincia, COUNT(*) as count 
            FROM posts 
            GROUP BY provincia 
            ORDER BY count DESC 
            LIMIT 5
        """).fetchall()
        
        # Prova a ottenere statistiche utenti dal servizio auth
        user_stats = {}
        try:
            user_response = requests.get(
                f"{settings.AUTH_SERVICE_URL}/admin/stats",
                cookies={"session_id": request.cookies.get("session_id")},
                timeout=5
            )
            if user_response.status_code == 200:
                user_stats = user_response.json()
        except Exception as e:
            logger.error(f"Errore recupero statistiche utenti: {e}")
            user_stats = {
                "total_users": 0,
                "users_today": 0,
                "active_users": 0
            }
        
        return {
            "total_posts": total_posts,
            "total_comments": total_comments,
            "total_sport_fields": total_fields,
            "posts_today": posts_today,
            "comments_today": comments_today,
            "popular_sports": [{"sport": row[0], "count": row[1]} for row in sport_stats],
            "active_provinces": [{"provincia": row[0], "count": row[1]} for row in province_stats],
            "user_stats": user_stats,
            "timestamp": datetime.utcnow().isoformat()
        }
        
    except Exception as e:
        logger.error(f"Errore statistiche: {e}")
        return {
            "total_posts": 0,
            "total_comments": 0,
            "total_sport_fields": 0,
            "posts_today": 0,
            "comments_today": 0,
            "popular_sports": [],
            "active_provinces": [],
            "user_stats": {
                "total_users": 0,
                "users_today": 0,
                "active_users": 0
            },
            "timestamp": datetime.utcnow().isoformat(),
            "error": str(e)
        }