package common

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ConnectPgDb conecta ao banco de dados PostgreSQL (banco principal)
func ConnectPgDb() *gorm.DB {
	host := os.Getenv("PG_HOST")
	port := os.Getenv("PG_PORT")
	user := os.Getenv("PG_USER")
	password := os.Getenv("PG_PASSWORD")
	dbname := os.Getenv("PG_DBNAME")
	sslmode := os.Getenv("PG_SSLMODE")
	certificate := os.Getenv("PG_CERTIFICATE")

	if host == "" || user == "" || dbname == "" {
		log.Println("PostgreSQL config incomplete (PG_HOST, PG_USER, PG_DBNAME required)")
		return nil
	}
	if port == "" {
		port = "5432"
	}
	if sslmode == "" {
		sslmode = "disable"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	if certificate != "" {
		dsn += fmt.Sprintf(" sslrootcert=%s", certificate)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Println("Error connecting to PostgreSQL: " + err.Error())
		return nil
	}
	log.Printf("Connected to PostgreSQL (%s@%s:%s/%s)", user, host, port, dbname)
	return db
}

// ConnectRedis conecta ao Redis para analytics
func ConnectRedis() *redis.Client {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	password := os.Getenv("REDIS_PASSWORD")

	dbNum := 0
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		if n, err := strconv.Atoi(dbStr); err == nil {
			dbNum = n
		}
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       dbNum,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Printf("Error connecting to Redis at %s: %v - analytics will be disabled", addr, err)
		return nil
	}

	log.Printf("Connected to Redis (%s, db=%d)", addr, dbNum)
	return client
}
