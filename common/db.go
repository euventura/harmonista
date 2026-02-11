package common

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/valkey-io/valkey-go"
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

// ConnectValkey conecta ao Valkey para analytics
func ConnectValkey() valkey.Client {
	addr := os.Getenv("VALKEY_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	username := os.Getenv("VALKEY_USERNAME")
	password := os.Getenv("VALKEY_PASSWORD")
	useTLS := os.Getenv("VALKEY_TLS") == "true"

	opts := valkey.ClientOption{
		InitAddress: []string{addr},
		Username:    username,
		Password:    password,
	}

	if dbStr := os.Getenv("VALKEY_DB"); dbStr != "" {
		if n, err := strconv.Atoi(dbStr); err == nil {
			opts.SelectDB = n
		}
	}

	if useTLS {
		opts.TLSConfig = &tls.Config{}
	}

	client, err := valkey.NewClient(opts)
	if err != nil {
		log.Printf("Error connecting to Valkey at %s: %v - analytics will be disabled", addr, err)
		return nil
	}

	if err := client.Do(context.Background(), client.B().Ping().Build()).Error(); err != nil {
		log.Printf("Error pinging Valkey at %s: %v - analytics will be disabled", addr, err)
		client.Close()
		return nil
	}

	log.Printf("Connected to Valkey (%s)", addr)
	return client
}
