-- name: ListTodo :many
SELECT * FROM todo_koordinator
ORDER BY
    CASE status WHEN 'belum' THEN 1 WHEN 'proses' THEN 2 WHEN 'selesai' THEN 3 ELSE 4 END,
    deadline IS NULL, deadline ASC, created_at DESC;

-- name: GetTodo :one
SELECT * FROM todo_koordinator WHERE id = ?;

-- name: CreateTodo :one
INSERT INTO todo_koordinator (judul, teknis, kebutuhan, deadline, pic, status, recurring, link_pendukung, catatan)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id;

-- name: UpdateTodo :exec
UPDATE todo_koordinator SET judul = ?, teknis = ?, kebutuhan = ?, deadline = ?, pic = ?, status = ?, recurring = ?, link_pendukung = ?, catatan = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: SetTodoStatus :exec
UPDATE todo_koordinator SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: DeleteTodo :exec
DELETE FROM todo_koordinator WHERE id = ?;
