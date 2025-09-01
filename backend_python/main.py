from typing import List
from fastapi import FastAPI, HTTPException, Depends
import requests
import json
import os
from sqlalchemy.orm import Session
from database import engine, SessionLocal, Base
from models import Post, Comment, FootballField
from schemas import PostCreate, PostResponse, CommentCreate, CommentResponse, FootballFieldResponse
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

# Funzione per caricare i campi da calcio dal file JSON
def load_football_fields():
    """Carica i campi da calcio dal file campi_calcio.json nel database"""
    db = SessionLocal()
    try:
        # Controlla se ci sono già campi nel database
        existing_fields = db.query(FootballField).first()
        if existing_fields:
            print("[INFO] Campi da calcio già presenti nel database")
            return
        
        # Carica dal file JSON
        json_path = "campi_calcio.json"
        if not os.path.exists(json_path):
            print(f"[WARNING] File {json_path} non trovato. Creazione di dati di esempio...")
            # Crea alcuni campi di esempio se il file non esiste
            example_fields = [
                {
                    "id": 1,
                    "nome": "Centro Sportivo San Siro",
                    "indirizzo": "Via dei Campi 15, Milano",
                    "provincia": "Milano",
                    "citta": "Milano",
                    "lat": 45.4784,
                    "lng": 9.1240,
                    "tipo": "Sintetico",
                    "descrizione": "Campo da calcio a 11 in erba sintetica"
                }
            ]
            fields_data = example_fields
        else:
            with open(json_path, 'r', encoding='utf-8') as f:
                fields_data = json.load(f)
        
        # Inserisce i campi nel database
        for field_data in fields_data:
            field = FootballField(
                nome=field_data["nome"],
                indirizzo=field_data["indirizzo"],
                provincia=field_data["provincia"],
                citta=field_data["citta"],
                lat=field_data["lat"],
                lng=field_data["lng"],
                tipo=field_data.get("tipo"),
                descrizione=field_data.get("descrizione")
            )
            db.add(field)
        
        db.commit()
        print(f"[INFO] Caricati {len(fields_data)} campi da calcio nel database")
    
    except Exception as e:
        print(f"[ERROR] Errore durante il caricamento dei campi: {e}")
        db.rollback()
    finally:
        db.close()

# Carica i campi all'avvio dell'applicazione
@app.on_event("startup")
async def startup_event():
    load_football_fields()

# Ottieni l'email dell'utente autenticato dall'auth service (Go)
def get_current_user_email(request: Request):
    session_cookie = request.cookies.get("session_id")
    print(f"[DEBUG] Session cookie ricevuto: {session_cookie}")

    if not session_cookie:
        print("[DEBUG] ERRORE: Nessun session cookie trovato")
        raise HTTPException(status_code=401, detail="Session cookie not found")

    try:
        auth_url = "http://auth-service:8080/api/user"
        print(f"[DEBUG] Chiamata auth service: {auth_url}")
        
        response = requests.get(auth_url, cookies={"session_id": session_cookie}, timeout=10)
        
        if response.status_code != 200:
            raise HTTPException(status_code=401, detail=f"Invalid session - Auth service returned {response.status_code}")
        
        user_data = response.json()
        user_email = user_data["email"]
        print(f"[DEBUG] Email utente ottenuta: {user_email}")
        return user_email
        
    except requests.exceptions.RequestException as e:
        print(f"[DEBUG] Errore connessione auth service: {e}")
        raise HTTPException(status_code=500, detail="Auth service unavailable")

#  ENDPOINT PER I CAMPI DA CALCIO

@app.get("/football-fields/", response_model=List[FootballFieldResponse])
def get_all_football_fields(db: Session = Depends(get_db)):
    """Ottieni tutti i campi da calcio"""
    fields = db.query(FootballField).all()
    return fields

@app.get("/football-fields/by-province/{provincia}", response_model=List[FootballFieldResponse])
def get_football_fields_by_province(provincia: str, db: Session = Depends(get_db)):
    """Ottieni i campi da calcio per provincia"""
    fields = db.query(FootballField).filter(FootballField.provincia == provincia).all()
    return fields

@app.get("/football-fields/{field_id}", response_model=FootballFieldResponse)
def get_football_field(field_id: int, db: Session = Depends(get_db)):
    """Ottieni un campo da calcio specifico"""
    field = db.query(FootballField).filter(FootballField.id == field_id).first()
    if not field:
        raise HTTPException(status_code=404, detail="Campo da calcio non trovato")
    return field

# ENDPOINT POST AGGIORNATI

@app.post("/posts/", response_model=PostResponse)
def create_post(post: PostCreate, request: Request, db: Session = Depends(get_db)):
    user_email = get_current_user_email(request)
    print("Creazione post:", post)

    # Verifica che il campo da calcio esista se specificato
    if post.campo_id:
        field = db.query(FootballField).filter(FootballField.id == post.campo_id).first()
        if not field:
            raise HTTPException(status_code=404, detail="Campo da calcio non trovato")

    new_post = Post(
        titolo=post.titolo,
        provincia=post.provincia,
        citta=post.citta,
        sport=post.sport,
        data_partita=post.data_partita,
        ora_partita=post.ora_partita,
        commento=post.commento,
        autore_email=user_email,
        campo_id=post.campo_id  # Nuovo campo
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

# ENDPOINT COMMENTI 

@app.post("/posts/{post_id}/comments/", response_model=CommentResponse)
def create_comment(post_id: int, comment: CommentCreate, request: Request, db: Session = Depends(get_db)):
    post = db.query(Post).filter(Post.id == post_id).first()
    if not post:
        raise HTTPException(status_code=404, detail="Post non trovato")
    
    user_email = get_current_user_email(request)
    
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
    post = db.query(Post).filter(Post.id == post_id).first()
    if not post:
        raise HTTPException(status_code=404, detail="Post non trovato")
    
    comments = db.query(Comment).filter(Comment.post_id == post_id).order_by(Comment.created_at.desc()).all()
    
    enriched_comments = []
    for comment in comments:
        try:
            encoded_email = requests.utils.quote(comment.autore_email)
            user_response = requests.get(f"http://auth-service:8080/api/user/by-email?email={encoded_email}")
            
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