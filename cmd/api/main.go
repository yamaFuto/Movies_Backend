package main

import (
	"backend/internal/repository"
	"backend/internal/repository/dbrepo"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

const port = 8080

type application struct {
	DSN string
	Domain string
	DB repository.DatabaseRepo
	auth Auth
	JWTSecret string
	JWTIssuer string
	JWTAudience string
	CookieDomain string
	APIKey string
}

func main() {
	//set application config
	var app application

	//read from command line
	//make a flag
	flag.StringVar(&app.DSN, "dsn", "host=localhost port=5432 user=postgres password=postgres dbname=movies sslmode=disable timezone=UTC connect_timeout=5", "Postgres connecion string")
	flag.StringVar(&app.JWTSecret, "jwt-secret", "verysecret", "signing secret")
	flag.StringVar(&app.JWTIssuer, "jwt-issuer", "example.com", "signing issuer")
	flag.StringVar(&app.JWTAudience, "jwt-audience", "example.com", "signing audience")
	flag.StringVar(&app.CookieDomain, "cookie-domain", "localhost", "signing secret")
	flag.StringVar(&app.JWTSecret,"domain", "jwt-secret", "domain")
	flag.StringVar(&app.APIKey, "api-key", "2710579fa35e478e139e06e27a2d51b6", "api key")
	//read flags
	flag.Parse()

	//connect to the database
	conn, err := app.connectToDB()
	if err != nil {
		log.Fatal(err)
	}

	app.DB = &dbrepo.PostgresDBRepo{DB: conn}
	defer app.DB.Connection().Close()

	app.auth = Auth{
		Issuer: app.JWTIssuer,
		Audience: app.JWTAudience,
		Secret: app.JWTSecret,
		TokenExpiry: time.Minute * 15,
		RefreshExpiry: time.Hour * 24,
		CookiePath: "/",
		CookieName: "refresh_token",
		CookieDomain: app.CookieDomain,
	}

	log.Println("Starting application on port", port)
	// //ListenAndServeのだいにひきすう（route設定)がdefaultの場合
	// http.HandleFunc("/", Hello)

	//star a web server
	//第二引数はrouterの設定
	err = http.ListenAndServe(fmt.Sprintf(":%d", port), app.routes())
	if err != nil {
		log.Fatal(err)
	}
}