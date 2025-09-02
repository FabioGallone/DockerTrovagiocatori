from typing import List
from fastapi import FastAPI, HTTPException, Depends
import requests
import json
import os
from sqlalchemy.orm import Session
from database import engine, SessionLocal, Base
from models import Post, Comment, SportField
from schemas import PostCreate, PostResponse, CommentCreate, CommentResponse, SportFieldResponse
from starlette.requests import Request

app = FastAPI(title="TrovaGiocatori API", description="API per la gestione di eventi sportivi e campi")

# Creazione delle tabelle nel database
Base.metadata.create_all(bind=engine)

# Dependency per la sessione del database
def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()

# Funzione per caricare i campi sportivi dal file JSON
def load_sport_fields():
    """Carica i campi sportivi dal file campi_calcio.json nel database"""
    db = SessionLocal()
    try:
        # Controlla se ci sono già campi nel database
        existing_fields = db.query(SportField).first()
        if existing_fields:
            print("[INFO] Campi sportivi già presenti nel database")
            return
        
        # Carica dal file JSON
        json_path = "campi.json"

        with open(json_path, 'r', encoding='utf-8') as f:
            fields_data = json.load(f)
        
        # Inserisce i campi nel database
        for field_data in fields_data:
            field = SportField(
                nome=field_data["nome"],
                indirizzo=field_data["indirizzo"],
                provincia=field_data["provincia"],
                citta=field_data["citta"],
                lat=field_data["lat"],
                lng=field_data["lng"],
                tipo=field_data.get("tipo"),
                descrizione=field_data.get("descrizione"),
                sports_disponibili=field_data.get("sports_disponibili", [])
            )
            db.add(field)
        
        db.commit()
        print(f"[INFO] Caricati {len(fields_data)} campi sportivi nel database")
    
    except Exception as e:
        print(f"[ERROR] Errore durante il caricamento dei campi: {e}")
        db.rollback()
    finally:
        db.close()

# Carica i campi all'avvio dell'applicazione
@app.on_event("startup")
async def startup_event():
    load_sport_fields()

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

# ========== ENDPOINT CAMPI SPORTIVI ==========

@app.get("/fields/", response_model=List[SportFieldResponse])
def get_all_fields(db: Session = Depends(get_db)):
    """Ottieni tutti i campi sportivi"""
    fields = db.query(SportField).all()
    return fields

@app.get("/fields/by-province/{provincia}", response_model=List[SportFieldResponse])
def get_fields_by_province(provincia: str, db: Session = Depends(get_db)):
    """Ottieni i campi sportivi per provincia"""
    fields = db.query(SportField).filter(SportField.provincia == provincia).all()
    return fields

@app.get("/fields/by-sport/{sport}", response_model=List[SportFieldResponse])
def get_fields_by_sport(sport: str, db: Session = Depends(get_db)):
    """Ottieni i campi sportivi che supportano uno sport specifico"""
    fields = db.query(SportField).filter(
        SportField.sports_disponibili.contains([sport])
    ).all()
    return fields

@app.get("/fields/by-province-and-sport/{provincia}/{sport}", response_model=List[SportFieldResponse])
def get_fields_by_province_and_sport(provincia: str, sport: str, db: Session = Depends(get_db)):
    """Ottieni i campi sportivi per provincia che supportano uno sport specifico"""
    fields = db.query(SportField).filter(
        SportField.provincia == provincia,
        SportField.sports_disponibili.contains([sport])
    ).all()
    return fields

@app.get("/fields/{field_id}", response_model=SportFieldResponse)
def get_field(field_id: int, db: Session = Depends(get_db)):
    """Ottieni un campo sportivo specifico"""
    field = db.query(SportField).filter(SportField.id == field_id).first()
    if not field:
        raise HTTPException(status_code=404, detail="Campo sportivo non trovato")
    return field

# ========== ENDPOINT POST ==========

@app.post("/posts/", response_model=PostResponse)
def create_post(post: PostCreate, request: Request, db: Session = Depends(get_db)):
    """Crea un nuovo post/evento sportivo"""
    user_email = get_current_user_email(request)
    print(f"[INFO] Creazione post da utente: {user_email}")

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
        
        print(f"[INFO] Campo verificato: {field.nome} supporta {post.sport}")

    # Crea il nuovo post CON IL LIVELLO E NUMERO GIOCATORI
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
        livello=post.livello,  # NUOVO CAMPO
        numero_giocatori=post.numero_giocatori  # NUOVO CAMPO
    )
    
    db.add(new_post)
    db.commit()
    db.refresh(new_post)
    
    print(f"[INFO] Post creato con ID: {new_post.id}, livello: {new_post.livello}, giocatori: {new_post.numero_giocatori}")
    return new_post

@app.get("/posts/search", response_model=List[PostResponse])
def search_posts(provincia: str, sport: str, livello: str = None, db: Session = Depends(get_db)):
    """Cerca post per provincia, sport e opzionalmente per livello"""
    query = db.query(Post).filter(
        Post.provincia == provincia, 
        Post.sport == sport
    )
    
    # Filtra per livello se specificato
    if livello:
        query = query.filter(Post.livello == livello)
    
    posts = query.all()
    
    if not posts:
        raise HTTPException(status_code=404, detail="Nessun post trovato per i criteri specificati")
    
    return posts

@app.get("/posts/{post_id}", response_model=PostResponse)
def get_post(post_id: int, db: Session = Depends(get_db)):
    """Ottieni un post specifico"""
    post = db.query(Post).filter(Post.id == post_id).first()
    if not post:
        raise HTTPException(status_code=404, detail="Post non trovato")
    return post

# ========== ENDPOINT COMMENTI ==========

@app.post("/posts/{post_id}/comments/", response_model=CommentResponse)
def create_comment(post_id: int, comment: CommentCreate, request: Request, db: Session = Depends(get_db)):
    """Aggiungi un commento a un post"""
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
    
    print(f"[INFO] Commento creato da {user_email} per post {post_id}")
    return new_comment

@app.get("/posts/{post_id}/comments/", response_model=List[CommentResponse])
def get_post_comments(post_id: int, db: Session = Depends(get_db)):
    """Ottieni tutti i commenti di un post con info utente"""
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
                f"http://auth-service:8080/api/user/by-email?email={encoded_email}",
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

# ========== ENDPOINT DI SISTEMA ==========

@app.get("/")
def root():
    """Endpoint di base per verificare che l'API sia attiva"""
    return {"message": "TrovaGiocatori API è attiva!", "status": "OK"}

@app.get("/health")
def health_check():
    """Endpoint per verificare lo stato dell'API"""
    return {"status": "healthy", "service": "trovagiocatori-api"}