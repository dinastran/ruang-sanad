package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/pressly/goose/v3"
	"golang.org/x/crypto/argon2"

	_ "github.com/mattn/go-sqlite3"
)

type seedUser struct {
	Name     string
	Email    string
	Role     string
	Password string
}

func main() {
	_ = godotenv.Load()

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/app.db"
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("gagal buka db: %v", err)
	}
	defer db.Close()

	// Run migrations first
	goose.SetBaseFS(nil)
	if err := goose.SetDialect("sqlite"); err != nil {
		log.Fatalf("gagal set dialect: %v", err)
	}
	if err := goose.Up(db, "./migrations"); err != nil {
		log.Fatalf("gagal migrasi: %v", err)
	}

	querier := queries.NewQuerier(db)

	defaultPassword := "rahasia123"

	users := []seedUser{
		{Name: "Super Admin", Email: "dinastran@gmail.com", Role: "super_admin", Password: defaultPassword},
		{Name: "CS Staff", Email: "cs@ruangsanad.com", Role: "cs", Password: defaultPassword},
		{Name: "Admin Kelas", Email: "adminkelas@ruangsanad.com", Role: "admin_kelas", Password: defaultPassword},
		{Name: "Keuangan Staff", Email: "keuangan@ruangsanad.com", Role: "keuangan", Password: defaultPassword},
	}

	for _, u := range users {
		existing, err := querier.GetUserByEmail(context.Background(), u.Email)
		if err == nil {
			// Update existing user
			querier.UpdateUserPassword(context.Background(), existing.ID, hashPassword(u.Password).String)
			querier.UpdateUserRole(context.Background(), queries.UpdateUserRoleParams{
				Role: u.Role, UpdatedAt: time.Now(), ID: existing.ID,
			})
			fmt.Printf("✓  %s (%s) — password: %s (updated)\n", u.Email, u.Role, defaultPassword)
			continue
		}
		user := &models.User{
			Email:         u.Email,
			Name:          u.Name,
			Password:      hashPassword(u.Password),
			Role:          models.UserRole(u.Role),
			EmailVerified: true,
		}
		if err := querier.CreateUser(context.Background(), user); err != nil {
			log.Fatalf("gagal buat user %s: %v", u.Email, err)
		}
		fmt.Printf("✓  %s (%s) — password: %s\n", u.Email, u.Role, defaultPassword)
	}

	fmt.Println("\n✅ Seed selesai. Silakan login dengan password: " + defaultPassword)
}

func hashPassword(password string) sql.NullString {
	salt := make([]byte, 16)
	for i := range salt {
		salt[i] = byte(i * 17)
	}
	hash := argon2.IDKey([]byte(password), salt, 1, 32*1024, 4, 32)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	encoded := fmt.Sprintf("$argon2id$v=19$m=32768,t=1,p=4$%s$%s", b64Salt, b64Hash)
	return sql.NullString{String: encoded, Valid: true}
}

func isDuplicate(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "UNIQUE constraint failed")
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
