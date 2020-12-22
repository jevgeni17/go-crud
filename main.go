package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"os"  
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" 
)

var db *sql.DB
var tpl *template.Template

type Customer struct {
	ID        int
	FirstName string
	LastName  string
	BirthDate string
	Gender    string
	Email     string
	Address   string
}

type SearchData struct {
	Customers []Customer
	SearchParameter string
}

func init() {
	// load .env file
	err := godotenv.Load(".env")
	db, err = sql.Open("postgres", os.Getenv("POSTGRES_URL"))
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	fmt.Println("Successfully connected!")

	tpl = template.Must(template.ParseGlob("templates/*.html"))
}

func main() {

	http.HandleFunc("/", index)
	http.HandleFunc("/customers", showCustomers)
	http.HandleFunc("/editcustomer", editCustomer)
	http.HandleFunc("/search", searchCustomer)
	http.HandleFunc("/editcustomeraction", editCustomerAction)
	http.HandleFunc("/createcustomer", createCustomerForm)
	http.HandleFunc("/createcustomeraction", createCustomerAction)
	http.ListenAndServe(":8080", nil)

}

func searchCustomer(w http.ResponseWriter, r *http.Request) {

	// search input value
	searchString := r.FormValue("param")

	if searchString == "" {
		http.Error(w, http.StatusText(400), http.StatusBadRequest)
		return
	}

	words := strings.Fields(searchString)

	rows, err := db.Query("SELECT * FROM Customers WHERE first_name=$1 OR last_name=$2", words[0], words[1])
	if err != nil {
		panic(err)// if not found
	}
	defer rows.Close()

	customers := make([]Customer, 0)

	for rows.Next() {
		customer := Customer{}
		err := rows.Scan(&customer.ID, &customer.FirstName, &customer.LastName, &customer.BirthDate, &customer.Gender, &customer.Email, &customer.Address)
		customer.BirthDate = customer.BirthDate[0:10]
		if err != nil {
			panic(err)
		}
		customers = append(customers, customer)
	}
	switch {
	case err == sql.ErrNoRows:
		http.NotFound(w, r)
		return
	case err != nil:
		http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		return
	}
	
	data := &SearchData{
		Customers:       customers,
		SearchParameter: searchString,
	}

	tpl.ExecuteTemplate(w, "search.html", data)
}

func editCustomerAction(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
		return
	}

	// form values
	customer := Customer{}
	id := r.FormValue("ID")
	if id == "" {
		http.Error(w, http.StatusText(400), http.StatusBadRequest)
		return
	}
	customer.ID, _ = strconv.Atoi(id)
	customer.FirstName = r.FormValue("firstName")
	customer.LastName = r.FormValue("lastName")
	customer.BirthDate = r.FormValue("birthDate")
	customer.Gender = r.FormValue("gender")
	customer.Email = r.FormValue("email")
	customer.Address = r.FormValue("address")

	// validate 
	if validateForm(&customer.FirstName, &customer.LastName, &customer.Gender, &customer.Address, &customer.Email) {
		http.Error(w, http.StatusText(400), http.StatusBadRequest)
	}

	txn, err1 := db.Begin()
	if err1 != nil {
		return
	}

	defer func() {
		// Rollback if fail
		if r := recover(); r != nil {
			txn.Rollback()
		}
	}()

	// insert values
	_, err := txn.Exec("UPDATE Customers SET first_name=$1, last_name=$2, birth_date=$3, gender=$4, email=$5, address=$6 WHERE ID=$7",
		customer.FirstName, customer.LastName, customer.BirthDate, customer.Gender, customer.Email, customer.Address, customer.ID)
	if err != nil {
		txn.Rollback()
		http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		return
	}

	// Commit
	err1 = txn.Commit()
	if err1 != nil {
		log.Fatal(err1)
	}

	// confirm insertion
	http.Redirect(w, r, "customers", http.StatusSeeOther)
}

func createCustomerAction(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
		return
	}

	// form values
	customer := Customer{}
	customer.FirstName = r.FormValue("firstName")
	customer.LastName = r.FormValue("lastName")
	customer.BirthDate = r.FormValue("birthDate")
	customer.Gender = r.FormValue("gender")
	customer.Email = r.FormValue("email")
	customer.Address = r.FormValue("address")

	// validate 
	if validateForm(&customer.FirstName, &customer.LastName, &customer.Gender, &customer.Address, &customer.Email) {
		http.Error(w, http.StatusText(400), http.StatusBadRequest)
		return
	}

	// insert values
	_, err := db.Exec("INSERT INTO Customers(first_name, last_name, birth_date, gender, email, address)  VALUES ($1, $2, $3, $4, $5, $6)",
		customer.FirstName, customer.LastName, customer.BirthDate, customer.Gender, customer.Email, customer.Address)

	if err != nil {
		http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		return
	}

	// confirm insertion
	http.Redirect(w, r, "customers", http.StatusSeeOther)
}

func showCustomers(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.Query("SELECT * FROM Customers")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	customers := make([]Customer, 0)

	for rows.Next() {
		customer := Customer{}
		err := rows.Scan(&customer.ID, &customer.FirstName, &customer.LastName, &customer.BirthDate, &customer.Gender, &customer.Email, &customer.Address)
		customer.BirthDate = customer.BirthDate[0:10]
		if err != nil {
			panic(err)
		}
		customers = append(customers, customer)
	}
	switch {
	case err == sql.ErrNoRows:
		http.NotFound(w, r)
		return
	case err != nil:
		http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		return
	}

	tpl.ExecuteTemplate(w, "all.html", customers)

}

func editCustomer(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
		return
	}

	id := r.FormValue("id")
	if id == "" {
		http.Error(w, http.StatusText(400), http.StatusBadRequest)
		return
	}

	row := db.QueryRow("SELECT * FROM Customers WHERE id = $1", id)

	customer := Customer{}
	err := row.Scan(&customer.ID, &customer.FirstName, &customer.LastName, &customer.BirthDate, &customer.Gender, &customer.Email, &customer.Address)
	customer.BirthDate = customer.BirthDate[0:10]
	switch {
	case err == sql.ErrNoRows:
		http.NotFound(w, r)
		return
	case err != nil:
		http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		return
	}
	tpl.ExecuteTemplate(w, "update.html", customer)

}

func createCustomerForm(w http.ResponseWriter, r *http.Request) {
	tpl.ExecuteTemplate(w, "create.html", nil)
}

func index(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "customers", http.StatusSeeOther)
}

//Email regex
var re = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

func validateForm(firstName, lastName, gender, address, email *string) bool {
	if (*firstName == "" || len(*firstName) > 100) ||
		(*lastName == "" || len(*lastName) > 100) ||
		(*gender != "Male" && *gender != "Female") ||
		(*address == "" || len(*address) > 200) ||
		(!re.MatchString(*email)) {
		return true
	} else {
		return false
	}
}
