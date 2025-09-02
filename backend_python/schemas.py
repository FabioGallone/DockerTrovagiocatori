from pydantic import BaseModel, validator
from datetime import datetime, date, time
from typing import List, Optional

class SportFieldResponse(BaseModel):
    id: int
    nome: str
    indirizzo: str
    provincia: str
    citta: str
    lat: float
    lng: float
    tipo: Optional[str] = None
    descrizione: Optional[str] = None
    sports_disponibili: Optional[List[str]] = []

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
    campo_id: Optional[int] = None  # Campo sportivo selezionato
    livello: str = "Intermedio"  # NUOVO CAMPO LIVELLO con default
    numero_giocatori: int = 1  # NUOVO CAMPO NUMERO GIOCATORI

    @validator('numero_giocatori')
    def validate_numero_giocatori(cls, v):
        if v < 1 or v > 50:
            raise ValueError('Il numero di giocatori deve essere tra 1 e 50')
        return v

    @validator('livello')
    def validate_livello(cls, v):
        allowed_levels = ['Principiante', 'Intermedio', 'Avanzato']
        if v not in allowed_levels:
            raise ValueError(f'Livello deve essere uno tra: {", ".join(allowed_levels)}')
        return v

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
    campo: Optional[SportFieldResponse] = None  # Informazioni complete del campo sportivo
    comments: Optional[List[CommentResponse]] = []
    livello: str = "Intermedio"  # NUOVO CAMPO LIVELLO
    numero_giocatori: int = 1  # NUOVO CAMPO NUMERO GIOCATORI

    class Config:
        orm_mode = True