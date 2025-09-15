from pydantic import BaseModel
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