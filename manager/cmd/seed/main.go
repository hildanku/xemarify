// cmd/seed/main.go — seeds the database with one default user per role.
// Run from the manager directory:
//
//	go run ./cmd/seed
//
// The seeder is idempotent: existing emails and user_id records are skipped
// via ON CONFLICT … DO NOTHING, so it is safe to run multiple times.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/config"
	infraLogger "github.com/hildanku/xemarify/internal/infrastructure/logger"
	"golang.org/x/crypto/bcrypt"
)

// seedUser describes one account to be seeded.
type seedUser struct {
	Username string
	Email    string
	Role     string
	Password string
}

var seeds = []seedUser{
	{
		Username: "manager",
		Email:    "manager@xemarify.local",
		Role:     "MANAGER",
		Password: "Manager@123",
	},
	{
		Username: "analyst",
		Email:    "analyst@xemarify.local",
		Role:     "ANALYST",
		Password: "Analyst@123",
	},
	{
		Username: "viewer",
		Email:    "viewer@xemarify.local",
		Role:     "VIEWER",
		Password: "Viewer@123",
	},
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := infraLogger.New(cfg.LogLevel)
	log.Info("starting user seeder")

	db, err := config.NewDatabasePool(cfg.Database, log)
	if err != nil {
		log.WithError(err).Fatal("failed to connect to database")
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, s := range seeds {
		userID := uuid.New()
		now := time.Now().UTC()

		const insertUser = `
			INSERT INTO users (id, username, email, role, created_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (email) DO NOTHING
		`
		tag, err := db.Exec(ctx, insertUser, userID, s.Username, s.Email, s.Role, now)
		if err != nil {
			log.WithError(err).Errorf("failed to insert user %s", s.Email)
			os.Exit(1)
		}

		if tag.RowsAffected() == 0 {
			log.Infof("skip: user %s already exists", s.Email)
			continue
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(s.Password), bcrypt.DefaultCost)
		if err != nil {
			log.WithError(err).Errorf("failed to hash password for %s", s.Email)
			os.Exit(1)
		}

		const insertAuth = `
			INSERT INTO authentications (id, user_id, password_hash, created_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id) DO NOTHING
		`
		_, err = db.Exec(ctx, insertAuth, uuid.New(), userID, string(hash), now)
		if err != nil {
			log.WithError(err).Errorf("failed to insert authentication for %s", s.Email)
			os.Exit(1)
		}

		log.Infof("seeded: %-10s  email=%-30s  role=%s", s.Username, s.Email, s.Role)
	}

	log.Info("seeder finished")
}
