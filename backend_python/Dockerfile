FROM python:3.11

# Imposta la cartella di lavoro dentro il container
WORKDIR /app

# Copia il file requirements.txt nella cartella di lavoro del container
COPY requirements.txt .

# Installa le dipendenze dichiarate in requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# Copia tutto il codice sorgente dell'app nel container
COPY . .

# Copia il file dei campi da calcio
COPY campi.json .

# Esponi la porta 8000 per il server FastAPI
EXPOSE 8000

# Avvia il server usando Uvicorn (che è il server per FastAPI)
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000", "--reload"]