from typing import List
from fastapi import FastAPI, HTTPException, Depends
import requests
from sqlalchemy.orm import Session
from database import engine, SessionLocal, Base
from models import Post
from schemas import PostCreate, PostResponse
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
    print("mi stampo session cookie", session_cookie)

    # Usa il nome del servizio nel network Docker
    response = requests.get("http://auth-service:8080/api/user", cookies={"session_id": session_cookie})
    if response.status_code != 200:
        raise HTTPException(status_code=401, detail="Invalid session")
    print("mi stampo l'email",    response.json()["email"])
    return response.json()["email"]


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