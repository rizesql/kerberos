CREATE TABLE principals (
    id            INTEGER             PRIMARY KEY AUTOINCREMENT,
    primary_name  TEXT      NOT NULL  CHECK(length(primary_name) > 0),
    instance      TEXT      NOT NULL,
    realm         TEXT      NOT NULL  CHECK(length(realm) > 0),
    key_bytes     BLOB      NOT NULL  CHECK(length(key_bytes) > 0),
    kvno          INTEGER   NOT NULL  DEFAULT 1,
    created_at    DATETIME            DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(primary_name, instance, realm)
);

CREATE INDEX idx_principals_lookup ON principals(primary_name, instance, realm);
