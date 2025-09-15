import os
from typing import List

class Settings:
    # Database
    DB_HOST: str = os.getenv("DB_HOST", "localhost")
    DB_USER: str = os.getenv("DB_USER", "APG")
    DB_PASSWORD: str = os.getenv("DB_PASSWORD")  
    DB_NAME: str = os.getenv("DB_NAME", "ProgCarc")
    
    def __init__(self):
        if not self.DB_PASSWORD:
            raise ValueError("DB_PASSWORD environment variable is required")
    
    @property
    def DATABASE_URL(self) -> str:
        return f"postgresql://{self.DB_USER}:{self.DB_PASSWORD}@{self.DB_HOST}/{self.DB_NAME}"
    
    # Auth Service
    AUTH_SERVICE_URL: str = "http://auth-service:8080"
    
    # Socket.IO
    SOCKETIO_PATH: str = "/ws/socket.io"
    CORS_ORIGINS: List[str] = ["*"]
    
    # File paths
    FIELDS_JSON_PATH: str = "campi.json"

settings = Settings()