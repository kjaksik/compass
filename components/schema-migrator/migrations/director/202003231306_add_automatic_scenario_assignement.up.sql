CREATE TABLE automatic_scenario_assignements (
    scenario VARCHAR(128),
    tenant_id UUID NOT NULL,
    key VARCHAR(256) NOT NULL,
    value JSONB);


CREATE INDEX ON automatic_scenario_assignements (tenant_id);

ALTER TABLE automatic_scenario_assignements
    ADD CONSTRAINT automatic_scenario_assignements_pk
    PRIMARY KEY (tenant_id, scenario);

ALTER TABLE automatic_scenario_assignements
    ADD CONSTRAINT automatic_scenario_assignements_tenant_constraint
    FOREIGN KEY (tenant_id)
    REFERENCES business_tenant_mappings(id)
    ON DELETE CASCADE;