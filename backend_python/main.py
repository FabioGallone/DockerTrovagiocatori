from typing import List
from fastapi import FastAPI, HTTPException, Depends
import requests
from sqlalchemy.orm import Session
from database import engine, SessionLocal, Base
from models import Post, Comment
from schemas import PostCreate, PostResponse, CommentCreate, CommentResponse
from starlette.requests import Request

app = FastAPI()

# Creazione delle tabelle nel database
Base.metadata.create_all(bind=engine)

# Dependency per la sessione del database
def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()

# Ottieni l'email dell'utente autenticato dall'auth service (Go)
def get_current_user_email(request: Request):
    session_cookie = request.cookies.get("session_id")
    print(f"[DEBUG] Session cookie ricevuto: {session_cookie}")

    if not session_cookie:
        print("[DEBUG] ERRORE: Nessun session cookie trovato")
        raise HTTPException(status_code=401, detail="Session cookie not found")

    # Usa il nome del servizio nel network Docker
    try:
        auth_url = "http://auth-service:8080/api/user"
        print(f"[DEBUG] Chiamata auth service: {auth_url}")
        print(f"[DEBUG] Cookie da inviare: session_id={session_cookie}")
        
        response = requests.get(auth_url, cookies={"session_id": session_cookie}, timeout=10)
        
        print(f"[DEBUG] Auth service status: {response.status_code}")
        print(f"[DEBUG] Auth service response: {response.text}")
        
        if response.status_code != 200:
            print(f"[DEBUG] Auth service rejected session: {session_cookie}")
            raise HTTPException(status_code=401, detail=f"Invalid session - Auth service returned {response.status_code}")
        
        user_data = response.json()
        user_email = user_data["email"]
        print(f"[DEBUG] Email utente ottenuta: {user_email}")
        return user_email
        
    except requests.exceptions.RequestException as e:
        print(f"[DEBUG] Errore connessione auth service: {e}")
        raise HTTPException(status_code=500, detail="Auth service unavailable")


# Endpoint per creare un post
@app.post("/posts/", response_model=PostResponse)
def create_post(post: PostCreate, request: Request, db: Session = Depends(get_db)):
    user_email = get_current_user_email(request)
    print("stampo il post ", post)

    # Creazione di un nuovo post con i campi aggiornati
    new_post = Post(
        titolo=post.titolo,
        provincia=post.provincia,  # Nuovo campo
        citta=post.citta,
        sport=post.sport,  # Nuovo campo
        data_partita=post.data_partita,
        ora_partita=post.ora_partita,
        commento=post.commento,
        autore_email=user_email  # Associa il post all'utente autenticato
    )
    db.add(new_post)
    db.commit()
    db.refresh(new_post)
    return new_post

@app.get("/posts/search", response_model=List[PostResponse])
def search_posts(provincia: str, sport: str, db: Session = Depends(get_db)):
    posts = db.query(Post).filter(Post.provincia == provincia, Post.sport == sport).all()
    if not posts:
        raise HTTPException(status_code=404, detail="Nessun post trovato per i criteri specificati")
    return posts

@app.get("/posts/{post_id}", response_model=PostResponse)
def get_post(post_id: int, db: Session = Depends(get_db)):
    post = db.query(Post).filter(Post.id == post_id).first()
    if not post:
        raise HTTPException(status_code=404, detail="Post non trovato")
    return post

# NUOVI ENDPOINT PER I COMMENTI

@app.post("/posts/{post_id}/comments/", response_model=CommentResponse)
def create_comment(post_id: int, comment: CommentCreate, request: Request, db: Session = Depends(get_db)):
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
    
    return new_comment

@app.get("/posts/{post_id}/comments/", response_model=List[CommentResponse])
def get_post_comments(post_id: int, db: Session = Depends(get_db)):
    # Verifica che il post esista
    post = db.query(Post).filter(Post.id == post_id).first()
    if not post:
        raise HTTPException(status_code=404, detail="Post non trovato")
    
    # Ottieni tutti i commenti del post ordinati per data di creazione
    comments = db.query(Comment).filter(Comment.post_id == post_id).order_by(Comment.created_at.desc()).all()
    
    # Arricchisci ogni commento con le informazioni dell'utente
    enriched_comments = []
    for comment in comments:
        try:
            # Chiama l'auth-service per ottenere le info dell'utente
            encoded_email = requests.utils.quote(comment.autore_email)
            user_response = requests.get(f"http://auth-service:8080/api/user/by-email?email={encoded_email}")
            
            if user_response.status_code == 200:
                user_data = user_response.json()
                # Crea un oggetto commento arricchito
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
                # Se non riesco a ottenere i dati utente, uso solo l'email
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
            print(f"[DEBUG] Errore nel recupero dati utente per {comment.autore_email}: {e}")
            # Fallback con solo email
            enriched_comment = CommentResponse(
                id=comment.id,
                post_id=comment.post_id,
                autore_email=comment.autore_email,
                contenuto=comment.contenuto,
                created_at=comment.created_at,
                autore_username=comment.autore_email,  # Usa email come fallback
                autore_nome="",
                autore_cognome=""
            )
            enriched_comments.append(enriched_comment)
    
    return enriched_comments