#!/bin/bash
docker run --rm -it python:3.12-slim bash -c "
pip install psycopg2-binary && python3 << 'PYEOF'
import psycopg2
dsn = 'postgresql://postgres.lzyxlztiqhfmdcnvwzan:Laziz9223361@aws-1-eu-west-1.pooler.supabase.com:5432/postgres'
try:
    conn = psycopg2.connect(dsn)
    cur = conn.cursor()

    # 1. Временной ряд (продажи по месяцам)
    cur.execute('''
        CREATE TABLE IF NOT EXISTS sales_monthly (
            year INT,
            month INT,
            product VARCHAR(50),
            actual DECIMAL(10,2),
            forecast DECIMAL(10,2)
        );
        TRUNCATE sales_monthly;
        INSERT INTO sales_monthly (year, month, product, actual, forecast) VALUES
        (2024, 1, 'Laptop', 1200, 1100),
        (2024, 2, 'Laptop', 1350, 1250),
        (2024, 3, 'Laptop', 1500, 1400),
        (2024, 4, 'Laptop', 1400, 1450),
        (2024, 5, 'Laptop', 1600, 1500),
        (2024, 6, 'Laptop', 1700, 1600),
        (2024, 1, 'Phone', 800, 750),
        (2024, 2, 'Phone', 850, 800),
        (2024, 3, 'Phone', 900, 850),
        (2024, 4, 'Phone', 880, 870),
        (2024, 5, 'Phone', 950, 900),
        (2024, 6, 'Phone', 1000, 950),
        (2024, 1, 'Tablet', 500, 450),
        (2024, 2, 'Tablet', 520, 480),
        (2024, 3, 'Tablet', 550, 500),
        (2024, 4, 'Tablet', 530, 520),
        (2024, 5, 'Tablet', 600, 550),
        (2024, 6, 'Tablet', 650, 600);
    ''')

    # 2. Данные для радара (несколько метрик по продуктам)
    cur.execute('''
        CREATE TABLE IF NOT EXISTS kpi_radar (
            product VARCHAR(50),
            quality INT,
            price INT,
            support INT,
            features INT,
            delivery INT
        );
        TRUNCATE kpi_radar;
        INSERT INTO kpi_radar (product, quality, price, support, features, delivery) VALUES
        ('Laptop', 85, 70, 80, 90, 75),
        ('Phone', 80, 85, 75, 80, 85),
        ('Tablet', 75, 80, 70, 75, 80);
    ''')

    # 3. Воронка (конверсия по этапам)
    cur.execute('''
        CREATE TABLE IF NOT EXISTS funnel_data (
            stage VARCHAR(50),
            value INT
        );
        TRUNCATE funnel_data;
        INSERT INTO funnel_data (stage, value) VALUES
        ('Visitors', 10000),
        ('Signups', 6000),
        ('Trial', 3000),
        ('Paid', 1500),
        ('Retained', 750);
    ''')

    conn.commit()
    print('✅ Demo data created successfully!')
    conn.close()
except Exception as e:
    print(f'Error: {e}')
PYEOF
"
