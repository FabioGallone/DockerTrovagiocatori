import datetime
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

# CORREZIONE: Import Socket.IO corretto
import socketio
from socketio_handler import sio

app = FastAPI(title="TrovaGiocatori API", description="API per la gestione di eventi sportivi e campi")

# Creazione delle tabelle nel database
Base.metadata.create_all(bind=engine)

# CORREZIONE: Monta Socket.IO correttamente
sio_asgi_app = socketio.ASGIApp(sio, other_asgi_app=app, socketio_path="/ws/socket.io")

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
    print("[INFO] ✅ Socket.IO server integrato correttamente")

# CORREZIONE: Aggiunto endpoint per testare Socket.IO
@app.get("/ws/test")
async def test_socket_endpoint():
    """Endpoint di test per verificare che Socket.IO sia configurato"""
    return {
        "message": "Socket.IO is running",
        "socketio_path": "/ws/socket.io/",
        "status": "OK"
    }

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
        livello=post.livello,  
        numero_giocatori=post.numero_giocatori 
    )
    
    db.add(new_post)
    db.commit()
    db.refresh(new_post)
    
    print(f"[INFO] Post creato con ID: {new_post.id}, livello: {new_post.livello}, giocatori: {new_post.numero_giocatori}")
    return new_post

# NUOVO: Endpoint specifico per ottenere i post dell'utente corrente
@app.get("/posts/by-user")
def get_posts_by_user(request: Request, db: Session = Depends(get_db)):
    """Ottieni tutti i post creati dall'utente autenticato con informazioni sui partecipanti"""
    try:
        user_email = get_current_user_email(request)
        print(f"[MY_POSTS] Recupero post per utente: {user_email}")
        
        # Ottieni tutti i post creati dall'utente
        posts = db.query(Post).filter(Post.autore_email == user_email).order_by(Post.data_partita.desc()).all()
        
        print(f"[MY_POSTS] Trovati {len(posts)} post nel database per l'utente {user_email}")
        
        if not posts:
            print(f"[MY_POSTS] Nessun post trovato per l'utente {user_email}")
            return []
        
        # Arricchisce ogni post con informazioni sui partecipanti
        enriched_posts = []
        for post in posts:
            print(f"[MY_POSTS] Elaborando post ID {post.id}: {post.titolo}")
            
            # Ottieni il numero di partecipanti
            participants_count = 0
            try:
                response = requests.get(f"http://auth-service:8080/events/{post.id}/participants", timeout=3)
                if response.status_code == 200:
                    data = response.json()
                    participants_count = data.get("count", 0)
                    print(f"[MY_POSTS] Post {post.id} ha {participants_count} partecipanti")
                else:
                    print(f"[MY_POSTS] Errore nel recupero partecipanti per post {post.id}: {response.status_code}")
            except Exception as e:
                print(f"[MY_POSTS] Eccezione nel recupero partecipanti per post {post.id}: {e}")
            
            # Calcola disponibilità
            posti_disponibili = max(0, post.numero_giocatori - participants_count)
            is_full = posti_disponibili == 0
            
            # Crea il dizionario del post con informazioni aggiuntive
            post_dict = {
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
                "is_full": is_full
            }
            enriched_posts.append(post_dict)
        
        print(f"[MY_POSTS] Restituendo {len(enriched_posts)} post arricchiti")
        return enriched_posts
        
    except HTTPException as he:
        print(f"[MY_POSTS] HTTPException: {he.detail}")
        raise he
    except Exception as e:
        print(f"[MY_POSTS] Errore generico: {e}")
        raise HTTPException(status_code=500, detail=f"Errore interno del server: {str(e)}")

@app.get("/posts/search")
def search_posts_with_participants(provincia: str, sport: str, livello: str = None, db: Session = Depends(get_db)):
    """Cerca post per provincia, sport e opzionalmente per livello, includendo info sui partecipanti"""
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
    
    # Arricchisce ogni post con informazioni sui partecipanti
    enriched_posts = []
    for post in posts:
        # Ottieni il numero di partecipanti
        try:
            response = requests.get(f"http://auth-service:8080/events/{post.id}/participants", timeout=3)
            if response.status_code == 200:
                data = response.json()
                participants_count = data.get("count", 0)
            else:
                participants_count = 0
        except Exception as e:
            print(f"[DEBUG] Errore nel recupero partecipanti per post {post.id}: {e}")
            participants_count = 0
        
        # Crea il dizionario del post con informazioni aggiuntive
        post_dict = {
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
            "posti_disponibili": max(0, post.numero_giocatori - participants_count)
        }
        enriched_posts.append(post_dict)
    
    return enriched_posts

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

# ========== ENDPOINT PARTECIPAZIONE EVENTI ==========

@app.get("/posts/{post_id}/participants-count")
def get_post_participants_count(post_id: int):
    """Ottiene il numero di partecipanti iscritti a un evento"""
    try:
        response = requests.get(f"http://auth-service:8080/events/{post_id}/participants", timeout=5)
        if response.status_code == 200:
            data = response.json()
            return {
                "success": True,
                "post_id": post_id,
                "participants_count": data.get("count", 0),
                "participants": data.get("participants", [])
            }
        else:
            return {
                "success": False,
                "post_id": post_id,
                "participants_count": 0,
                "participants": []
            }
    except Exception as e:
        print(f"[ERROR] Errore nel recupero partecipanti per post {post_id}: {e}")
        return {
            "success": False,
            "post_id": post_id,
            "participants_count": 0,
            "participants": []
        }

@app.get("/posts/{post_id}/availability")
def get_post_availability(post_id: int, db: Session = Depends(get_db)):
    """Calcola i posti disponibili per un evento"""
    # Ottieni il post
    post = db.query(Post).filter(Post.id == post_id).first()
    if not post:
        raise HTTPException(status_code=404, detail="Post non trovato")
    
    # Ottieni il numero di partecipanti iscritti
    try:
        response = requests.get(f"http://auth-service:8080/events/{post_id}/participants", timeout=5)
        if response.status_code == 200:
            data = response.json()
            participants_count = data.get("count", 0)
        else:
            participants_count = 0
    except Exception as e:
        print(f"[ERROR] Errore nel recupero partecipanti: {e}")
        participants_count = 0
    
    # Calcola i posti disponibili
    posti_disponibili = max(0, post.numero_giocatori - participants_count)
    
    return {
        "success": True,
        "post_id": post_id,
        "numero_giocatori_richiesti": post.numero_giocatori,
        "partecipanti_iscritti": participants_count,
        "posti_disponibili": posti_disponibili,
        "is_full": posti_disponibili == 0
    }

# Endpoint per ottenere i dettagli di un singolo post con partecipanti
@app.get("/posts/{post_id}/details")
def get_post_with_participants(post_id: int, db: Session = Depends(get_db)):
    """Ottieni un post specifico con informazioni sui partecipanti"""
    post = db.query(Post).filter(Post.id == post_id).first()
    if not post:
        raise HTTPException(status_code=404, detail="Post non trovato")
    
    # Ottieni informazioni sui partecipanti
    try:
        response = requests.get(f"http://auth-service:8080/events/{post_id}/participants", timeout=5)
        if response.status_code == 200:
            participants_data = response.json()
            participants = participants_data.get("participants", [])
            participants_count = participants_data.get("count", 0)
        else:
            participants = []
            participants_count = 0
    except Exception as e:
        print(f"[ERROR] Errore nel recupero partecipanti: {e}")
        participants = []
        participants_count = 0
    
    # Restituisci il post arricchito con informazioni sui partecipanti
    return {
        "post": post,
        "participants": participants,
        "participants_count": participants_count,
        "posti_disponibili": max(0, post.numero_giocatori - participants_count),
        "is_full": participants_count >= post.numero_giocatori
    }

# ========== ENDPOINT DI SISTEMA ==========

@app.get("/")
def root():
    """Endpoint di base per verificare che l'API sia attiva"""
    return {"message": "TrovaGiocatori API è attiva!", "status": "OK"}

@app.get("/health")
def health_check():
    """Endpoint per verificare lo stato dell'API"""
    return {"status": "healthy", "service": "trovagiocatori-api"}


# ========== ENDPOINT AMMINISTRATORE ==========

def verify_admin_user(request: Request):
    """Verifica che l'utente sia un amministratore"""
    try:
        user_email = get_current_user_email(request)
        
        # Chiama l'auth service per verificare se è admin
        session_cookie = request.cookies.get("session_id")
        auth_response = requests.get(
            "http://auth-service:8080/profile",
            cookies={"session_id": session_cookie},
            timeout=5
        )
        
        if auth_response.status_code != 200:
            raise HTTPException(status_code=403, detail="Non autorizzato")
        
        user_data = auth_response.json()
        if not user_data.get("is_admin", False):
            print(f"[ADMIN] Accesso negato per {user_email}: non è admin")
            raise HTTPException(status_code=403, detail="Privilegi amministratore richiesti")
        
        print(f"[ADMIN] Accesso consentito per admin {user_email}")
        return user_email
    except HTTPException:
        raise
    except Exception as e:
        print(f"[ADMIN] Errore verifica admin: {e}")
        raise HTTPException(status_code=500, detail="Errore verifica privilegi admin")

@app.delete("/admin/posts/{post_id}")
def admin_delete_post(post_id: int, request: Request, db: Session = Depends(get_db)):
    """Elimina un post (solo amministratori)"""
    # Verifica privilegi admin
    admin_email = verify_admin_user(request)
    
    # Trova il post
    post = db.query(Post).filter(Post.id == post_id).first()
    if not post:
        print(f"[ADMIN] Post {post_id} non trovato")
        raise HTTPException(status_code=404, detail="Post non trovato")
    
    print(f"[ADMIN] Eliminazione post {post_id}: '{post.titolo}' di {post.autore_email}")
    
    # Elimina il post (i commenti vengono eliminati automaticamente per cascade)
    db.delete(post)
    db.commit()
    
    print(f"[ADMIN] ✅ Post {post_id} eliminato dall'amministratore {admin_email}")
    
    return {
        "success": True,
        "message": f"Post {post_id} eliminato con successo",
        "deleted_by": admin_email
    }

@app.delete("/admin/comments/{comment_id}")
def admin_delete_comment(comment_id: int, request: Request, db: Session = Depends(get_db)):
    """Elimina un commento (solo amministratori)"""
    # Verifica privilegi admin
    admin_email = verify_admin_user(request)
    
    # Trova il commento
    comment = db.query(Comment).filter(Comment.id == comment_id).first()
    if not comment:
        print(f"[ADMIN] Commento {comment_id} non trovato")
        raise HTTPException(status_code=404, detail="Commento non trovato")
    
    print(f"[ADMIN] Eliminazione commento {comment_id} del post {comment.post_id}")
    
    # Elimina il commento
    db.delete(comment)
    db.commit()
    
    print(f"[ADMIN] ✅ Commento {comment_id} eliminato dall'amministratore {admin_email}")
    
    return {
        "success": True,
        "message": f"Commento {comment_id} eliminato con successo",
        "deleted_by": admin_email
    }

@app.get("/admin/stats")
def get_admin_stats(request: Request, db: Session = Depends(get_db)):
    """Statistiche per amministratori"""
    # Verifica privilegi admin
    verify_admin_user(request)
    
    total_posts = db.query(Post).count()
    total_comments = db.query(Comment).count()
    total_fields = db.query(SportField).count()
    
    return {
        "total_posts": total_posts,
        "total_comments": total_comments,
        "total_sport_fields": total_fields,
        "timestamp": datetime.utcnow().isoformat()
    }

# IMPORTANTE: Esporta l'app ASGI corretta per uvicorn
app = sio_asgi_app


@app.get("/admin/posts")
def get_admin_posts(request: Request, db: Session = Depends(get_db)):
    """Lista tutti i post per admin con statistiche partecipanti"""
    # Verifica privilegi admin
    verify_admin_user(request)
    
    query = """
        SELECT 
            p.id, p.titolo, p.autore_email, p.sport, p.citta, p.provincia,
            p.data_partita, p.ora_partita, p.numero_giocatori, p.livello,
            p.created_at,
            COUNT(ep.user_id) as partecipanti_iscritti
        FROM posts p
        LEFT JOIN event_participants ep ON p.id = ep.post_id AND ep.status = 'confirmed'
        GROUP BY p.id, p.titolo, p.autore_email, p.sport, p.citta, p.provincia,
                 p.data_partita, p.ora_partita, p.numero_giocatori, p.livello, p.created_at
        ORDER BY p.created_at DESC
    """
    
    try:
        result = db.execute(query)
        posts = []
        
        for row in result:
            # Determina lo status del post
            posti_liberi = max(0, row.numero_giocatori - row.partecipanti_iscritti)
            status = "Completo" if posti_liberi == 0 else "Aperto"
            
            post = {
                "id": row.id,
                "titolo": row.titolo,
                "autore_email": row.autore_email,
                "sport": row.sport,
                "citta": row.citta,
                "provincia": row.provincia,
                "data_creazione": row.created_at,
                "data_partita": row.data_partita,
                "numero_giocatori": row.numero_giocatori,
                "partecipanti_iscritti": row.partecipanti_iscritti,
                "posti_liberi": posti_liberi,
                "status": status,
                "livello": row.livello
            }
            posts.append(post)
            
        return posts
        
    except Exception as e:
        print(f"[ADMIN] Errore recupero post: {e}")
        raise HTTPException(status_code=500, detail="Errore recupero post")

@app.get("/admin/comments")
def get_admin_comments(request: Request, db: Session = Depends(get_db)):
    """Lista tutti i commenti per admin"""
    # Verifica privilegi admin
    verify_admin_user(request)
    
    try:
        # Query per ottenere commenti con informazioni del post
        comments = db.query(Comment).order_by(Comment.created_at.desc()).all()
        
        enriched_comments = []
        for comment in comments:
            # Ottieni informazioni del post
            post = db.query(Post).filter(Post.id == comment.post_id).first()
            post_titolo = post.titolo if post else f"Post #{comment.post_id}"
            
            comment_data = {
                "id": comment.id,
                "post_id": comment.post_id,
                "post_titolo": post_titolo,
                "autore_email": comment.autore_email,
                "contenuto": comment.contenuto,
                "data_creazione": comment.created_at,
                "contenuto_preview": comment.contenuto[:100] + "..." if len(comment.contenuto) > 100 else comment.contenuto
            }
            enriched_comments.append(comment_data)
            
        return enriched_comments
        
    except Exception as e:
        print(f"[ADMIN] Errore recupero commenti: {e}")
        raise HTTPException(status_code=500, detail="Errore recupero commenti")

@app.get("/admin/posts/by-user/{user_email}")
def get_posts_by_user_admin(user_email: str, request: Request, db: Session = Depends(get_db)):
    """Ottieni post di un utente specifico (admin)"""
    # Verifica privilegi admin
    verify_admin_user(request)
    
    try:
        posts = db.query(Post).filter(Post.autore_email == user_email).order_by(Post.created_at.desc()).all()
        
        posts_data = []
        for post in posts:
            # Conta partecipanti
            try:
                response = requests.get(f"http://auth-service:8080/events/{post.id}/participants", timeout=3)
                participants_count = response.json().get("count", 0) if response.status_code == 200 else 0
            except:
                participants_count = 0
            
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
        print(f"[ADMIN] Errore recupero post utente {user_email}: {e}")
        raise HTTPException(status_code=500, detail="Errore recupero post utente")

@app.get("/admin/posts")
def get_all_posts_for_admin(request: Request, db: Session = Depends(get_db)):
    """Ottiene tutti i post per il pannello admin"""
    try:
        # Verifica privilegi admin (usa la funzione esistente)
        admin_email = verify_admin_user(request)
        
        # Ottieni tutti i post
        posts = db.query(Post).order_by(Post.id.desc()).all()
        
        admin_posts = []
        for post in posts:
            # Conta i partecipanti
            participants_count = 0
            try:
                response = requests.get(f"http://auth-service:8080/events/{post.id}/participants", timeout=3)
                if response.status_code == 200:
                    data = response.json()
                    participants_count = data.get("count", 0)
            except:
                participants_count = 0
            
            # Calcola disponibilità
            posti_disponibili = max(0, post.numero_giocatori - participants_count)
            status = "Completo" if posti_disponibili == 0 else "Aperto"
            
            admin_post = {
                "id": post.id,
                "titolo": post.titolo,
                "autore_email": post.autore_email,
                "sport": post.sport,
                "citta": post.citta,
                "provincia": post.provincia,
                "data_creazione": datetime.utcnow(),  # Placeholder
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
        
        print(f"[ADMIN] Restituiti {len(admin_posts)} post per admin")
        return admin_posts
        
    except Exception as e:
        print(f"[ADMIN] Errore nel recupero post admin: {e}")
        # Fallback: restituisci lista vuota invece di errore
        return []

@app.get("/admin/comments")
def get_all_comments_for_admin(request: Request, db: Session = Depends(get_db)):
    """Ottiene tutti i commenti per il pannello admin"""
    try:
        # Verifica privilegi admin
        admin_email = verify_admin_user(request)
        
        # Ottieni tutti i commenti con il post
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
        
        print(f"[ADMIN] Restituiti {len(admin_comments)} commenti per admin")
        return admin_comments
        
    except Exception as e:
        print(f"[ADMIN] Errore nel recupero commenti admin: {e}")
        # Fallback: restituisci lista vuota
        return []

@app.get("/admin/dashboard-stats")
def get_admin_dashboard_stats(request: Request, db: Session = Depends(get_db)):
    """Statistiche per la dashboard admin"""
    try:
        # Verifica privilegi admin
        admin_email = verify_admin_user(request)
        
        # Statistiche base
        total_posts = db.query(Post).count()
        total_comments = db.query(Comment).count()
        total_fields = db.query(SportField).count()
        
        # Prova a ottenere gli utenti dal Go service
        total_users = 0
        try:
            user_response = requests.get("http://auth-service:8080/admin/stats", 
                                       cookies={"session_id": request.cookies.get("session_id")}, 
                                       timeout=5)
            if user_response.status_code == 200:
                user_data = user_response.json()
                total_users = user_data.get("total_users", 0)
        except:
            total_users = 0
        
        return {
            "total_posts": total_posts,
            "total_comments": total_comments,
            "total_sport_fields": total_fields,
            "total_users": total_users,
            "generated_at": datetime.utcnow().isoformat()
        }
        
    except Exception as e:
        print(f"[ADMIN] Errore calcolo statistiche: {e}")
        # Fallback con dati base
        return {
            "total_posts": 0,
            "total_comments": 0,
            "total_sport_fields": 0,
            "total_users": 0,
            "generated_at": datetime.utcnow().isoformat()
        }

# Assicurati che questo endpoint sia aggiornato
@app.get("/admin/stats")
def get_admin_stats(request: Request, db: Session = Depends(get_db)):
    """Statistiche per amministratori - versione base"""
    try:
        # Verifica privilegi admin
        verify_admin_user(request)
        
        total_posts = db.query(Post).count()
        total_comments = db.query(Comment).count()
        total_fields = db.query(SportField).count()
        
        return {
            "total_posts": total_posts,
            "total_comments": total_comments,
            "total_sport_fields": total_fields,
            "timestamp": datetime.utcnow().isoformat()
        }
    except Exception as e:
        print(f"[ADMIN] Errore nel calcolo statistiche base: {e}")
        return {
            "total_posts": 0,
            "total_comments": 0,
            "total_sport_fields": 0,
            "timestamp": datetime.utcnow().isoformat()
        }

# Migliora l'endpoint statistiche
@app.get("/admin/stats")
def get_admin_stats_enhanced(request: Request, db: Session = Depends(get_db)):
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
        
        return {
            "total_posts": total_posts,
            "total_comments": total_comments,
            "total_sport_fields": total_fields,
            "posts_today": posts_today,
            "comments_today": comments_today,
            "popular_sports": [{"sport": row.sport, "count": row.count} for row in sport_stats],
            "active_provinces": [{"provincia": row.provincia, "count": row.count} for row in province_stats],
            "timestamp": datetime.utcnow().isoformat()
        }
        
    except Exception as e:
        print(f"[ADMIN] Errore statistiche: {e}")
        return {
            "total_posts": 0,
            "total_comments": 0,
            "total_sport_fields": 0,
            "posts_today": 0,
            "comments_today": 0,
            "popular_sports": [],
            "active_provinces": [],
            "timestamp": datetime.utcnow().isoformat(),
            "error": str(e)
        }