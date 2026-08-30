import psycopg2
dsn = 'postgresql://postgres.lzyxlztiqhfmdcnvwzan:Laziz9223361@aws-1-eu-west-1.pooler.supabase.com:5432/postgres'
conn = psycopg2.connect(dsn)
cur = conn.cursor()
# Попытка удалить constraint по имени
try:
    cur.execute("ALTER TABLE dashboard_widgets DROP CONSTRAINT IF EXISTS chart_type_check;")
    conn.commit()
    print("✅ Dropped chart_type_check")
except Exception as e:
    print(f"Error: {e}")
    # Если не получилось, попробуем найти все ограничения и удалить те, что связаны с chart_type
    cur.execute("""
        SELECT conname 
        FROM pg_constraint 
        WHERE conrelid = 'dashboard_widgets'::regclass 
        AND contype = 'c'
        AND pg_get_constraintdef(oid) LIKE '%chart_type%'
    """)
    rows = cur.fetchall()
    for row in rows:
        conname = row[0]
        print(f"Dropping found constraint: {conname}")
        cur.execute(f"ALTER TABLE dashboard_widgets DROP CONSTRAINT {conname};")
        conn.commit()
        print(f"✅ Dropped {conname}")
conn.close()
