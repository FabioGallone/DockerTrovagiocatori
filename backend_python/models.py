from sqlalchemy import Column, Integer, String, Date, Time
from database import Base

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
