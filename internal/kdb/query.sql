-- name: CreatePrincipal :one
INSERT INTO principals (
    primary_name,
    instance,
    realm,
    key_bytes,
    kvno
) VALUES (
    ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetPrincipal :one
SELECT key_bytes, kvno
FROM principals
WHERE primary_name = ? AND instance = ? AND realm = ?
LIMIT 1;

-- name: ListPrincipals :many
SELECT primary_name, instance, realm
FROM principals
ORDER BY primary_name, instance;
