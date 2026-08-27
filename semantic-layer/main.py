from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Optional, Dict, Any
import uuid
import json
import os
import psycopg2
from psycopg2.extras import RealDictCursor
from datetime import datetime

app = FastAPI(title="Semantic Layer Service", version="1.0")

# Подключение к БД (используем ту же Postgres, что и основной бекенд)
DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://postgres:postgres@db:5432/postgres")

def get_db_connection():
    return psycopg2.connect(DATABASE_URL, cursor_factory=RealDictCursor)

# Модели данных
class MetricCreate(BaseModel):
    name: str
    description: Optional[str] = None
    expression: str  # SQL‑выражение, например "SUM(price * quantity)"
    data_source_id: str
    table: str

class MetricResponse(BaseModel):
    id: str
    name: str
    description: Optional[str]
    expression: str
    data_source_id: str
    table_name: str
    created_at: datetime

class DatasetCreate(BaseModel):
    name: str
    description: Optional[str] = None
    columns: List[str]  # список имён колонок
    data_source_id: str
    table: str
    filters: Optional[Dict[str, Any]] = None

class DatasetResponse(BaseModel):
    id: str
    name: str
    description: Optional[str]
    columns: List[str]
    data_source_id: str
    table_name: str
    filters: Optional[Dict[str, Any]]
    created_at: datetime

# Создание таблиц при запуске (если не существуют)
def init_db():
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute("""
        CREATE TABLE IF NOT EXISTS semantic_metrics (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            name TEXT NOT NULL,
            description TEXT,
            expression TEXT NOT NULL,
            data_source_id UUID NOT NULL,
            table_name TEXT NOT NULL,
            created_at TIMESTAMPTZ DEFAULT now()
        );
    """)
    cur.execute("""
        CREATE TABLE IF NOT EXISTS semantic_datasets (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            name TEXT NOT NULL,
            description TEXT,
            columns JSONB NOT NULL,
            data_source_id UUID NOT NULL,
            table_name TEXT NOT NULL,
            filters JSONB,
            created_at TIMESTAMPTZ DEFAULT now()
        );
    """)
    conn.commit()
    cur.close()
    conn.close()

init_db()

# --- Эндпоинты для метрик ---

@app.post("/metrics", response_model=MetricResponse)
def create_metric(metric: MetricCreate):
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute("""
        INSERT INTO semantic_metrics (name, description, expression, data_source_id, table_name)
        VALUES (%s, %s, %s, %s, %s) RETURNING id, name, description, expression, data_source_id, table_name, created_at
    """, (metric.name, metric.description, metric.expression, metric.data_source_id, metric.table))
    row = cur.fetchone()
    conn.commit()
    cur.close()
    conn.close()
    return MetricResponse(**row)

@app.get("/metrics", response_model=List[MetricResponse])
def list_metrics():
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute("SELECT id, name, description, expression, data_source_id, table_name, created_at FROM semantic_metrics ORDER BY created_at DESC")
    rows = cur.fetchall()
    cur.close()
    conn.close()
    return [MetricResponse(**r) for r in rows]

@app.get("/metrics/{metric_id}", response_model=MetricResponse)
def get_metric(metric_id: str):
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute("SELECT id, name, description, expression, data_source_id, table_name, created_at FROM semantic_metrics WHERE id = %s", (metric_id,))
    row = cur.fetchone()
    cur.close()
    conn.close()
    if not row:
        raise HTTPException(404, "Metric not found")
    return MetricResponse(**row)

@app.delete("/metrics/{metric_id}")
def delete_metric(metric_id: str):
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute("DELETE FROM semantic_metrics WHERE id = %s", (metric_id,))
    if cur.rowcount == 0:
        raise HTTPException(404, "Metric not found")
    conn.commit()
    cur.close()
    conn.close()
    return {"status": "deleted"}

# --- Эндпоинты для датасетов (аналогично) ---

@app.post("/datasets", response_model=DatasetResponse)
def create_dataset(dataset: DatasetCreate):
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute("""
        INSERT INTO semantic_datasets (name, description, columns, data_source_id, table_name, filters)
        VALUES (%s, %s, %s, %s, %s, %s) RETURNING id, name, description, columns, data_source_id, table_name, filters, created_at
    """, (dataset.name, dataset.description, json.dumps(dataset.columns), dataset.data_source_id, dataset.table, json.dumps(dataset.filters) if dataset.filters else None))
    row = cur.fetchone()
    conn.commit()
    cur.close()
    conn.close()
    return DatasetResponse(**row)

@app.get("/datasets", response_model=List[DatasetResponse])
def list_datasets():
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute("SELECT id, name, description, columns, data_source_id, table_name, filters, created_at FROM semantic_datasets ORDER BY created_at DESC")
    rows = cur.fetchall()
    cur.close()
    conn.close()
    return [DatasetResponse(**r) for r in rows]

@app.get("/datasets/{dataset_id}", response_model=DatasetResponse)
def get_dataset(dataset_id: str):
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute("SELECT id, name, description, columns, data_source_id, table_name, filters, created_at FROM semantic_datasets WHERE id = %s", (dataset_id,))
    row = cur.fetchone()
    cur.close()
    conn.close()
    if not row:
        raise HTTPException(404, "Dataset not found")
    return DatasetResponse(**row)

@app.delete("/datasets/{dataset_id}")
def delete_dataset(dataset_id: str):
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute("DELETE FROM semantic_datasets WHERE id = %s", (dataset_id,))
    if cur.rowcount == 0:
        raise HTTPException(404, "Dataset not found")
    conn.commit()
    cur.close()
    conn.close()
    return {"status": "deleted"}
