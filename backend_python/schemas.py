from pydantic import BaseModel, validator
from datetime import datetime, date, time

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

class PostResponse(PostCreate):
    id: int
    autore_email: str

    class Config:
        # Se usi Pydantic v1, usa orm_mode
        orm_mode = True
        # Se usi Pydantic v2, puoi invece usare:
        # from_attributes = True
