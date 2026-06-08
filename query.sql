-- name: FindUser :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;


-- #TODO password for now is not hashed and salted
-- name: CreateUser :one
INSERT INTO users (
  username, password, email
) VALUES (
  $1, $2, $3
)
RETURNING *;
