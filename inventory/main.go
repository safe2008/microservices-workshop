package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4/pgxpool"
)

func main() {
	log.Println("Starting the service")

	db := setupDB()

	r := createRouter(db)
	srv := &http.Server{
		Addr:         "0.0.0.0:8081",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}

	go func() {
		log.Println("The service is ready to listen and serve.")
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	wait := 15 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	err := srv.Shutdown(ctx)
	if err != nil {
		log.Println("error occur while shutting down", err)
	}

	log.Println("shutting down")
	os.Exit(0)
}

func setupDB() *pgxpool.Pool {

	config, err := pgxpool.ParseConfig("postgres://root:password@localhost:26257/defaultdb?sslmode=disable")
	if err != nil {
		log.Fatal("error configuring the database: ", err)
	}

	conn, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		log.Fatal("error connecting to the database: ", err)
	}

	_, err = conn.Exec(context.Background(),
		`CREATE TABLE IF NOT EXISTS products (
			id SERIAL PRIMARY KEY,
			name VARCHAR (255) NOT NULL,
			amount INT NULL,
			price INT NULL
			)`)
	if err != nil {
		log.Fatal("error creating the table: ", err)
	}

	_, err = conn.Exec(context.Background(),
		`CREATE TABLE IF NOT EXISTS carts (
			id SERIAL PRIMARY KEY,
			user_id INT NOT NULL
			)`)
	if err != nil {
		log.Fatal("error creating the table: ", err)
	}

	_, err = conn.Exec(context.Background(),
		`CREATE TABLE IF NOT EXISTS cart_products (
			cart_id INT NOT NULL,
			product_id INT NOT NULL,
			amount INT NULL,
			CONSTRAINT cart_products_pk PRIMARY KEY (cart_id, product_id)
			)`)
	if err != nil {
		log.Fatal("error creating the table: ", err)
	}

	return conn
}

func createRouter(db *pgxpool.Pool) *mux.Router {

	r := mux.NewRouter()

	product := ProductHandler{db}
	r.HandleFunc("/products/recommendations", product.GetRecommendations).Methods(http.MethodGet)
	r.HandleFunc("/products", product.ListProduct).Methods(http.MethodGet)
	r.HandleFunc("/product", product.AddProduct).Methods(http.MethodPost)
	r.HandleFunc("/product/{id}", product.GetProductByID).Methods(http.MethodGet)
	r.HandleFunc("/product/{id}", product.UpdateProduct).Methods(http.MethodPut)

	cart := CartHandler{db}
	r.HandleFunc("/cart", cart.Create).Methods(http.MethodPost)
	r.HandleFunc("/cart/{cartId}", cart.Get).Methods(http.MethodGet)
	r.HandleFunc("/cart/{cartId}/products", cart.RemoveAllProduct).Methods(http.MethodDelete)
	r.HandleFunc("/cart/{cartId}/product/{productId}", cart.AddProduct).Methods(http.MethodPost)
	r.HandleFunc("/cart/{cartId}/product/{productId}", cart.RemoveProduct).Methods(http.MethodDelete)

	return r
}