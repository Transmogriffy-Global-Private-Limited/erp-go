DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_roles
        WHERE rolname = 'erp_app'
    ) THEN
        REVOKE EXECUTE ON FUNCTION core.current_tenant_id() FROM erp_app;

        REVOKE SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA
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
        FROM erp_app;

        REVOKE USAGE, SELECT ON ALL SEQUENCES IN SCHEMA
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
        FROM erp_app;

        REVOKE USAGE ON SCHEMA
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
        FROM erp_app;

        EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM erp_app', current_database());
    END IF;
END;
$$;
