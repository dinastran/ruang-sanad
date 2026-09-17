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

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no CGO → easy cross-compile)
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

	db, err := sql.Open("sqlite", dbPath)
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
		{Name: "Admin", Email: "admin@ruangsanad.com", Role: "admin", Password: defaultPassword},
		{Name: "CS Staff", Email: "cs@ruangsanad.com", Role: "cs", Password: defaultPassword},
		{Name: "Admin Kelas", Email: "adminkelas@ruangsanad.com", Role: "admin_kelas", Password: defaultPassword},
		{Name: "Keuangan Staff", Email: "keuangan@ruangsanad.com", Role: "keuangan", Password: defaultPassword},
		{Name: "User Biasa", Email: "user@ruangsanad.com", Role: "user", Password: defaultPassword},
		{Name: "Guru Test", Email: "guru@ruangsanad.com", Role: "guru", Password: defaultPassword},
		{Name: "Koordinator Guru", Email: "koordinator@ruangsanad.com", Role: "koordinator_guru", Password: defaultPassword},
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

	// Create guru master records and link to guru/koordinator user accounts
	guruUser, err := querier.GetUserByEmail(context.Background(), "guru@ruangsanad.com")
	if err == nil {
		// Check if guru record already exists for this user
		_, err := querier.GuruGetByUserID(context.Background(), sql.NullInt64{Int64: guruUser.ID, Valid: true})
		if err != nil {
			// Create guru master record
			if err := querier.CreateGuru(context.Background(), queries.CreateGuruParams{
				Nama:         "Guru Test",
				JenisKelamin: "Laki-laki",
				IsAktif:      1,
			}); err != nil {
				log.Fatalf("gagal buat guru master: %v", err)
			}
			// Get the newly created guru (first guru with nama Guru Test)
			gurus, _ := querier.GuruListAll(context.Background())
			var guruID int64
			for _, g := range gurus {
				if g.Nama == "Guru Test" && !g.UserID.Valid {
					guruID = g.ID
					break
				}
			}
			if guruID > 0 {
				if err := querier.UpdateGuruUserID(context.Background(), queries.UpdateGuruUserIDParams{
					UserID: sql.NullInt64{Int64: guruUser.ID, Valid: true},
					ID:     guruID,
				}); err != nil {
					log.Fatalf("gagal link guru ke user: %v", err)
				}
				fmt.Printf("✓  Guru Test linked to user (guru_id=%d, user_id=%d)\n", guruID, guruUser.ID)
			}
		} else {
			fmt.Printf("✓  Guru Test already linked\n")
		}
	}

	koordinatorUser, err := querier.GetUserByEmail(context.Background(), "koordinator@ruangsanad.com")
	if err == nil {
		_, err := querier.GuruGetByUserID(context.Background(), sql.NullInt64{Int64: koordinatorUser.ID, Valid: true})
		if err != nil {
			if err := querier.CreateGuru(context.Background(), queries.CreateGuruParams{
				Nama:         "Koordinator Guru",
				JenisKelamin: "Laki-laki",
				IsAktif:      1,
			}); err != nil {
				log.Fatalf("gagal buat guru koordinator: %v", err)
			}
			gurus, _ := querier.GuruListAll(context.Background())
			var guruID int64
			for _, g := range gurus {
				if g.Nama == "Koordinator Guru" && !g.UserID.Valid {
					guruID = g.ID
					break
				}
			}
			if guruID > 0 {
				if err := querier.UpdateGuruUserID(context.Background(), queries.UpdateGuruUserIDParams{
					UserID: sql.NullInt64{Int64: koordinatorUser.ID, Valid: true},
					ID:     guruID,
				}); err != nil {
					log.Fatalf("gagal link koordinator ke user: %v", err)
				}
				fmt.Printf("✓  Koordinator Guru linked to user (guru_id=%d, user_id=%d)\n", guruID, koordinatorUser.ID)
			}
		} else {
			fmt.Printf("✓  Koordinator Guru already linked\n")
		}
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
