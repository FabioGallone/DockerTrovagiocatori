import json
from sqlalchemy.orm import Session
from database.models import SportField
from config.settings import settings
import logging

logger = logging.getLogger(__name__)

def load_sport_fields(db: Session) -> None:
    "Carica i campi sportivi dal file JSON nel database"
    try:
        # Controlla se ci sono già campi nel database
        existing_fields = db.query(SportField).first()
        if existing_fields:
            logger.info("Campi sportivi già presenti nel database")
            return
        
        # Carica dal file JSON
        with open(settings.FIELDS_JSON_PATH, 'r', encoding='utf-8') as f:
            fields_data = json.load(f)
        
        # Inserisce i campi nel database
        for field_data in fields_data:
            field = SportField(
                nome=field_data["nome"],
                indirizzo=field_data["indirizzo"],
                provincia=field_data["provincia"],
                citta=field_data["citta"],
                lat=field_data["lat"],
                lng=field_data["lng"],
                tipo=field_data.get("tipo"),
                descrizione=field_data.get("descrizione"),
                sports_disponibili=field_data.get("sports_disponibili", [])
            )
            db.add(field)
        
        db.commit()
        logger.info(f"Caricati {len(fields_data)} campi sportivi nel database")
    
    except Exception as e:
        logger.error(f"Errore durante il caricamento dei campi: {e}")
        db.rollback()
        raise