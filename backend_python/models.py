from sqlalchemy import Column, Integer, String, Date, Time, DateTime, ForeignKey, Text, Float, JSON
from sqlalchemy.orm import relationship
from database import Base
from datetime import datetime

class SportField(Base):
    __tablename__ = "sport_fields"

    id = Column(Integer, primary_key=True, index=True)
    nome = Column(String, nullable=False)
    indirizzo = Column(String, nullable=False)
    provincia = Column(String, nullable=False)
    citta = Column(String, nullable=False)
    lat = Column(Float, nullable=False)
    lng = Column(Float, nullable=False)
    tipo = Column(String, nullable=True)  # "Sintetico", "Erba naturale", "Terra battuta", etc.
    descrizione = Column(Text, nullable=True)
    sports_disponibili = Column(JSON, nullable=True)  # Lista di sport supportati

class Post(Base):
    __tablename__ = "posts"

    id = Column(Integer, primary_key=True, index=True)
    titolo = Column(String, nullable=False)
    provincia = Column(String, nullable=False)
    citta = Column(String, nullable=False)
    sport = Column(String, nullable=False)
    data_partita = Column(Date, nullable=False)
    ora_partita = Column(Time, nullable=False)
    commento = Column(String, nullable=True)
    autore_email = Column(String, nullable=False)
    # Campo da calcio selezionato (ora rinominato più genericamente)
    campo_id = Column(Integer, ForeignKey("sport_fields.id"), nullable=True)
    
    # Relazioni
    comments = relationship("Comment", back_populates="post", cascade="all, delete-orphan")
    campo = relationship("SportField")

class Comment(Base):
    __tablename__ = "comments"

    id = Column(Integer, primary_key=True, index=True)
    post_id = Column(Integer, ForeignKey("posts.id"), nullable=False)
    autore_email = Column(String, nullable=False)
    contenuto = Column(Text, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    
    # Relazione con il post
    post = relationship("Post", back_populates="comments")