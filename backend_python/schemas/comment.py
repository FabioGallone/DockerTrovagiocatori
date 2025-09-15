from pydantic import BaseModel
from datetime import datetime
from typing import Optional

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