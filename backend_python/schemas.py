from pydantic import BaseModel, validator
from datetime import datetime, date, time
from typing import List, Optional

class PostCreate(BaseModel):
    titolo: str
    provincia: str  # Nuovo campo per provincia
    citta: str
    sport: str      # Nuovo campo per sport
    data_partita: date
    ora_partita: time
    commento: str

    @validator('data_partita', pre=True)
    def parse_data_partita(cls, v):
        """
        Prova a interpretare la data nel formato dd-MM-yyyy.
        Se fallisce, tenta il formato ISO (yyyy-MM-dd).
        """
        if isinstance(v, str):
            try:
                # Tenta a interpretare dd-MM-yyyy
                return datetime.strptime(v, "%d-%m-%Y").date()
            except ValueError:
                try:
                    # Tenta a interpretare yyyy-MM-dd
                    return datetime.strptime(v, "%Y-%m-%d").date()
                except ValueError:
                    raise ValueError("data_partita deve essere nel formato dd-MM-yyyy oppure yyyy-MM-dd")
        return v

# Schema per creare un commento
class CommentCreate(BaseModel):
    contenuto: str

# Schema per la risposta del commento
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

class PostResponse(PostCreate):
    id: int
    autore_email: str
    comments: Optional[List[CommentResponse]] = []

    class Config:
        # Se usi Pydantic v1, usa orm_mode
        orm_mode = True
        # Se usi Pydantic v2, puoi invece usare:
        # from_attributes = True