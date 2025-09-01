from pydantic import BaseModel, validator
from datetime import datetime, date, time
from typing import List, Optional

class FootballFieldResponse(BaseModel):
    id: int
    nome: str
    indirizzo: str
    provincia: str
    citta: str
    lat: float
    lng: float
    tipo: Optional[str] = None
    descrizione: Optional[str] = None

    class Config:
        orm_mode = True

class PostCreate(BaseModel):
    titolo: str
    provincia: str
    citta: str
    sport: str
    data_partita: date
    ora_partita: time
    commento: str
    campo_id: Optional[int] = None  #campo opzionale per il campo da calcio

    @validator('data_partita', pre=True)
    def parse_data_partita(cls, v):
        if isinstance(v, str):
            try:
                return datetime.strptime(v, "%d-%m-%Y").date()
            except ValueError:
                try:
                    return datetime.strptime(v, "%Y-%m-%d").date()
                except ValueError:
                    raise ValueError("data_partita deve essere nel formato dd-MM-yyyy oppure yyyy-MM-dd")
        return v

class CommentCreate(BaseModel):
    contenuto: str

class CommentResponse(BaseModel):
    id: int
    post_id: int
    autore_email: str
    contenuto: str
    created_at: datetime
    autore_username: Optional[str] = None
    autore_nome: Optional[str] = None
    autore_cognome: Optional[str] = None

    class Config:
        orm_mode = True

class PostResponse(BaseModel):
    id: int
    titolo: str
    provincia: str
    citta: str
    sport: str
    data_partita: date
    ora_partita: time
    commento: str
    autore_email: str
    campo_id: Optional[int] = None
    campo: Optional[FootballFieldResponse] = None  # Informazioni complete del campo
    comments: Optional[List[CommentResponse]] = []

    class Config:
        orm_mode = True