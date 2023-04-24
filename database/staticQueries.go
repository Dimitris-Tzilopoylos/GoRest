package database

import (
	"fmt"
	"strings"
)

var EXCLUDED_SCHEMAS = []string{
	"'pg_toast'",
	"'pg_catalog'",
	"'information_schema'",
	"'hdb_catalog'",
	"'hdb_views'",
	"'audit'",
	"'public'",
}

var GET_DATABASES string = fmt.Sprintf(`SELECT schema_name FROM information_schema.schemata WHERE schema_name NOT IN (%s) ORDER BY schema_name;`, strings.Join(EXCLUDED_SCHEMAS, ","))

const GET_DATABASE_TABLES = `SELECT table_name FROM information_schema.tables WHERE table_schema = $1 ORDER BY table_name;`
const GET_DATABASE_TABLE_COLUMN = `SELECT column_name,data_type,character_maximum_length,
CASE WHEN is_nullable = 'NO' THEN false ELSE true END,CASE WHEN column_default IS NULL THEN NULL ELSE column_default::text END
 FROM information_schema.columns WHERE table_schema = $1 AND table_name   = $2 ORDER BY ordinal_position;`
const GET_DATABASE_TABLE_INDEXES = `SELECT tc.constraint_name,  tc.table_name, kcu.column_name, ccu.table_name AS referer_table_name, 
    ccu.column_name AS referer_column_name,
	tc.constraint_type
    FROM information_schema.table_constraints AS tc 
    JOIN information_schema.key_column_usage AS kcu
        ON tc.constraint_name = kcu.constraint_name
        AND tc.table_schema = kcu.table_schema
    JOIN information_schema.constraint_column_usage AS ccu
        ON ccu.constraint_name = tc.constraint_name
        AND ccu.table_schema = tc.table_schema 
    WHERE  tc.table_schema = $1 AND tc.table_name = $2 
    GROUP BY tc.constraint_name,tc.constraint_type,tc.table_schema,tc.table_name,kcu.column_name,ccu.table_name,ccu.table_schema,ccu.column_name;`
const GET_ENGINE_RELATIONS = `SELECT relations.id,relations.alias,relations.db,relations.from_table,relations.from_column,relations.to_table,
       relations.to_column,relations.relation FROM root_engine.relations;`
const GET_GLOBAL_AUTH_CONFIG = `SELECT id,created_at,db,tbl,auth_config FROM root_engine.engine_auth_provider ORDER BY created_at ASC;`
