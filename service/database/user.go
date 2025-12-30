package database

import (
	"database/sql"
	"errors"
)

func (db *appdbimpl) CreateUser(id, name string) error {
	_, err := db.c.Exec("INSERT INTO users (id, name, display_name) VALUES (?, ?, ?)", id, name, nil)
	return err
}

func (db *appdbimpl) GetUserByID(id string) (*User, error) {
	var u User
	err := db.c.QueryRow("SELECT id, name, display_name, photo_url FROM users WHERE id = ?", id).Scan(&u.ID, &u.Name, &u.DisplayName, &u.PhotoURL)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *appdbimpl) GetUserByName(name string) (*User, error) {
	var u User
	err := db.c.QueryRow("SELECT id, name, display_name, photo_url FROM users WHERE name = ?", name).Scan(&u.ID, &u.Name, &u.DisplayName, &u.PhotoURL)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *appdbimpl) UpdateUsername(userID, newName string) error {
	_, err := db.c.Exec("UPDATE users SET name = ? WHERE id = ?", newName, userID)
	return err
}

func (db *appdbimpl) UpdateUserPhoto(userID string, photoURL *string) error {
	_, err := db.c.Exec("UPDATE users SET photo_url = ? WHERE id = ?", photoURL, userID)
	return err
}

func (db *appdbimpl) SearchUsers(query string) ([]User, error) {
	rows, err := db.c.Query("SELECT id, name, display_name, photo_url FROM users WHERE name LIKE ?", "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.DisplayName, &u.PhotoURL); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (db *appdbimpl) GetAllUsers() ([]User, error) {
	rows, err := db.c.Query("SELECT id, name, display_name, photo_url FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.DisplayName, &u.PhotoURL); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (db *appdbimpl) GetUsersPaginated(limit, offset int) ([]User, error) {
	rows, err := db.c.Query("SELECT id, name, display_name, photo_url FROM users LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.DisplayName, &u.PhotoURL); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
