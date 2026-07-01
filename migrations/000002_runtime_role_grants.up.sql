DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_roles
        WHERE rolname = 'erp_app'
    ) THEN
        RAISE EXCEPTION 'Required role erp_app does not exist. Create it before applying this migration.';
    END IF;
END;
$$;

DO $$
BEGIN
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO erp_app', current_database());
END;
$$;

GRANT USAGE ON SCHEMA
    control,
    core,
    documents,
    audit,
    workflow,
    inventory,
    sales,
    purchase,
    accounting,
    reports
TO erp_app;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA
    control,
    core,
    documents,
    audit,
    workflow,
    inventory,
    sales,
    purchase,
    accounting,
    reports
TO erp_app;

GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA
    control,
    core,
    documents,
    audit,
    workflow,
    inventory,
    sales,
    purchase,
    accounting,
    reports
TO erp_app;

GRANT EXECUTE ON FUNCTION core.current_tenant_id() TO erp_app;

ALTER DEFAULT PRIVILEGES IN SCHEMA
    control,
    core,
    documents,
    audit,
    workflow,
    inventory,
    sales,
    purchase,
    accounting,
    reports
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO erp_app;

ALTER DEFAULT PRIVILEGES IN SCHEMA
    control,
    core,
    documents,
    audit,
    workflow,
    inventory,
    sales,
    purchase,
    accounting,
    reports
GRANT USAGE, SELECT ON SEQUENCES TO erp_app;
