from sqlalchemy import Column, Integer, String, Date, Time, DateTime, ForeignKey, Text
from sqlalchemy.orm import relationship
from database import Base
from datetime import datetime

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
    autore_email = Column(String, nullable=False)  # Collegato all'utente autenticato
    
    # Relazione con i commenti
    comments = relationship("Comment", back_populates="post", cascade="all, delete-orphan")

class Comment(Base):
    __tablename__ = "comments"

    id = Column(Integer, primary_key=True, index=True)
    post_id = Column(Integer, ForeignKey("posts.id"), nullable=False)
    autore_email = Column(String, nullable=False)
    contenuto = Column(Text, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    
    # Relazione con il post
    post = relationship("Post", back_populates="comments")