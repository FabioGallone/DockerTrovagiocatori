from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session
from typing import List
from database.connection import get_db
from database.models import SportField
from schemas.field import SportFieldResponse

router = APIRouter(prefix="/fields", tags=["Campi Sportivi"])

@router.get("/", response_model=List[SportFieldResponse])
def get_all_fields(db: Session = Depends(get_db)):
    """Ottieni tutti i campi sportivi"""
    fields = db.query(SportField).all()
    return fields

@router.get("/by-province/{provincia}", response_model=List[SportFieldResponse])
def get_fields_by_province(provincia: str, db: Session = Depends(get_db)):
    """Ottieni i campi sportivi per provincia"""
    fields = db.query(SportField).filter(SportField.provincia == provincia).all()
    return fields

@router.get("/by-sport/{sport}", response_model=List[SportFieldResponse])
def get_fields_by_sport(sport: str, db: Session = Depends(get_db)):
    """Ottieni i campi sportivi che supportano uno sport specifico"""
    fields = db.query(SportField).filter(
        SportField.sports_disponibili.contains([sport])
    ).all()
    return fields

@router.get("/by-province-and-sport/{provincia}/{sport}", response_model=List[SportFieldResponse])
def get_fields_by_province_and_sport(provincia: str, sport: str, db: Session = Depends(get_db)):
    """Ottieni i campi sportivi per provincia che supportano uno sport specifico"""
    fields = db.query(SportField).filter(
        SportField.provincia == provincia,
        SportField.sports_disponibili.contains([sport])
    ).all()
    return fields

@router.get("/{field_id}", response_model=SportFieldResponse)
def get_field(field_id: int, db: Session = Depends(get_db)):
    """Ottieni un campo sportivo specifico"""
    field = db.query(SportField).filter(SportField.id == field_id).first()
    if not field:
        raise HTTPException(status_code=404, detail="Campo sportivo non trovato")
    return field